package service

import (
	"net"
	"strings"
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

func (s *PingService) Ping(server model.DNSServer) model.PingResult {
	result := model.PingResult{
		ServerName: server.Name,
		ServerAddr: server.Address,
	}

	addr := server.Address
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}

	pinger, err := probing.NewPinger(addr)
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

	addr := server.Address
	if !strings.Contains(addr, ":") {
		addr += ":53"
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, s.timeout)
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
