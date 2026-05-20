package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	available := release.TagName != "" &&
		s.currentVersion != "dev" &&
		release.TagName != s.currentVersion

	// Find the asset matching the current platform.
	suffix := platformSuffix()
	var downloadURL string
	for _, a := range release.Assets {
		if strings.Contains(a.Name, suffix) {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	return &model.UpdateInfo{
		Current:      s.currentVersion,
		Latest:       release.TagName,
		Available:    available,
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
		ReleaseURL:   release.HTMLURL,
		DownloadURL:  downloadURL,
	}, nil
}

// Apply downloads the binary at downloadURL, atomically replaces the running
// executable, and returns. The caller is responsible for restarting the process.
func (s *UpdateService) Apply(downloadURL string) error {
	dl := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "dnstester/"+s.currentVersion)

	resp, err := dl.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", resp.Status)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".dnstester-update-")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmpName, exe); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// platformSuffix returns the suffix used in release asset filenames for the
// current OS/arch, matching the naming convention in scripts/build.sh.
func platformSuffix() string {
	switch runtime.GOOS {
	case "windows":
		return "windows-amd64"
	case "linux":
		switch runtime.GOARCH {
		case "arm":
			return "linux-armhf"
		case "arm64":
			return "linux-arm64"
		default:
			return "linux-amd64"
		}
	default:
		return runtime.GOOS + "-" + runtime.GOARCH
	}
}
