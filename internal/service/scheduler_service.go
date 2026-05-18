package service

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/model"
	"github.com/tomerklein/dnstester/internal/store"
)

type SchedulerService struct {
	cfgSvc  *config.Service
	testSvc *TestService
	runs    *store.RunStore
	done    chan struct{}
}

func NewSchedulerService(cfgSvc *config.Service, testSvc *TestService, runs *store.RunStore) *SchedulerService {
	return &SchedulerService{
		cfgSvc:  cfgSvc,
		testSvc: testSvc,
		runs:    runs,
		done:    make(chan struct{}),
	}
}

func (s *SchedulerService) Start() { go s.loop() }

func (s *SchedulerService) Stop() { close(s.done) }

func (s *SchedulerService) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Cache last-run times to avoid re-querying DB every tick.
	lastRuns := map[string]time.Time{}

	for {
		select {
		case <-s.done:
			return
		case now := <-ticker.C:
			cfg, err := s.cfgSvc.Load()
			if err != nil {
				continue
			}
			for _, sc := range cfg.Schedules {
				if !sc.Enabled {
					continue
				}
				lr, cached := lastRuns[sc.ID]
				if !cached {
					lr, _ = s.runs.LastRunForSchedule(sc.ID)
					lastRuns[sc.ID] = lr
				}
				if shouldRun(&sc, lr, now) {
					lastRuns[sc.ID] = now
					go s.execute(sc)
				}
			}
		}
	}
}

func (s *SchedulerService) execute(sc model.ScheduledScan) {
	cfg, err := s.cfgSvc.Load()
	if err != nil {
		log.Printf("scheduler[%s]: load config: %v", sc.Name, err)
		return
	}
	run := s.testSvc.Run(cfg.Servers, cfg.FQDNs)
	run.ScheduleID = sc.ID
	if err := s.runs.Save(run); err != nil {
		log.Printf("scheduler[%s]: save run: %v", sc.Name, err)
	} else {
		log.Printf("scheduler[%s]: completed run %s (%d results)", sc.Name, run.ID, len(run.DNSResults))
	}
}

// shouldRun returns true if schedule sc should fire at time now given its last run time.
func shouldRun(sc *model.ScheduledScan, lastRun time.Time, now time.Time) bool {
	switch sc.Type {
	case "interval":
		if sc.IntervalMinutes <= 0 {
			return false
		}
		if lastRun.IsZero() {
			return true
		}
		return now.After(lastRun.Add(time.Duration(sc.IntervalMinutes) * time.Minute))

	case "daily":
		t := todayAt(sc.TimeOfDay, now)
		return now.After(t) && lastRun.Before(t)

	case "weekdays":
		if !intSliceContains(sc.Weekdays, int(now.Weekday())) {
			return false
		}
		t := todayAt(sc.TimeOfDay, now)
		return now.After(t) && lastRun.Before(t)

	case "weekly":
		if int(now.Weekday()) != sc.Weekday {
			return false
		}
		t := todayAt(sc.TimeOfDay, now)
		return now.After(t) && lastRun.Before(t)

	case "monthly":
		if now.Day() != sc.DayOfMonth {
			return false
		}
		t := todayAt(sc.TimeOfDay, now)
		return now.After(t) && lastRun.Before(t)

	case "once":
		if sc.RunAt == "" || !lastRun.IsZero() {
			return false
		}
		runAt, err := time.Parse(time.RFC3339, sc.RunAt)
		if err != nil {
			return false
		}
		return now.After(runAt)
	}
	return false
}

func todayAt(hhmm string, ref time.Time) time.Time {
	h, m := parseHHMM(hhmm)
	return time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, 0, 0, ref.Location())
}

func parseHHMM(t string) (h, m int) {
	parts := strings.SplitN(t, ":", 2)
	h, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		m, _ = strconv.Atoi(parts[1])
	}
	return
}

func intSliceContains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
