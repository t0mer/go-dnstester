package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/tomerklein/dnstester/internal/model"
)

type TestService struct {
	dns  *DNSService
	ping *PingService
	mu   sync.RWMutex
	last *model.TestRun
}

func NewTestService(dns *DNSService, ping *PingService) *TestService {
	return &TestService{dns: dns, ping: ping}
}

func (s *TestService) Run(servers []model.DNSServer, fqdns []string) *model.TestRun {
	run := &model.TestRun{
		ID:        newID(),
		StartedAt: time.Now(),
		Status:    "running",
	}

	s.mu.Lock()
	s.last = run
	s.mu.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, srv := range servers {
		if !srv.Enabled {
			continue
		}

		for _, fqdn := range fqdns {
			wg.Add(1)
			go func(srv model.DNSServer, fqdn string) {
				defer wg.Done()
				r := s.dns.Query(srv, fqdn)
				mu.Lock()
				run.DNSResults = append(run.DNSResults, r)
				mu.Unlock()
			}(srv, fqdn)
		}

		wg.Add(1)
		go func(srv model.DNSServer) {
			defer wg.Done()
			r := s.ping.Ping(srv)
			mu.Lock()
			run.PingResults = append(run.PingResults, r)
			mu.Unlock()
		}(srv)
	}

	wg.Wait()

	now := time.Now()
	run.CompletedAt = &now
	run.Status = "completed"

	return run
}

func (s *TestService) Latest() *model.TestRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.last
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
