package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tomerklein/dnstester/internal/service"
)

type UpdateHandler struct {
	svc *service.UpdateService
}

func NewUpdateHandler(svc *service.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

func (h *UpdateHandler) Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"version": h.svc.CurrentVersion()})
}

func (h *UpdateHandler) Check(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.Check()
	if err != nil {
		http.Error(w, "update check failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, info)
}

func (h *UpdateHandler) Apply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DownloadURL == "" {
		http.Error(w, "download_url required", http.StatusBadRequest)
		return
	}

	if err := h.svc.Apply(body.DownloadURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "restarting"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Exit so the process manager (Docker, systemd) restarts with the new binary.
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Println("update applied — restarting")
		os.Exit(0)
	}()
}
