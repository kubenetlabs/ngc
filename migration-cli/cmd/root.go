package cmd

import (
	"github.com/spf13/cobra"
)

var (
	kubeconfig string
	namespace  string
)

var rootCmd = &cobra.Command{
	Use:   "ngf-migrate",
	Short: "Migrate NGINX Ingress Controller resources to Gateway API",
	Long: `ngf-migrate is a CLI tool for migrating NGINX Ingress Controller
configurations to the Kubernetes Gateway API. It supports scanning existing
resources, generating migration plans, applying changes, and validating results.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "namespace to operate in (default: all namespaces)")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(validateCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
