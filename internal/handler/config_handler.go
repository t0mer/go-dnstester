package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/model"
)

type ConfigHandler struct {
	svc *config.Service
}

func NewConfigHandler(svc *config.Service) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg)
}

func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	var cfg model.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.svc.Save(&cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg)
}

func (h *ConfigHandler) Backup(w http.ResponseWriter, r *http.Request) {
	// Save current config first so we have something to back up even on first run.
	cfg, err := h.svc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.svc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.svc.Backup(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConfigHandler) Restore(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.Restore()
	if err != nil {
		http.Error(w, "no backup found", http.StatusNotFound)
		return
	}
	writeJSON(w, cfg)
}

func (h *ConfigHandler) Export(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.Export()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="dnstester-config.json"`)
	w.Write(data)
}

func (h *ConfigHandler) Import(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	cfg, err := h.svc.Import(data)
	if err != nil {
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}
	writeJSON(w, cfg)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
