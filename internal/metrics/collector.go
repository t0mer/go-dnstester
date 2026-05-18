package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tomerklein/dnstester/internal/service"
	"github.com/tomerklein/dnstester/internal/store"
)

const (
	ns       = "dnstester"
	maxRuns  = 5
)

// Collector implements prometheus.Collector.
//
// Two sets of metrics are emitted:
//
//  1. Scalar / latest-only metrics (no run_id label) — useful for simple
//     alerting rules and dashboards that only care about the current state.
//
//  2. Per-run metrics (run_id label, last 5 runs) — each scrape reflects the
//     most recent 5 completed runs, enabling Grafana to plot trends and
//     compare results across time.
type Collector struct {
	testSvc *service.TestService
	runs    *store.RunStore

	// scalar metrics — latest run
	dnsResponseSeconds *prometheus.Desc
	pingLatencySeconds *prometheus.Desc
	lastRunTimestamp   *prometheus.Desc
	lastRunDuration    *prometheus.Desc
	testRunsTotal      *prometheus.Desc

	// per-run metrics — last 5 runs (labelled by run_id)
	runInfo               *prometheus.Desc
	runDNSResponseSeconds *prometheus.Desc
	runPingLatencySeconds *prometheus.Desc
	runDurationSeconds    *prometheus.Desc
}

func NewCollector(testSvc *service.TestService, runs *store.RunStore) *Collector {
	return &Collector{
		testSvc: testSvc,
		runs:    runs,

		// --- scalar ---
		dnsResponseSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "dns", "response_seconds"),
			"DNS query response time in seconds from the latest test run.",
			[]string{"server_name", "server_addr", "fqdn", "status"}, nil,
		),
		pingLatencySeconds: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "ping", "latency_seconds"),
			"ICMP ping latency in seconds from the latest test run.",
			[]string{"server_name", "server_addr", "status"}, nil,
		),
		lastRunTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "last_run_timestamp_seconds"),
			"Unix timestamp of the most recent test run.",
			nil, nil,
		),
		lastRunDuration: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "last_run_duration_seconds"),
			"Wall-clock duration of the most recent test run in seconds.",
			nil, nil,
		),
		testRunsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "test_runs_total"),
			"Total number of test runs recorded in the database.",
			nil, nil,
		),

		// --- per-run (last 5) ---
		runInfo: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "run", "info"),
			"Metadata for each of the last 5 test runs. Value is always 1; use labels to identify the run.",
			[]string{"run_id", "started_at", "status"}, nil,
		),
		runDurationSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "run", "duration_seconds"),
			"Wall-clock duration of a test run in seconds.",
			[]string{"run_id"}, nil,
		),
		runDNSResponseSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "run", "dns_response_seconds"),
			"DNS query response time in seconds, labelled by run.",
			[]string{"run_id", "server_name", "server_addr", "fqdn", "status"}, nil,
		),
		runPingLatencySeconds: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "run", "ping_latency_seconds"),
			"ICMP ping latency in seconds, labelled by run.",
			[]string{"run_id", "server_name", "server_addr", "status"}, nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dnsResponseSeconds
	ch <- c.pingLatencySeconds
	ch <- c.lastRunTimestamp
	ch <- c.lastRunDuration
	ch <- c.testRunsTotal
	ch <- c.runInfo
	ch <- c.runDurationSeconds
	ch <- c.runDNSResponseSeconds
	ch <- c.runPingLatencySeconds
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	// --- aggregate ---
	if count, err := c.runs.Count(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.testRunsTotal, prometheus.GaugeValue, float64(count))
	}

	// --- scalar (latest run from memory) ---
	latest := c.testSvc.Latest()
	if latest != nil {
		ch <- prometheus.MustNewConstMetric(
			c.lastRunTimestamp, prometheus.GaugeValue,
			float64(latest.StartedAt.Unix()),
		)
		if latest.CompletedAt != nil {
			ch <- prometheus.MustNewConstMetric(
				c.lastRunDuration, prometheus.GaugeValue,
				latest.CompletedAt.Sub(latest.StartedAt).Seconds(),
			)
		}
		for _, r := range latest.DNSResults {
			ch <- prometheus.MustNewConstMetric(
				c.dnsResponseSeconds, prometheus.GaugeValue,
				r.ResponseMs/1000.0,
				r.ServerName, r.ServerAddr, r.FQDN, r.Status,
			)
		}
		for _, r := range latest.PingResults {
			ch <- prometheus.MustNewConstMetric(
				c.pingLatencySeconds, prometheus.GaugeValue,
				r.LatencyMs/1000.0,
				r.ServerName, r.ServerAddr, r.Status,
			)
		}
	}

	// --- per-run (last 5 from DB) ---
	recentRuns, err := c.runs.ListFull(maxRuns)
	if err != nil {
		return
	}
	for _, run := range recentRuns {
		startedAt := run.StartedAt.UTC().Format("2006-01-02T15:04:05Z")

		ch <- prometheus.MustNewConstMetric(
			c.runInfo, prometheus.GaugeValue, 1,
			run.ID, startedAt, run.Status,
		)

		if run.CompletedAt != nil {
			ch <- prometheus.MustNewConstMetric(
				c.runDurationSeconds, prometheus.GaugeValue,
				run.CompletedAt.Sub(run.StartedAt).Seconds(),
				run.ID,
			)
		}

		for _, r := range run.DNSResults {
			ch <- prometheus.MustNewConstMetric(
				c.runDNSResponseSeconds, prometheus.GaugeValue,
				r.ResponseMs/1000.0,
				run.ID, r.ServerName, r.ServerAddr, r.FQDN, r.Status,
			)
		}

		for _, r := range run.PingResults {
			ch <- prometheus.MustNewConstMetric(
				c.runPingLatencySeconds, prometheus.GaugeValue,
				r.LatencyMs/1000.0,
				run.ID, r.ServerName, r.ServerAddr, r.Status,
			)
		}
	}
}
