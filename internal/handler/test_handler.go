package handler

import (
	"log"
	"net/http"

	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/service"
	"github.com/tomerklein/dnstester/internal/store"
)


type TestHandler struct {
	cfgSvc  *config.Service
	testSvc *service.TestService
	runs    *store.RunStore
}

func NewTestHandler(cfgSvc *config.Service, testSvc *service.TestService, runs *store.RunStore) *TestHandler {
	return &TestHandler{cfgSvc: cfgSvc, testSvc: testSvc, runs: runs}
}

func (h *TestHandler) Run(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	run := h.testSvc.Run(cfg.Servers, cfg.FQDNs)
	if err := h.runs.Save(run); err != nil {
		log.Printf("save run %s: %v", run.ID, err)
	}
	writeJSON(w, run)
}

func (h *TestHandler) Latest(w http.ResponseWriter, r *http.Request) {
	// Fast path: in-memory result from the current process.
	if run := h.testSvc.Latest(); run != nil {
		writeJSON(w, run)
		return
	}
	// Fallback: most recent run stored in the database (survives restarts).
	summaries, err := h.runs.List(store.ListFilter{Limit: 1, NoTimeFilter: true})
	if err != nil || len(summaries) == 0 {
		http.Error(w, "no results yet", http.StatusNotFound)
		return
	}
	run, err := h.runs.Get(summaries[0].ID)
	if err != nil || run == nil {
		http.Error(w, "no results yet", http.StatusNotFound)
		return
	}
	writeJSON(w, run)
}
