package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/model"
)

type ScheduleHandler struct {
	cfgSvc *config.Service
}

func NewScheduleHandler(cfgSvc *config.Service) *ScheduleHandler {
	return &ScheduleHandler{cfgSvc: cfgSvc}
}

// List godoc
//
//	@Summary		List scheduled scans
//	@Tags			Schedules
//	@Produce		json
//	@Success		200	{array}		model.ScheduledScan
//	@Failure		500	{string}	string	"internal error"
//	@Router			/schedules [get]
func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	schedules := cfg.Schedules
	if schedules == nil {
		schedules = []model.ScheduledScan{}
	}
	writeJSON(w, schedules)
}

// Create godoc
//
//	@Summary		Create a scheduled scan
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			schedule	body		model.ScheduledScan	true	"Scheduled scan definition (id is ignored and generated server-side)"
//	@Success		201	{object}	model.ScheduledScan
//	@Failure		400	{string}	string	"invalid request"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/schedules [post]
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sc model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := validateSchedule(&sc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sc.ID = genScheduleID()

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	cfg.Schedules = append(cfg.Schedules, sc)
	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sc)
}

// Update godoc
//
//	@Summary		Update an existing scheduled scan
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string				true	"Schedule ID"
//	@Param			schedule	body		model.ScheduledScan	true	"Updated schedule (id in body is ignored; path id is used)"
//	@Success		200	{object}	model.ScheduledScan
//	@Failure		400	{string}	string	"invalid request"
//	@Failure		404	{string}	string	"schedule not found"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/schedules/{id} [put]
func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var sc model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := validateSchedule(&sc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sc.ID = id

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	idx := -1
	for i, s := range cfg.Schedules {
		if s.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Error(w, "schedule not found", http.StatusNotFound)
		return
	}
	cfg.Schedules[idx] = sc
	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, sc)
}

// Delete godoc
//
//	@Summary		Delete a scheduled scan
//	@Tags			Schedules
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		204	{string}	string	"no content"
//	@Failure		404	{string}	string	"schedule not found"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/schedules/{id} [delete]
func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	idx := -1
	for i, s := range cfg.Schedules {
		if s.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Error(w, "schedule not found", http.StatusNotFound)
		return
	}
	cfg.Schedules = append(cfg.Schedules[:idx], cfg.Schedules[idx+1:]...)
	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var validScheduleTypes = map[string]bool{
	"interval": true,
	"daily":    true,
	"weekdays": true,
	"weekly":   true,
	"monthly":  true,
	"once":     true,
}

func validateSchedule(sc *model.ScheduledScan) error {
	if sc.Name == "" {
		return errors.New("name is required")
	}
	if !validScheduleTypes[sc.Type] {
		return fmt.Errorf("invalid type %q; must be one of: interval, daily, weekdays, weekly, monthly, once", sc.Type)
	}
	switch sc.Type {
	case "interval":
		if sc.IntervalMinutes <= 0 {
			return errors.New("interval_minutes must be > 0")
		}
	case "daily", "weekdays", "weekly", "monthly":
		if sc.TimeOfDay == "" {
			return errors.New("time_of_day is required for this schedule type")
		}
	case "once":
		if sc.RunAt == "" {
			return errors.New("run_at is required for once schedule")
		}
	}
	return nil
}

func genScheduleID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}
