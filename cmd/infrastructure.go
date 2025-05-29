package cmd

import (
	"fmt"

	"github.com/DevOpsBeerer/dbeerer-cli/internal/infrastructure"

	"github.com/spf13/cobra"
)

// infraCmd represents the infrastructure command
var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Manage infrastructure components",
	Long:  "Deploy and manage core infrastructure: K3s, ingress controller, Keycloak, and cert-manager",
}

var infraDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy core infrastructure",
	Long:  "Deploy K3s cluster with ingress controller, Keycloak, and cert-manager using playground repository scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🍺 Deploying DevOpsBeerer infrastructure...\n")
		fmt.Printf("📋 This will:\n")
		fmt.Printf("   1. Clone playground repository\n")
		fmt.Printf("   2. Install K3s cluster\n")
		fmt.Printf("   3. Install cert-manager, SSO (Keycloak), and ingress controller\n")
		fmt.Println()

		// Create infrastructure manager
		manager := infrastructure.NewManager()

		// Deploy infrastructure
		if err := manager.DeployInfrastructure(); err != nil {
			return fmt.Errorf("❌ Infrastructure deployment failed: %w", err)
		}

		fmt.Println()
		fmt.Println("🎉 Infrastructure deployment completed!")
		fmt.Println("🔗 You can now start scenarios with: dbeerer start <scenario-id>")

		return nil
	},
}

var infraStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check infrastructure status",
	Long:  "Check the status of infrastructure components",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🍺 Checking infrastructure status...")

		// Create infrastructure manager
		manager := infrastructure.NewManager()

		// Check infrastructure status
		status, err := manager.CheckInfrastructure()
		if err != nil {
			return fmt.Errorf("failed to check infrastructure: %w", err)
		}

		fmt.Println()
		fmt.Printf("Kubectl Available: %s\n", getStatusIcon(status.KubectlAvailable))
		fmt.Printf("Cluster Running: %s\n", getStatusIcon(status.ClusterRunning))
		fmt.Println()
		fmt.Println("Components:")

		for component, running := range status.Components {
			fmt.Printf("  %s: %s\n", component, getStatusIcon(running))
		}

		return nil
	},
}

// getStatusIcon returns appropriate icon for status
func getStatusIcon(running bool) string {
	if running {
		return "✅ Running"
	}
	return "❌ Not Running"
}

func init() {
	// Add subcommands
	infraCmd.AddCommand(infraDeployCmd)
	infraCmd.AddCommand(infraStatusCmd)
	rootCmd.AddCommand(infraCmd)
}
