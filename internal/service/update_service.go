package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tomerklein/dnstester/internal/model"
)

const githubReleasesURL = "https://api.github.com/repos/t0mer/go-dnstester/releases/latest"

type UpdateService struct {
	currentVersion string
	client         *http.Client
}

func NewUpdateService(currentVersion string) *UpdateService {
	return &UpdateService{
		currentVersion: currentVersion,
		client:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *UpdateService) CurrentVersion() string {
	return s.currentVersion
}

func (s *UpdateService) Check() (*model.UpdateInfo, error) {
	req, err := http.NewRequest(http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "dnstester/"+s.currentVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %s", resp.Status)
	}

	var release struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	available := release.TagName != "" &&
		s.currentVersion != "dev" &&
		release.TagName != s.currentVersion

	return &model.UpdateInfo{
		Current:      s.currentVersion,
		Latest:       release.TagName,
		Available:    available,
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
		ReleaseURL:   release.HTMLURL,
	}, nil
}
