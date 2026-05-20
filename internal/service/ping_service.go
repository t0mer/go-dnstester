package service

import (
	"net"
	"net/url"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/tomerklein/dnstester/internal/model"
)

type PingService struct {
	timeout time.Duration
}

func NewPingService() *PingService {
	return &PingService{timeout: 5 * time.Second}
}

// pingTargets returns the ICMP host and the TCP address to use as fallback.
func pingTargets(server model.DNSServer) (icmpHost, tcpAddr string) {
	switch server.Protocol {
	case "doh":
		u, err := url.Parse(server.Address)
		if err != nil {
			return server.Address, server.Address
		}
		host := u.Hostname()
		port := u.Port()
		if port == "" {
			port = "443"
		}
		return host, net.JoinHostPort(host, port)
	case "dot":
		addr := server.Address
		if host, port, err := net.SplitHostPort(addr); err == nil {
			return host, net.JoinHostPort(host, port)
		}
		return addr, net.JoinHostPort(addr, "853")
	default:
		addr := server.Address
		if host, _, err := net.SplitHostPort(addr); err == nil {
			return host, addr
		}
		return addr, net.JoinHostPort(addr, "53")
	}
}

func (s *PingService) Ping(server model.DNSServer) model.PingResult {
	result := model.PingResult{
		ServerName: server.Name,
		ServerAddr: server.Address,
	}

	icmpHost, _ := pingTargets(server)

	pinger, err := probing.NewPinger(icmpHost)
	if err != nil {
		return s.tcpPing(server)
	}

	pinger.Count = 3
	pinger.Timeout = s.timeout
	pinger.SetPrivileged(false)

	if err := pinger.Run(); err != nil {
		return s.tcpPing(server)
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		result.Status = "timeout"
		return result
	}

	result.LatencyMs = float64(stats.AvgRtt) / float64(time.Millisecond)
	result.Status = "ok"
	return result
}

func (s *PingService) tcpPing(server model.DNSServer) model.PingResult {
	result := model.PingResult{
		ServerName: server.Name,
		ServerAddr: server.Address,
	}

	_, tcpAddr := pingTargets(server)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", tcpAddr, s.timeout)
	if err != nil {
		result.Status = "error"
		result.Error = "unreachable"
		return result
	}
	conn.Close()

	result.LatencyMs = float64(time.Since(start)) / float64(time.Millisecond)
	result.Status = "ok"
	return result
}
