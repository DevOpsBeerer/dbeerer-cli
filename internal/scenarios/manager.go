package scenarios

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	MetadataURL    = "https://raw.githubusercontent.com/DevOpsBeerer/playground-scenarios-charts/refs/heads/main/metadata.json"
	RequestTimeout = 10 * time.Second
	ReleaseName    = "devopsbeerer-scenario"
	ChartBaseURL   = "https://raw.githubusercontent.com/DevOpsBeerer/playground-scenarios-charts/refs/heads/main"
)

// Scenario represents a single scenario from metadata
type Scenario struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Metadata represents the structure of metadata.json
type Metadata struct {
	Scenarios []Scenario `json:"scenarios"`
}

// Manager handles scenario operations
type Manager struct {
	httpClient *http.Client
	settings   *cli.EnvSettings
	namespace  string
}

// NewManager creates a new scenario manager
func NewManager(namespace string) *Manager {
	settings := cli.New()
	settings.SetNamespace(namespace)

	return &Manager{
		settings:  settings,
		namespace: namespace,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// InstallScenario installs a scenario using Helm
func (m *Manager) InstallScenario(scenarioID string) error {
	fmt.Printf("🔍 Checking for existing scenario deployment...\n")

	// Remove existing deployment if it exists
	if err := m.UninstallScenario(); err != nil {
		// Log but don't fail if uninstall fails (might not exist)
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	fmt.Printf("📥 Downloading chart for scenario: %s\n", scenarioID)

	// Download chart from GitHub
	chartPath, err := m.downloadChart(scenarioID)
	if err != nil {
		return fmt.Errorf("failed to download chart: %w", err)
	}
	defer os.RemoveAll(chartPath) // Clean up temp files

	fmt.Printf("📦 Installing scenario via Helm...\n")

	// Install the chart
	if err := m.installChart(chartPath); err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// UninstallScenario removes the current scenario deployment
func (m *Manager) UninstallScenario() error {
	actionConfig := new(action.Configuration)

	// Initialize Helm action configuration
	if err := actionConfig.Init(
		&genericclioptions.ConfigFlags{Namespace: &m.namespace},
		m.namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			// Silent debug function
		},
	); err != nil {
		return fmt.Errorf("failed to initialize Helm config: %w", err)
	}

	// Create uninstall action
	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = true
	uninstall.Timeout = 5 * time.Minute

	// Check if release exists
	_, err := actionConfig.Releases.Last(ReleaseName)
	if err != nil {
		return fmt.Errorf("release '%s' not found", ReleaseName)
	}

	fmt.Printf("🗑️  Removing existing scenario deployment...\n")

	// Uninstall the release
	_, err = uninstall.Run(ReleaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	fmt.Printf("✅ Previous scenario removed successfully\n")
	return nil
}

// downloadChart downloads the Helm chart from GitHub
func (m *Manager) downloadChart(scenarioID string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "devopsbeerer-chart-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	chartURL := fmt.Sprintf("%s/%s", ChartBaseURL, scenarioID)

	// Use Helm's downloader to fetch the chart

	// Download chart to temp directory
	chartPath := filepath.Join(tempDir, scenarioID)
	if err := os.MkdirAll(chartPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create chart directory: %w", err)
	}

	// For GitHub raw content, we need to download files individually
	// This is a simplified approach - in production you might want to use git clone
	if err := m.downloadChartFiles(chartURL, chartPath); err != nil {
		return "", fmt.Errorf("failed to download chart files: %w", err)
	}

	return chartPath, nil
}

// downloadChartFiles downloads individual chart files from GitHub
func (m *Manager) downloadChartFiles(baseURL, chartPath string) error {
	// Required Helm chart files
	files := []string{"Chart.yaml", "values.yaml"}

	getter, err := getter.NewHTTPGetter()

	if err != nil {
		fmt.Printf("Error")
	}

	for _, file := range files {
		url := fmt.Sprintf("%s/%s", baseURL, file)
		filePath := filepath.Join(chartPath, file)

		fmt.Printf("📥 Downloading %s...\n", file)

		// Download file
		resp, err := getter.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", file, err)
		}

		// Write to file
		if err := os.WriteFile(filePath, resp.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	// Download templates directory (assuming it exists)
	templatesDir := filepath.Join(chartPath, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// TODO: Download template files - this would need to be enhanced
	// to discover and download all template files from the GitHub repository

	return nil
}

// installChart installs the downloaded Helm chart
func (m *Manager) installChart(chartPath string) error {
	actionConfig := new(action.Configuration)

	// Initialize Helm action configuration
	if err := actionConfig.Init(
		&genericclioptions.ConfigFlags{Namespace: &m.namespace},
		m.namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			fmt.Printf(format, v...)
		},
	); err != nil {
		return fmt.Errorf("failed to initialize Helm config: %w", err)
	}

	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Create install action
	install := action.NewInstall(actionConfig)
	install.ReleaseName = ReleaseName
	install.Namespace = m.namespace
	install.Wait = true
	install.Timeout = 10 * time.Minute
	install.CreateNamespace = true

	// Install the chart
	_, err = install.Run(chart, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// GetScenarioStatus checks if a scenario is currently deployed
func (m *Manager) GetScenarioStatus() (bool, string, error) {
	actionConfig := new(action.Configuration)

	// Initialize Helm action configuration
	if err := actionConfig.Init(
		&genericclioptions.ConfigFlags{Namespace: &m.namespace},
		m.namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			// Silent debug function
		},
	); err != nil {
		return false, "", fmt.Errorf("failed to initialize Helm config: %w", err)
	}

	// Check if release exists
	release, err := actionConfig.Releases.Last(ReleaseName)
	if err != nil {
		return false, "", nil // Release doesn't exist
	}

	return true, release.Info.Status.String(), nil
}

// ListScenarios fetches and returns all available scenarios
func (m *Manager) ListScenarios() ([]Scenario, error) {
	resp, err := m.httpClient.Get(MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metadata: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return metadata.Scenarios, nil
}

// FindScenario finds a scenario by ID
func (m *Manager) FindScenario(id string) (*Scenario, error) {
	scenarios, err := m.ListScenarios()
	if err != nil {
		return nil, err
	}

	for _, scenario := range scenarios {
		if scenario.ID == id {
			return &scenario, nil
		}
	}

	return nil, fmt.Errorf("scenario '%s' not found", id)
}
