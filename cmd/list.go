package cmd

import (
	"fmt"

	"github.com/DevOpsBeerer/dbeerer-cli/internal/scenarios"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available scenarios",
	Long:  "List all available scenarios from DevOpsBeerer/playground-scenarios-charts",
	RunE:  runListCommand,
}

func runListCommand(cmd *cobra.Command, args []string) error {
	fmt.Println("🍺 Fetching available scenarios...")

	// Create scenario manager
	manager := scenarios.NewManager("")

	// Fetch scenarios from GitHub
	scenarioList, err := manager.ListScenarios()
	if err != nil {
		return fmt.Errorf("failed to fetch scenarios: %w", err)
	}

	if len(scenarioList) == 0 {
		fmt.Println("❌ No scenarios found")
		return nil
	}

	fmt.Printf("\n🍺 Available Scenarios (%d found):\n\n", len(scenarioList))

	// Display scenarios
	for _, scenario := range scenarioList {
		fmt.Printf("  📋 %s (%s)\n", scenario.Name, scenario.ID)
		fmt.Printf("     %s\n\n", scenario.Description)
	}

	fmt.Println("Usage: dbeerer start <scenario-id>")
	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}
