package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/tomerklein/dnstester/internal/handler"
)

type Server struct {
	port           int
	cfgHandler     *handler.ConfigHandler
	testHandler    *handler.TestHandler
	historyHandler *handler.HistoryHandler
	ui             fs.FS
	httpSrv        *http.Server
}

func New(port int, cfg *handler.ConfigHandler, test *handler.TestHandler, history *handler.HistoryHandler, ui fs.FS) *Server {
	return &Server{port: port, cfgHandler: cfg, testHandler: test, historyHandler: history, ui: ui}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", s.cfgHandler.Get)
	mux.HandleFunc("PUT /api/config", s.cfgHandler.Update)
	mux.HandleFunc("POST /api/config/backup", s.cfgHandler.Backup)
	mux.HandleFunc("POST /api/config/restore", s.cfgHandler.Restore)
	mux.HandleFunc("GET /api/config/export", s.cfgHandler.Export)
	mux.HandleFunc("POST /api/config/import", s.cfgHandler.Import)
	mux.HandleFunc("POST /api/test/run", s.testHandler.Run)
	mux.HandleFunc("GET /api/test/latest", s.testHandler.Latest)
	mux.HandleFunc("GET /api/history", s.historyHandler.List)
	mux.HandleFunc("GET /api/history/{id}", s.historyHandler.Get)
	mux.HandleFunc("GET /api/compare", s.historyHandler.Compare)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.Handle("/", spaHandler(s.ui))

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	fmt.Printf("DNS Tester listening on http://%s\n", addr)
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
