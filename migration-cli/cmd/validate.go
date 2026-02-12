package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validatePlan string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate migration results",
	Long: `Validate that the migrated Gateway API resources are correctly configured
and functioning in the target cluster.

This command prints a validation checklist and instructions for verifying
the migration. Full automated validation requires the NGF Console API.`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&validatePlan, "plan", "p", "migration-plan.yaml", "migration plan file to validate against")
}

func runValidate(_ *cobra.Command, _ []string) error {
	fmt.Println("=== Migration Validation Checklist ===")
	fmt.Println()

	checks := []struct {
		name        string
		description string
		command     string
	}{
		{
			name:        "Gateway Status",
			description: "Verify the Gateway resource is programmed and has an address",
			command:     "kubectl get gateway migrated-gateway -o wide",
		},
		{
			name:        "Route Status",
			description: "Verify HTTPRoutes are accepted by the Gateway",
			command:     "kubectl get httproutes -o wide",
		},
		{
			name:        "Backend Health",
			description: "Verify backend services are running and have ready endpoints",
			command:     "kubectl get endpoints -l app.kubernetes.io/managed-by=ngf-migrate",
		},
		{
			name:        "TLS Certificates",
			description: "Verify TLS secrets exist and are not expired",
			command:     "kubectl get secrets -l migration.ngf.io/tls=true",
		},
		{
			name:        "DNS Resolution",
			description: "Verify hostnames resolve to the Gateway address",
			command:     "kubectl get gateway migrated-gateway -o jsonpath='{.status.addresses[0].value}'",
		},
		{
			name:        "Traffic Test",
			description: "Send a test request through the migrated routes",
			command:     "curl -v http://<gateway-address>/",
		},
		{
			name:        "Old Resources",
			description: "Check that old Ingress/VirtualServer resources can be safely removed",
			command:     "kubectl get ingress,virtualservers --all-namespaces",
		},
	}

	for i, c := range checks {
		fmt.Printf("  %d. [  ] %s\n", i+1, c.name)
		fmt.Printf("       %s\n", c.description)
		fmt.Printf("       $ %s\n", c.command)
		fmt.Println()
	}

	fmt.Println("---")
	fmt.Println()
	fmt.Println("For automated validation, use the NGF Console API:")
	fmt.Println()
	fmt.Printf("  curl -X POST http://localhost:8080/api/v1/migration/validate \\\n")
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"importId\": \"<id>\"}'\n")
	fmt.Println()
	fmt.Println("Or use the NGF Console UI Migration section for a visual validation report.")

	return nil
}
