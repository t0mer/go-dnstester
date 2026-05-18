package service

import (
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/tomerklein/dnstester/internal/model"
)

type DNSService struct {
	timeout time.Duration
}

func NewDNSService() *DNSService {
	return &DNSService{timeout: 5 * time.Second}
}

func (s *DNSService) Query(server model.DNSServer, fqdn string) model.QueryResult {
	result := model.QueryResult{
		ServerName: server.Name,
		ServerAddr: server.Address,
		FQDN:       fqdn,
		Timestamp:  time.Now(),
	}

	c := &dns.Client{Timeout: s.timeout}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	m.RecursionDesired = true

	addr := server.Address
	if !strings.Contains(addr, ":") {
		addr += ":53"
	}

	start := time.Now()
	r, _, err := c.Exchange(m, addr)
	result.ResponseMs = float64(time.Since(start).Milliseconds())

	if err != nil {
		result.Status = "error"
		result.Error = "query failed"
		return result
	}

	if r.Rcode != dns.RcodeSuccess {
		result.Status = "error"
		result.Error = dns.RcodeToString[r.Rcode]
		return result
	}

	result.Status = "ok"
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			result.Answers = append(result.Answers, a.A.String())
		}
	}
	return result
}
