package handler

import (
	"net/http"

	"github.com/tomerklein/dnstester/internal/service"
)

type UpdateHandler struct {
	svc *service.UpdateService
}

func NewUpdateHandler(svc *service.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

func (h *UpdateHandler) Check(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.Check()
	if err != nil {
		http.Error(w, "update check failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, info)
}
