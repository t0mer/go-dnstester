package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tomerklein/dnstester/internal/service"
	"github.com/tomerklein/dnstester/internal/store"
)

const ns = "dnstester"

// Collector implements prometheus.Collector and emits metrics derived from
// the latest in-memory test run and aggregate counts from the database.
type Collector struct {
	testSvc *service.TestService
	runs    *store.RunStore

	dnsResponseSeconds  *prometheus.Desc
	pingLatencySeconds  *prometheus.Desc
	lastRunTimestamp    *prometheus.Desc
	lastRunDuration     *prometheus.Desc
	testRunsTotal       *prometheus.Desc
}

func NewCollector(testSvc *service.TestService, runs *store.RunStore) *Collector {
	return &Collector{
		testSvc: testSvc,
		runs:    runs,

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
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dnsResponseSeconds
	ch <- c.pingLatencySeconds
	ch <- c.lastRunTimestamp
	ch <- c.lastRunDuration
	ch <- c.testRunsTotal
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if count, err := c.runs.Count(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.testRunsTotal, prometheus.GaugeValue, float64(count))
	}

	run := c.testSvc.Latest()
	if run == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.lastRunTimestamp, prometheus.GaugeValue,
		float64(run.StartedAt.Unix()),
	)

	if run.CompletedAt != nil {
		ch <- prometheus.MustNewConstMetric(
			c.lastRunDuration, prometheus.GaugeValue,
			run.CompletedAt.Sub(run.StartedAt).Seconds(),
		)
	}

	for _, r := range run.DNSResults {
		ch <- prometheus.MustNewConstMetric(
			c.dnsResponseSeconds, prometheus.GaugeValue,
			r.ResponseMs/1000.0,
			r.ServerName, r.ServerAddr, r.FQDN, r.Status,
		)
	}

	for _, r := range run.PingResults {
		ch <- prometheus.MustNewConstMetric(
			c.pingLatencySeconds, prometheus.GaugeValue,
			r.LatencyMs/1000.0,
			r.ServerName, r.ServerAddr, r.Status,
		)
	}
}
