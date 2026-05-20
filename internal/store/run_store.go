package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tomerklein/dnstester/internal/model"
)

type RunSummary struct {
	ID            string     `json:"id"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Status        string     `json:"status"`
	ScheduleID    string     `json:"schedule_id"`
	TotalQueries  int        `json:"total_queries"`
	SuccessCount  int        `json:"success_count"`
	AvgResponseMs float64    `json:"avg_response_ms"`
}

// ListFilter controls which runs are returned by List.
type ListFilter struct {
	From          time.Time // inclusive; if zero and !NoTimeFilter, defaults to now-Hours
	To            time.Time // inclusive; if zero, defaults to now
	Hours         int       // look-back window when From is zero; 0 → 24
	Limit         int       // max rows; 0 → 100
	Offset        int       // rows to skip (for pagination)
	ScheduledOnly bool
	NoTimeFilter  bool // skip time-range filtering entirely
}

type RunStore struct {
	db *sql.DB
}

func NewRunStore(db *sql.DB) *RunStore {
	return &RunStore{db: db}
}

func (s *RunStore) Save(run *model.TestRun) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	isScheduled := 0
	if run.ScheduleID != "" {
		isScheduled = 1
	}
	_, err = tx.Exec(
		`INSERT INTO test_runs (id, started_at, completed_at, status, is_scheduled, schedule_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		run.ID, run.StartedAt, run.CompletedAt, run.Status, isScheduled, run.ScheduleID,
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}

	for _, r := range run.DNSResults {
		answers, _ := json.Marshal(r.Answers)
		_, err = tx.Exec(
			`INSERT INTO dns_results (run_id, server_name, server_addr, fqdn, response_ms, status, answers, error, timestamp, protocol)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			run.ID, r.ServerName, r.ServerAddr, r.FQDN, r.ResponseMs, r.Status, string(answers), r.Error, r.Timestamp, r.Protocol,
		)
		if err != nil {
			return fmt.Errorf("insert dns result: %w", err)
		}
	}

	for _, r := range run.PingResults {
		_, err = tx.Exec(
			`INSERT INTO ping_results (run_id, server_name, server_addr, latency_ms, status, error)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			run.ID, r.ServerName, r.ServerAddr, r.LatencyMs, r.Status, r.Error,
		)
		if err != nil {
			return fmt.Errorf("insert ping result: %w", err)
		}
	}

	return tx.Commit()
}

// buildWhere constructs the shared WHERE clause and args for List and Count.
func buildWhere(f ListFilter) (string, []any) {
	var clauses []string
	var args []any

	if !f.NoTimeFilter {
		now := time.Now()
		to := f.To
		if to.IsZero() {
			to = now
		}
		from := f.From
		if from.IsZero() {
			hours := f.Hours
			if hours <= 0 {
				hours = 24
			}
			from = now.Add(-time.Duration(hours) * time.Hour)
		}
		clauses = append(clauses, "r.started_at >= ?", "r.started_at <= ?")
		args = append(args, from, to)
	}

	if f.ScheduledOnly {
		clauses = append(clauses, "r.is_scheduled = 1")
	}

	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	return where, args
}

func (s *RunStore) List(f ListFilter) ([]RunSummary, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}

	where, args := buildWhere(f)
	args = append(args, limit, f.Offset)

	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT
			r.id, r.started_at, r.completed_at, r.status, r.schedule_id,
			COUNT(d.id)                                                        AS total,
			SUM(CASE WHEN d.status = 'ok' THEN 1 ELSE 0 END)                  AS success,
			COALESCE(AVG(CASE WHEN d.status = 'ok' THEN d.response_ms END), 0) AS avg_ms
		FROM test_runs r
		LEFT JOIN dns_results d ON d.run_id = r.id
		%s
		GROUP BY r.id
		ORDER BY r.started_at DESC
		LIMIT ? OFFSET ?`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunSummary
	for rows.Next() {
		var run RunSummary
		var completedAt sql.NullTime
		if err := rows.Scan(&run.ID, &run.StartedAt, &completedAt, &run.Status, &run.ScheduleID,
			&run.TotalQueries, &run.SuccessCount, &run.AvgResponseMs); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// CountFiltered returns the total number of runs matching f (ignoring Limit/Offset).
func (s *RunStore) CountFiltered(f ListFilter) (int64, error) {
	where, args := buildWhere(f)
	var n int64
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM test_runs r %s`, where), args...,
	).Scan(&n)
	return n, err
}

