package model

type Config struct {
	Servers    []DNSServer     `json:"servers"`
	FQDNs      []string        `json:"fqdns"`
	Schedules  []ScheduledScan `json:"schedules"`
	AutoUpdate bool            `json:"auto_update"`
	Auth       AuthConfig      `json:"auth"`
}

type AuthConfig struct {
	Enabled         bool   `json:"enabled"`
	Username        string `json:"username"`
	PasswordHash    string `json:"password_hash"`
	APITokenEnabled bool   `json:"api_token_enabled"`
	APITokenHash    string `json:"api_token_hash"`
}

// ScheduledScan defines when an automatic test run is triggered.
// Only the fields relevant to the chosen Type need to be set.
type ScheduledScan struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // interval | daily | weekdays | weekly | monthly | once

	// interval — run every N minutes
	IntervalMinutes int `json:"interval_minutes,omitempty"`

	// daily / weekdays / weekly / monthly — time of day "HH:MM" (24-hour)
	TimeOfDay string `json:"time_of_day,omitempty"`

	// weekdays — days of week to run (0=Sun … 6=Sat)
	Weekdays []int `json:"weekdays,omitempty"`

	// weekly — single weekday (0=Sun … 6=Sat)
	Weekday int `json:"weekday,omitempty"`

	// monthly — day within the month (1–31)
	DayOfMonth int `json:"day_of_month,omitempty"`

	// once — exact moment to run (RFC3339)
	RunAt string `json:"run_at,omitempty"`
}
