package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	applyPlan string
	dryRun    bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a migration plan to the target cluster",
	Long: `Apply the generated migration plan to create Gateway API resources
in the target cluster. Use --dry-run to preview changes without applying them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Applying migration plan...")
		return nil
	},
}

func init() {
	applyCmd.Flags().StringVarP(&applyPlan, "plan", "p", "migration-plan.yaml", "migration plan file to apply")
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
}
