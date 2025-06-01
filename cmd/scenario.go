package cmd

import (
	"fmt"

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
		scenarioManager, err := scenarios.NewManager()
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}

		err = scenarioManager.InstallScenario(scenarioID)

		if err != nil {
			return fmt.Errorf("❌ installing scenario : %w", err)
		}

		return nil
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current playground scenario",
	Long:  "Stop and clean up the current scenario deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🍺 Stopping current scenario...\n")

		// Validate scenario exists
		scenarioManager, err := scenarios.NewManager()
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}

		scenarioManager.UninstallScenario()
		return nil
	},
}

func init() {
	// Add commands to root
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}
