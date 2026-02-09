package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	planInput  string
	planOutput string
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate a migration plan from scan output",
	Long: `Generate a migration plan that maps NGINX Ingress Controller resources
to their Gateway API equivalents. The plan can be reviewed before applying.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Generating migration plan...")
		return nil
	},
}

func init() {
	planCmd.Flags().StringVarP(&planInput, "input", "i", "scan-results.yaml", "input file from scan results")
	planCmd.Flags().StringVarP(&planOutput, "output", "o", "migration-plan.yaml", "output file for migration plan")
}
