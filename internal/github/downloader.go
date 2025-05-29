package github

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	GitHubAPIURL   = "https://api.github.com"
	RepoOwner      = "DevOpsBeerer"
	RepoName       = "playground-scenarios-charts"
	RequestTimeout = 30 * time.Second
)

// Downloader handles downloading Helm charts from GitHub
type Downloader struct {
	httpClient *http.Client
}

// NewDownloader creates a new GitHub downloader
func NewDownloader() *Downloader {
	return &Downloader{
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// DownloadChart downloads a specific scenario chart from GitHub
func (d *Downloader) DownloadChart(scenarioID, destPath string) error {
	fmt.Printf("📥 Downloading chart for scenario: %s\n", scenarioID)

	// Download the entire repository as a tarball
	tarballURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.tar.gz", RepoOwner, RepoName)

	// Download tarball
	resp, err := d.httpClient.Get(tarballURL)
	if err != nil {
		return fmt.Errorf("failed to download repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download repository: HTTP %d", resp.StatusCode)
	}

	// Extract the specific scenario directory
	if err := d.extractScenario(resp.Body, scenarioID, destPath); err != nil {
		return fmt.Errorf("failed to extract scenario: %w", err)
	}

	fmt.Printf("✅ Chart downloaded successfully to %s\n", destPath)
	return nil
}

// extractScenario extracts only the specified scenario from the tarball
func (d *Downloader) extractScenario(reader io.Reader, scenarioID, destPath string) error {
	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Expected prefix in the tarball (GitHub adds repo name prefix)
	expectedPrefix := fmt.Sprintf("%s-main/%s/", RepoName, scenarioID)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Check if this file belongs to our scenario
		if !strings.HasPrefix(header.Name, expectedPrefix) {
			continue
		}

		// Calculate relative path within the scenario
		relativePath := strings.TrimPrefix(header.Name, expectedPrefix)
		if relativePath == "" {
			continue // Skip the directory itself
		}

		targetPath := filepath.Join(destPath, relativePath)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create file
			if err := d.extractFile(tarReader, targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}
			fmt.Printf("📄 Extracted: %s\n", relativePath)
		}
	}

	return nil
}

// extractFile extracts a single file from the tar reader
func (d *Downloader) extractFile(tarReader *tar.Reader, targetPath string, mode os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for file: %w", err)
	}

	// Create file
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, tarReader); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}

// ListScenarios lists all available scenarios from the repository
func (d *Downloader) ListScenarios() ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents", GitHubAPIURL, RepoOwner, RepoName)

	resp, err := d.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repository contents: HTTP %d", resp.StatusCode)
	}

	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to parse repository contents: %w", err)
	}

	var scenarios []string
	for _, item := range contents {
		if item.Type == "dir" && strings.HasPrefix(item.Name, "scenario-") {
			scenarios = append(scenarios, item.Name)
		}
	}

	return scenarios, nil
}
