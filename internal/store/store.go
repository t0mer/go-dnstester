package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS test_runs (
			id           TEXT PRIMARY KEY,
			started_at   DATETIME NOT NULL,
			completed_at DATETIME,
			status       TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS dns_results (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id      TEXT    NOT NULL REFERENCES test_runs(id),
			server_name TEXT    NOT NULL,
			server_addr TEXT    NOT NULL,
			fqdn        TEXT    NOT NULL,
			response_ms REAL    NOT NULL,
			status      TEXT    NOT NULL,
			answers     TEXT,
			error       TEXT,
			timestamp   DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS ping_results (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id      TEXT NOT NULL REFERENCES test_runs(id),
			server_name TEXT NOT NULL,
			server_addr TEXT NOT NULL,
			latency_ms  REAL NOT NULL,
			status      TEXT NOT NULL,
			error       TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_dns_run  ON dns_results(run_id);
		CREATE INDEX IF NOT EXISTS idx_ping_run ON ping_results(run_id);
	`)
	if err != nil {
		return err
	}
	// Additive column migrations — errors mean the column already exists; safe to ignore.
	addColumn(db, "test_runs", "is_scheduled", "INTEGER NOT NULL DEFAULT 0")
	addColumn(db, "test_runs", "schedule_id", "TEXT NOT NULL DEFAULT ''")
	return nil
}

func addColumn(db *sql.DB, table, column, def string) {
	db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, def)) //nolint:errcheck
}
