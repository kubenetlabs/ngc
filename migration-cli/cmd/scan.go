package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var scanOutput string

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Discover NGINX Ingress Controller resources in the cluster",
	Long: `Scan the source cluster for Ingress, VirtualServer, and TransportServer
resources managed by NGINX Ingress Controller. The discovered resources are
written to an output file for use by the plan command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Scanning cluster for NGINX Ingress Controller resources...")
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "scan-results.yaml", "output file for scan results")
}
