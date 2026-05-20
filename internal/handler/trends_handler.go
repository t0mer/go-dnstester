package handler

import (
	"net/http"
	"strconv"

	"github.com/tomerklein/dnstester/internal/model"
	"github.com/tomerklein/dnstester/internal/store"
)

type TrendsHandler struct {
	runs *store.RunStore
}

func NewTrendsHandler(runs *store.RunStore) *TrendsHandler {
	return &TrendsHandler{runs: runs}
}

func (h *TrendsHandler) Get(w http.ResponseWriter, r *http.Request) {
	hours := 168 // 7 days default
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 8760 {
			hours = n
		}
	}

	points, err := h.runs.QueryTrends(hours)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []model.TrendPoint{}
	}
	writeJSON(w, points)
}
