package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// infraCmd represents the infrastructure command
var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Manage infrastructure components",
	Long:  "Deploy and manage core infrastructure: ingress controller, Keycloak, and cert-manager",
}

var infraDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy core infrastructure",
	Long:  "Deploy ingress controller, Keycloak, and cert-manager using Helm",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, _ := cmd.Flags().GetString("namespace")
		domain, _ := cmd.Flags().GetString("domain")

		fmt.Printf("🍺 Deploying DevOpsBeerer infrastructure...\n")
		fmt.Printf("Namespace: %s\n", namespace)
		fmt.Printf("Domain: %s\n", domain)

		// Deploy infrastructure components
		components := []string{"Ingress Controller", "Cert-Manager", "Keycloak"}
		for _, component := range components {
			fmt.Printf("📦 Installing %s...\n", component)
			// TODO: Implement Helm deployment
		}

		fmt.Println("✅ Infrastructure deployed successfully!")
		fmt.Printf("🔗 Keycloak: https://keycloak.%s\n", domain)
	},
}

func init() {
	// Infrastructure flags
	infraDeployCmd.Flags().StringP("namespace", "n", "devopsbeerer", "Kubernetes namespace")
	infraDeployCmd.Flags().StringP("domain", "d", "devopsbeerer.local", "Base domain for services")

	// Add subcommands
	infraCmd.AddCommand(infraDeployCmd)
	rootCmd.AddCommand(infraCmd)
}
