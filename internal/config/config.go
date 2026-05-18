package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tomerklein/dnstester/internal/model"
)

var defaultServers = []model.DNSServer{
	{Name: "Cloudflare", Address: "1.1.1.1", Enabled: true},
	{Name: "Cloudflare Alt", Address: "1.0.0.1", Enabled: true},
	{Name: "Google", Address: "8.8.8.8", Enabled: true},
	{Name: "Google Alt", Address: "8.8.4.4", Enabled: true},
	{Name: "Quad9", Address: "9.9.9.9", Enabled: true},
	{Name: "OpenDNS", Address: "208.67.222.222", Enabled: true},
	{Name: "OpenDNS Alt", Address: "208.67.220.220", Enabled: true},
	{Name: "AdGuard", Address: "94.140.14.14", Enabled: true},
}

var defaultFQDNs = []string{
	"google.com",
	"cloudflare.com",
	"github.com",
	"microsoft.com",
	"apple.com",
}

type Service struct {
	path string
}

func NewService(configDir string) *Service {
	return &Service{path: filepath.Join(configDir, "dnstester.json")}
}

func (s *Service) Load() (*model.Config, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &model.Config{Servers: defaultServers, FQDNs: defaultFQDNs}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func (s *Service) Save(cfg *model.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return atomicWrite(s.path, data)
}

func (s *Service) Backup() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read config for backup: %w", err)
	}
	return atomicWrite(backupPath(s.path), data)
}

func (s *Service) Restore() (*model.Config, error) {
	data, err := os.ReadFile(backupPath(s.path))
	if err != nil {
		return nil, fmt.Errorf("read backup: %w", err)
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse backup: %w", err)
	}
	if err := atomicWrite(s.path, data); err != nil {
		return nil, fmt.Errorf("restore config: %w", err)
	}
	return &cfg, nil
}

func (s *Service) Export() ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		cfg := &model.Config{Servers: defaultServers, FQDNs: defaultFQDNs}
		return json.MarshalIndent(cfg, "", "  ")
	}
	return data, err
}

func (s *Service) Import(data []byte) (*model.Config, error) {
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse import: %w", err)
	}
	if err := atomicWrite(s.path, data); err != nil {
		return nil, fmt.Errorf("import config: %w", err)
	}
	return &cfg, nil
}

func backupPath(path string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)] + ".backup" + ext
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
