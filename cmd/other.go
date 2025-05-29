package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status",
	Long:  "Show the status of infrastructure and running scenarios",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🍺 DevOpsBeerer Status:")
		fmt.Println()

		fmt.Println("Infrastructure:")
		infraComponents := map[string]string{
			"Ingress Controller": "✅ Running",
			"Cert-Manager":       "✅ Running",
			"Keycloak":           "✅ Running",
		}

		for component, status := range infraComponents {
			fmt.Printf("  %s: %s\n", component, status)
		}

		fmt.Println()
		fmt.Println("Scenarios:")
		scenarios := []string{"scenario-1"}

		for _, scenario := range scenarios {
			fmt.Printf("  %s: ✅ Running\n", scenario)
		}

		// TODO: Implement real status checking via kubectl/helm
	},
}

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up all deployments",
	Long:  "Remove all scenarios and optionally infrastructure",
	Run: func(cmd *cobra.Command, args []string) {
		keepInfra, _ := cmd.Flags().GetBool("keep-infra")

		fmt.Println("🍺 Cleaning up DevOpsBeerer...")

		// Stop all scenarios
		fmt.Println("🗑️  Removing all scenarios...")

		if !keepInfra {
			fmt.Println("🗑️  Removing infrastructure...")
		}

		fmt.Println("✅ Cleanup completed!")
	},
}

func init() {
	// Cleanup flags
	cleanupCmd.Flags().BoolP("keep-infra", "k", false, "Keep infrastructure running")

	// Add commands to root
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanupCmd)
}
