package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevOpsBeerer/dbeerer-cli/internal/github"
	"github.com/DevOpsBeerer/dbeerer-cli/internal/helm"
	"github.com/DevOpsBeerer/dbeerer-cli/internal/scenarios"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [scenario-id]",
	Short: "Start a playground scenario",
	Long:  "Start a specific scenario by deploying its Helm chart from DevOpsBeerer/playground-scenarios-charts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scenarioID := args[0]
		namespace := scenarioID

		fmt.Printf("🍺 Starting scenario: %s\n", scenarioID)
		fmt.Printf("Namespace: %s\n", namespace)

		// Validate scenario exists
		scenarioManager := scenarios.NewManager(scenarioID)
		scenario, err := scenarioManager.FindScenario(scenarioID)
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}

		fmt.Printf("📋 Scenario: %s\n", scenario.Name)
		fmt.Printf("📝 Description: %s\n", scenario.Description)
		fmt.Println()

		// Create Helm manager
		helmManager := helm.NewManager(namespace)

		// Create temporary directory for chart
		tempDir, err := os.MkdirTemp("", "devopsbeerer-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir) // Clean up

		chartPath := filepath.Join(tempDir, scenarioID)

		// Download chart from GitHub
		downloader := github.NewDownloader()
		if err := downloader.DownloadChart(scenarioID, chartPath); err != nil {
			return fmt.Errorf("failed to download chart: %w", err)
		}

		// Install scenario via Helm
		fmt.Printf("📦 Installing scenario via Helm...\n")
		if err := helmManager.InstallScenario(scenarioID, chartPath); err != nil {
			return fmt.Errorf("failed to install scenario: %w", err)
		}

		fmt.Println("✅ Scenario started successfully!")
		fmt.Printf("🔗 Access your scenario at: https://%s.devopsbeerer.local\n", scenarioID)

		return nil
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current playground scenario",
	Long:  "Stop and clean up the current scenario deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		scenarioID := args[0]

		fmt.Printf("🍺 Stopping current scenario...\n")

		// Create Helm manager
		helmManager := helm.NewManager(scenarioID)

		// Check if scenario is running
		isRunning, status, err := helmManager.GetScenarioStatus(scenarioID)
		if err != nil {
			return fmt.Errorf("failed to check scenario status: %w", err)
		}

		if !isRunning {
			fmt.Println("ℹ️  No scenario is currently running")
			return nil
		}

		fmt.Printf("📋 Current scenario status: %s\n", status)

		// Uninstall scenario
		if err := helmManager.UninstallScenario(scenarioID); err != nil {
			return fmt.Errorf("failed to stop scenario: %w", err)
		}

		fmt.Println("✅ Scenario stopped successfully!")
		return nil
	},
}

func init() {
	// Scenario flags
	stopCmd.Flags().StringP("namespace", "n", "devopsbeerer", "Kubernetes namespace")

	// Add commands to root
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}
