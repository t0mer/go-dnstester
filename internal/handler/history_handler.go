package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tomerklein/dnstester/internal/store"
)

type HistoryHandler struct {
	runs *store.RunStore
}

func NewHistoryHandler(runs *store.RunStore) *HistoryHandler {
	return &HistoryHandler{runs: runs}
}

// List godoc
//
//	@Summary		List test run history
//	@Description	Returns summaries of past test runs. Defaults to the last 24 hours. Use `hours` to change the look-back window or provide explicit `from`/`to` dates.
//	@Tags			History
//	@Produce		json
//	@Param			hours		query	int		false	"Look-back window in hours (default 24, ignored when from/to are set)"	minimum(1)
//	@Param			from		query	string	false	"Start of date range (RFC3339, e.g. 2006-01-02T15:04:05Z)"
//	@Param			to			query	string	false	"End of date range (RFC3339)"
//	@Param			limit		query	int		false	"Maximum number of results (default 100, max 500)"	minimum(1)	maximum(500)
//	@Param			scheduled	query	bool	false	"Filter to scheduled runs only"
//	@Success		200	{array}		store.RunSummary
//	@Failure		500	{string}	string	"internal error"
//	@Router			/history [get]
type historyPage struct {
	Total int64            `json:"total"`
	Items []store.RunSummary `json:"items"`
}

func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	f := store.ListFilter{}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			f.Limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}

	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Hours = n
		}
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = t
		}
	}

	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = t
		}
	}

	f.ScheduledOnly = r.URL.Query().Get("scheduled") == "true"

	total, err := h.runs.CountFiltered(f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	runs, err := h.runs.List(f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []store.RunSummary{}
	}
	writeJSON(w, historyPage{Total: total, Items: runs})
}

// Get godoc
//
//	@Summary		Get test run by ID
//	@Tags			History
//	@Produce		json
//	@Param			id	path		string	true	"Run ID"
//	@Success		200	{object}	model.TestRun
//	@Failure		404	{string}	string	"not found"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/history/{id} [get]
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

// Compare godoc
//
//	@Summary		Compare two test runs
//	@Tags			History
//	@Produce		json
//	@Param			a	query		string	true	"Run ID A"
//	@Param			b	query		string	true	"Run ID B"
//	@Success		200	{object}	model.CompareResult
//	@Failure		400	{string}	string	"query params 'a' and 'b' are required"
//	@Failure		404	{string}	string	"run not found"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/compare [get]
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