func (s *RunStore) Get(id string) (*model.TestRun, error) {
	var run model.TestRun
	var completedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, started_at, completed_at, status, schedule_id FROM test_runs WHERE id = ?`, id,
	).Scan(&run.ID, &run.StartedAt, &completedAt, &run.Status, &run.ScheduleID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	dRows, err := s.db.Query(
		`SELECT server_name, server_addr, fqdn, response_ms, status, answers, error, timestamp, protocol
		 FROM dns_results WHERE run_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var r model.QueryResult
		var answers string
		var errStr sql.NullString
		if err := dRows.Scan(&r.ServerName, &r.ServerAddr, &r.FQDN, &r.ResponseMs,
			&r.Status, &answers, &errStr, &r.Timestamp, &r.Protocol); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(answers), &r.Answers) //nolint:errcheck
		if errStr.Valid {
			r.Error = errStr.String
		}
		run.DNSResults = append(run.DNSResults, r)
	}
	if err := dRows.Err(); err != nil {
		return nil, err
	}

	pRows, err := s.db.Query(
		`SELECT server_name, server_addr, latency_ms, status, error
		 FROM ping_results WHERE run_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()
	for pRows.Next() {
		var r model.PingResult
		var errStr sql.NullString
		if err := pRows.Scan(&r.ServerName, &r.ServerAddr, &r.LatencyMs, &r.Status, &errStr); err != nil {
			return nil, err
		}
		if errStr.Valid {
			r.Error = errStr.String
		}
		run.PingResults = append(run.PingResults, r)
	}
	return &run, pRows.Err()
}

// QueryTrends returns per-server average response times bucketed by time.
// hours controls the look-back window; hourly buckets are used for ≤48 h,
// daily buckets otherwise. Bucketing is done in Go to avoid SQLite strftime
// format-string compatibility issues with different driver time encodings.
func (s *RunStore) QueryTrends(hours int) ([]model.TrendPoint, error) {
	from := time.Now().Add(-time.Duration(hours) * time.Hour)
	hourly := hours <= 48

	rows, err := s.db.Query(`
		SELECT d.server_name, d.server_addr, d.protocol,
		       r.started_at, d.response_ms
		FROM test_runs r
		JOIN dns_results d ON d.run_id = r.id
		WHERE r.started_at >= ? AND d.status = 'ok'
		ORDER BY r.started_at ASC
	`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type accKey struct{ server, addr, proto, bucket string }
	type acc struct {
		total float64
		count int
	}
	accMap := map[accKey]*acc{}
	var keyOrder []accKey

	for rows.Next() {
		var serverName, serverAddr, protocol string
		var startedAt time.Time
		var responseMs float64
		if err := rows.Scan(&serverName, &serverAddr, &protocol, &startedAt, &responseMs); err != nil {
			return nil, err
		}
		var bucket string
		if hourly {
			bucket = startedAt.UTC().Format("2006-01-02 15:00")
		} else {
			bucket = startedAt.UTC().Format("2006-01-02")
		}
		k := accKey{serverName, serverAddr, protocol, bucket}
		if _, ok := accMap[k]; !ok {
			accMap[k] = &acc{}
			keyOrder = append(keyOrder, k)
		}
		accMap[k].total += responseMs
		accMap[k].count++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]model.TrendPoint, 0, len(keyOrder))
	for _, k := range keyOrder {
		a := accMap[k]
		points = append(points, model.TrendPoint{
			ServerName:  k.server,
			ServerAddr:  k.addr,
			Protocol:    k.proto,
			Bucket:      k.bucket,
			AvgMs:       math.Round(a.total/float64(a.count)*100) / 100,
			SampleCount: a.count,
		})
	}
	return points, nil
}

// ListFull returns the last n test runs with their full DNS and ping results.
func (s *RunStore) ListFull(n int) ([]*model.TestRun, error) {
	summaries, err := s.List(ListFilter{Limit: n, NoTimeFilter: true})
	if err != nil {
		return nil, err
	}
	runs := make([]*model.TestRun, 0, len(summaries))
	for _, summary := range summaries {
		run, err := s.Get(summary.ID)
		if err != nil || run == nil {
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// Count returns the total number of test runs stored in the database.
func (s *RunStore) Count() (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM test_runs`).Scan(&n)
	return n, err
}

// LastRunForSchedule returns when the given schedule last ran (zero time if never).
func (s *RunStore) LastRunForSchedule(scheduleID string) (time.Time, error) {
	var t time.Time
	err := s.db.QueryRow(
		`SELECT started_at FROM test_runs WHERE schedule_id = ? ORDER BY started_at DESC LIMIT 1`,
		scheduleID,
	).Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}

// Compare computes a structured diff between two runs.
func Compare(a, b *model.TestRun) *model.CompareResult {
	type acc struct {
		addr             string
		totalA, okA      int
		sumA             float64
		totalB, okB      int
		sumB             float64
	}
	m := map[string]*acc{}

	for _, r := range a.DNSResults {
		if _, ok := m[r.ServerName]; !ok {
			m[r.ServerName] = &acc{addr: r.ServerAddr}
		}
		m[r.ServerName].totalA++
		if r.Status == "ok" {
			m[r.ServerName].okA++
			m[r.ServerName].sumA += r.ResponseMs
		}
	}
	for _, r := range b.DNSResults {
		if _, ok := m[r.ServerName]; !ok {
			m[r.ServerName] = &acc{addr: r.ServerAddr}
		}
		m[r.ServerName].totalB++
		if r.Status == "ok" {
			m[r.ServerName].okB++
			m[r.ServerName].sumB += r.ResponseMs
		}
	}

	result := &model.CompareResult{RunA: a, RunB: b}
	var totalAvgA, totalAvgB float64
	var count int

	for name, ac := range m {
		stat := model.ServerStat{
			ServerName: name,
			ServerAddr: ac.addr,
			SuccessA:   ac.okA,
			SuccessB:   ac.okB,
			TotalA:     ac.totalA,
			TotalB:     ac.totalB,
		}
		if ac.okA > 0 {
			stat.AvgMsA = ac.sumA / float64(ac.okA)
		}
		if ac.okB > 0 {
			stat.AvgMsB = ac.sumB / float64(ac.okB)
		}
		if ac.okA > 0 && ac.okB > 0 {
			stat.DeltaMs = stat.AvgMsB - stat.AvgMsA
			if stat.AvgMsA != 0 {
				stat.DeltaPct = (stat.DeltaMs / stat.AvgMsA) * 100
			}
			totalAvgA += stat.AvgMsA
			totalAvgB += stat.AvgMsB
			count++
		}
		result.ByServer = append(result.ByServer, stat)
	}

	if count > 0 {
		avgA := totalAvgA / float64(count)
		avgB := totalAvgB / float64(count)
		result.OverallDeltaMs = avgB - avgA
		if avgA != 0 {
			result.OverallDeltaPct = math.Round((result.OverallDeltaMs/avgA)*1000) / 10
		}
	}

	return result
}
