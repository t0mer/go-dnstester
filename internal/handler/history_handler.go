package handler

import (
	"net/http"
	"strconv"

	"github.com/tomerklein/dnstester/internal/store"
)

type HistoryHandler struct {
	runs *store.RunStore
}

func NewHistoryHandler(runs *store.RunStore) *HistoryHandler {
	return &HistoryHandler{runs: runs}
}

func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	scheduledOnly := r.URL.Query().Get("scheduled") == "true"

	runs, err := h.runs.List(limit, scheduledOnly)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []store.RunSummary{}
	}
	writeJSON(w, runs)
}

func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.runs.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if run == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, run)
}

func (h *HistoryHandler) Compare(w http.ResponseWriter, r *http.Request) {
	idA := r.URL.Query().Get("a")
	idB := r.URL.Query().Get("b")
	if idA == "" || idB == "" {
		http.Error(w, "query params 'a' and 'b' are required", http.StatusBadRequest)
		return
	}

	runA, err := h.runs.Get(idA)
	if err != nil || runA == nil {
		http.Error(w, "run 'a' not found", http.StatusNotFound)
		return
	}
	runB, err := h.runs.Get(idB)
	if err != nil || runB == nil {
		http.Error(w, "run 'b' not found", http.StatusNotFound)
		return
	}

	writeJSON(w, store.Compare(runA, runB))
}
