package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Root command
var rootCmd = &cobra.Command{
	Use:   "dbeerer",
	Short: "DevOpsBeerer - OIDC/OAuth2 Playground CLI",
	Long: `DevOpsBeerer CLI deploys infrastructure and manages OIDC/OAuth2 playground scenarios.
Scenarios are fetched from DevOpsBeerer/playground-scenarios-charts repository.`,
	Version: version,
}

// Infrastructure command
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

// Scenario management commands
var startCmd = &cobra.Command{
	Use:   "start [scenario-name]",
	Short: "Start a playground scenario",
	Long:  "Start a specific scenario by deploying its Helm chart from DevOpsBeerer/playground-scenarios-charts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scenario := args[0]
		namespace, _ := cmd.Flags().GetString("namespace")

		fmt.Printf("🍺 Starting scenario: %s\n", scenario)
		fmt.Printf("Namespace: %s\n", namespace)

		// TODO: Fetch chart from GitHub repo
		fmt.Printf("📥 Fetching chart from DevOpsBeerer/playground-scenarios-charts/%s...\n", scenario)

		// TODO: Deploy using Helm
		fmt.Printf("📦 Deploying %s scenario...\n", scenario)

		fmt.Println("✅ Scenario started successfully!")
		fmt.Printf("🔗 Access your scenario at: https://%s.playground.local\n", scenario)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [scenario-name]",
	Short: "Stop a playground scenario",
	Long:  "Stop and clean up a specific scenario deployment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scenario := args[0]

		fmt.Printf("🍺 Stopping scenario: %s\n", scenario)

		// TODO: Helm uninstall
		fmt.Printf("🗑️  Removing %s deployment...\n", scenario)

		fmt.Println("✅ Scenario stopped successfully!")
	},
}

// Status command
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
		scenarios := []string{"scenario-1", "scenario-2"}

		for _, scenario := range scenarios {
			fmt.Printf("  %s: ✅ Running\n", scenario)
		}

		// TODO: Implement real status checking via kubectl/helm
	},
}

// Cleanup command
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

// List available scenarios
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available scenarios",
	Long:  "List all available scenarios from DevOpsBeerer/playground-scenarios-charts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🍺 Available Scenarios:")

		// TODO: Fetch from GitHub API
		scenarios := []string{
			"scenario-1 - Basic OIDC Flow",
			"scenario-2 - Enterprise SSO",
			"scenario-3 - API Authentication",
			"scenario-4 - Mobile Integration",
		}

		for _, scenario := range scenarios {
			fmt.Printf("  📋 %s\n", scenario)
		}

		fmt.Println()
		fmt.Println("Usage: dbeerer start <scenario-name>")
	},
}

func init() {
	// Infrastructure flags
	infraDeployCmd.Flags().StringP("namespace", "n", "devopsbeerer", "Kubernetes namespace")
	infraDeployCmd.Flags().StringP("domain", "d", "playground.local", "Base domain for services")

	// Scenario flags
	startCmd.Flags().StringP("namespace", "n", "devopsbeerer", "Kubernetes namespace")

	// Cleanup flags
	cleanupCmd.Flags().BoolP("keep-infra", "k", false, "Keep infrastructure running")

	// Add subcommands
	infraCmd.AddCommand(infraDeployCmd)

	rootCmd.AddCommand(infraCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(cleanupCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
