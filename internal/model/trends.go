package model

// TrendPoint is one aggregated data point: the average successful DNS response
// time for a specific server during a time bucket (hour or day).
type TrendPoint struct {
	ServerName  string  `json:"server_name"`
	ServerAddr  string  `json:"server_addr"`
	Protocol    string  `json:"protocol,omitempty"`
	Bucket      string  `json:"bucket"`       // "YYYY-MM-DD" or "YYYY-MM-DD HH:00"
	AvgMs       float64 `json:"avg_ms"`
	SampleCount int     `json:"sample_count"`
}
