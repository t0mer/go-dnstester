package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tomerklein/dnstester/internal/handler"
)

type Server struct {
	port            int
	cfgHandler      *handler.ConfigHandler
	testHandler     *handler.TestHandler
	historyHandler  *handler.HistoryHandler
	scheduleHandler *handler.ScheduleHandler
	updateHandler   *handler.UpdateHandler
	ui              fs.FS
	httpSrv         *http.Server
}

func New(
	port int,
	cfg *handler.ConfigHandler,
	test *handler.TestHandler,
	history *handler.HistoryHandler,
	schedule *handler.ScheduleHandler,
	update *handler.UpdateHandler,
	ui fs.FS,
) *Server {
	return &Server{
		port:            port,
		cfgHandler:      cfg,
		testHandler:     test,
		historyHandler:  history,
		scheduleHandler: schedule,
		updateHandler:   update,
		ui:              ui,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()

	// Tests
	mux.HandleFunc("GET /api/test/run", s.testHandler.Run)
	mux.HandleFunc("POST /api/test/run", s.testHandler.Run)
	mux.HandleFunc("GET /api/test/latest", s.testHandler.Latest)

	// History
	mux.HandleFunc("GET /api/history", s.historyHandler.List)
	mux.HandleFunc("GET /api/history/{id}", s.historyHandler.Get)
	mux.HandleFunc("GET /api/compare", s.historyHandler.Compare)

	// Settings (canonical path; also kept under /api/config for backward compat)
	mux.HandleFunc("GET /api/settings", s.cfgHandler.Get)
	mux.HandleFunc("PUT /api/settings", s.cfgHandler.Update)

	// Config (backward-compat aliases + backup/restore/export/import)
	mux.HandleFunc("GET /api/config", s.cfgHandler.Get)
	mux.HandleFunc("PUT /api/config", s.cfgHandler.Update)
	mux.HandleFunc("POST /api/config/backup", s.cfgHandler.Backup)
	mux.HandleFunc("POST /api/config/restore", s.cfgHandler.Restore)
	mux.HandleFunc("GET /api/config/export", s.cfgHandler.Export)
	mux.HandleFunc("POST /api/config/import", s.cfgHandler.Import)

	// Schedules
	mux.HandleFunc("GET /api/schedules", s.scheduleHandler.List)
	mux.HandleFunc("POST /api/schedules", s.scheduleHandler.Create)
	mux.HandleFunc("PUT /api/schedules/{id}", s.scheduleHandler.Update)
	mux.HandleFunc("DELETE /api/schedules/{id}", s.scheduleHandler.Delete)

	// Update
	mux.HandleFunc("GET /api/version", s.updateHandler.Version)
	mux.HandleFunc("GET /api/update/check", s.updateHandler.Check)
	mux.HandleFunc("POST /api/update/apply", s.updateHandler.Apply)

	// Prometheus metrics
	mux.Handle("GET /metrics", promhttp.Handler())

	// API documentation
	mux.HandleFunc("GET /api/openapi.json", handler.ServeSpec)
	mux.HandleFunc("GET /api/docs", handler.ServeSwaggerUI)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.Handle("/", spaHandler(s.ui))

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	fmt.Printf("DNS Tester listening on http://%s\n", addr)
	fmt.Printf("API docs:           http://%s/api/docs\n", addr)
	fmt.Printf("Prometheus metrics: http://%s/metrics\n", addr)
	s.httpSrv = &http.Server{Addr: addr, Handler: mux}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(timeout time.Duration) error {
	if s.httpSrv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

func spaHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := assets.Open(path)
		if err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
