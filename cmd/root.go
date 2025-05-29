package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dbeerer",
	Short: "DevOpsBeerer - OIDC/OAuth2 Playground CLI",
	Long: `DevOpsBeerer CLI deploys infrastructure and manages OIDC/OAuth2 playground scenarios.
Scenarios are fetched from DevOpsBeerer/playground-scenarios-charts repository.`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
