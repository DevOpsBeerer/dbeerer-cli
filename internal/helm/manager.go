package helm

import (
	"fmt"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const ()

// Manager handles Helm operations
type Manager struct {
	settings  *cli.EnvSettings
	namespace string
}

// NewManager creates a new Helm manager
func NewManager(namespace string) *Manager {
	settings := cli.New()
	settings.SetNamespace(namespace)

	return &Manager{
		settings:  settings,
		namespace: namespace,
	}
}

// InstallScenario installs a scenario using Helm
func (m *Manager) InstallScenario(scenarioID, chartPath string) error {
	fmt.Printf("🔍 Checking for existing scenario deployment...\n")

	// Remove existing deployment if it exists
	if err := m.UninstallScenario(scenarioID); err != nil {
		// Log but don't fail if uninstall fails (might not exist)
		fmt.Printf("ℹ️  No previous scenario to remove\n")
	}

	fmt.Printf("📦 Installing scenario via Helm...\n")

	// Install the chart
	if err := m.installChart(scenarioID, chartPath); err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// UninstallScenario removes the current scenario deployment
func (m *Manager) UninstallScenario(scenarioID string) error {
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
	_, err := actionConfig.Releases.Last(scenarioID)
	if err != nil {
		return fmt.Errorf("release '%s' not found", scenarioID)
	}

	fmt.Printf("🗑️  Removing existing scenario deployment...\n")

	// Uninstall the release
	_, err = uninstall.Run(scenarioID)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	fmt.Printf("✅ Previous scenario removed successfully\n")
	return nil
}

// installChart installs the downloaded Helm chart
func (m *Manager) installChart(scenarioID, chartPath string) error {
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
	install.ReleaseName = scenarioID
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
func (m *Manager) GetScenarioStatus(scenarioID string) (bool, string, error) {
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
	release, err := actionConfig.Releases.Last(scenarioID)
	if err != nil {
		return false, "", nil // Release doesn't exist
	}

	return true, release.Info.Status.String(), nil
}
