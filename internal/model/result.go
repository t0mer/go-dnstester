package model

import "time"

type QueryResult struct {
	ServerName string    `json:"server_name"`
	ServerAddr string    `json:"server_addr"`
	FQDN       string    `json:"fqdn"`
	ResponseMs float64   `json:"response_ms"`
	Status     string    `json:"status"` // ok | error | timeout
	Answers    []string  `json:"answers,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type PingResult struct {
	ServerName string  `json:"server_name"`
	ServerAddr string  `json:"server_addr"`
	LatencyMs  float64 `json:"latency_ms"`
	Status     string  `json:"status"` // ok | error | timeout
	Error      string  `json:"error,omitempty"`
}

type TestRun struct {
	ID          string        `json:"id"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	DNSResults  []QueryResult `json:"dns_results"`
	PingResults []PingResult  `json:"ping_results"`
	Status      string        `json:"status"`               // running | completed
	ScheduleID  string        `json:"schedule_id,omitempty"` // set when triggered by a schedule
}

// ServerStat is a per-server comparison summary returned by the compare endpoint.
type ServerStat struct {
	ServerName string  `json:"server_name"`
	ServerAddr string  `json:"server_addr"`
	AvgMsA     float64 `json:"avg_ms_a"`
	AvgMsB     float64 `json:"avg_ms_b"`
	DeltaMs    float64 `json:"delta_ms"`
	DeltaPct   float64 `json:"delta_pct"`
	SuccessA   int     `json:"success_a"`
	SuccessB   int     `json:"success_b"`
	TotalA     int     `json:"total_a"`
	TotalB     int     `json:"total_b"`
}

type CompareResult struct {
	RunA            *TestRun     `json:"run_a"`
	RunB            *TestRun     `json:"run_b"`
	ByServer        []ServerStat `json:"by_server"`
	OverallDeltaMs  float64      `json:"overall_delta_ms"`
	OverallDeltaPct float64      `json:"overall_delta_pct"`
}
