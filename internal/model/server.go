package model

type DNSServer struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	// Protocol controls how queries are sent.
	// "" or "udp" = plain DNS/53 (default)
	// "dot"        = DNS over TLS / port 853
	// "doh"        = DNS over HTTPS (Address is a full URL)
	Protocol string `json:"protocol,omitempty"`
	Enabled  bool   `json:"enabled"`
}
