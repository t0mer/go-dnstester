package model

type DNSServer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}
