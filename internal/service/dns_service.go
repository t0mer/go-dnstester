package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/tomerklein/dnstester/internal/model"
)

type DNSService struct {
	timeout    time.Duration
	httpClient *http.Client
}

func NewDNSService() *DNSService {
	return &DNSService{
		timeout:    5 * time.Second,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *DNSService) Query(server model.DNSServer, fqdn string) model.QueryResult {
	result := model.QueryResult{
		ServerName: server.Name,
		ServerAddr: server.Address,
		Protocol:   server.Protocol,
		FQDN:       fqdn,
		Timestamp:  time.Now(),
	}
	switch server.Protocol {
	case "dot":
		return s.queryDoT(result, server.Address, fqdn)
	case "doh":
		return s.queryDoH(result, server.Address, fqdn)
	default:
		return s.queryUDP(result, server.Address, fqdn)
	}
}

func buildMsg(fqdn string) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	m.RecursionDesired = true
	return m
}

func extractAnswers(r *dns.Msg) []string {
	var out []string
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			out = append(out, a.A.String())
		}
	}
	return out
}

func (s *DNSService) queryUDP(result model.QueryResult, addr, fqdn string) model.QueryResult {
	c := &dns.Client{Timeout: s.timeout}
	if !strings.Contains(addr, ":") {
		addr += ":53"
	}
	start := time.Now()
	r, _, err := c.Exchange(buildMsg(fqdn), addr)
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
	result.Answers = extractAnswers(r)
	return result
}

func (s *DNSService) queryDoT(result model.QueryResult, addr, fqdn string) model.QueryResult {
	c := &dns.Client{
		Net:       "tcp-tls",
		Timeout:   s.timeout,
		TLSConfig: &tls.Config{InsecureSkipVerify: true}, // diagnostic tool — IP addresses are common
	}
	if !strings.Contains(addr, ":") {
		addr += ":853"
	}
	start := time.Now()
	r, _, err := c.Exchange(buildMsg(fqdn), addr)
	result.ResponseMs = float64(time.Since(start).Milliseconds())
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}
	if r.Rcode != dns.RcodeSuccess {
		result.Status = "error"
		result.Error = dns.RcodeToString[r.Rcode]
		return result
	}
	result.Status = "ok"
	result.Answers = extractAnswers(r)
	return result
}

func (s *DNSService) queryDoH(result model.QueryResult, url, fqdn string) model.QueryResult {
	m := buildMsg(fqdn)
	m.Id = 0 // RFC 8484 §4.1

	packed, err := m.Pack()
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("pack: %v", err)
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(packed))
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("build request: %v", err)
		return result
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	result.ResponseMs = float64(time.Since(start).Milliseconds())
	if err != nil {
		result.Status = "error"
		result.Error = "request failed"
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Status = "error"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		result.Status = "error"
		result.Error = "read response failed"
		return result
	}

	var r dns.Msg
	if err := r.Unpack(body); err != nil {
		result.Status = "error"
		result.Error = "parse response failed"
		return result
	}
	if r.Rcode != dns.RcodeSuccess {
		result.Status = "error"
		result.Error = dns.RcodeToString[r.Rcode]
		return result
	}
	result.Status = "ok"
	result.Answers = extractAnswers(&r)
	return result
}
