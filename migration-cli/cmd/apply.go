package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
in the target cluster. Use --dry-run to preview changes without applying them.

Note: Actual cluster operations are performed through the NGF Console API.
This command reads the plan file and displays what would be applied.`,
	RunE: runApply,
}

func init() {
	applyCmd.Flags().StringVarP(&applyPlan, "plan", "p", "migration-plan.yaml", "migration plan file to apply")
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
}

// applyMapping represents a source-to-target migration mapping parsed from
// the plan file.
type applyMapping struct {
	sourceKind      string
	sourceName      string
	sourceNamespace string
	targetKind      string
	targetName      string
	targetNamespace string
	confidence      string
}

func runApply(_ *cobra.Command, _ []string) error {
	mappings, gatewayName, err := parsePlanFile(applyPlan)
	if err != nil {
		return fmt.Errorf("failed to read migration plan: %w", err)
	}

	if len(mappings) == 0 {
		fmt.Println("No mappings found in migration plan. Nothing to apply.")
		return nil
	}

	if dryRun {
		fmt.Println("=== DRY RUN — no changes will be made ===")
		fmt.Println()
	}

	// Display gateway creation.
	if gatewayName != "" {
		fmt.Printf("Gateway: %s (will be created)\n\n", gatewayName)
	}

	fmt.Printf("Resources to apply (%d mapping(s)):\n\n", len(mappings))
	for i, m := range mappings {
		fmt.Printf("  %d. %s/%s (%s)\n", i+1, m.sourceNamespace, m.sourceName, m.sourceKind)
		fmt.Printf("     -> %s/%s (%s)  [confidence: %s]\n", m.targetNamespace, m.targetName, m.targetKind, m.confidence)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run complete. No resources were modified.")
		fmt.Println("Remove --dry-run to apply, or use the NGF Console API for cluster operations.")
		return nil
	}

	fmt.Println("WARNING: Direct cluster operations require the NGF Console API server.")
	fmt.Println("To apply this plan to a live cluster, use one of the following methods:")
	fmt.Println()
	fmt.Println("  1. NGF Console UI:  Upload the plan file in the Migration section")
	fmt.Println("  2. NGF Console API: POST the plan to /api/v1/migration/apply")
	fmt.Println()
	fmt.Printf("  Example:\n")
	fmt.Printf("    curl -X POST http://localhost:8080/api/v1/migration/apply \\\n")
	fmt.Printf("      -H 'Content-Type: application/json' \\\n")
	fmt.Printf("      -d '{\"importId\": \"<id>\", \"dryRun\": false, \"resources\": [...]}'\n")
	fmt.Println()
	fmt.Println("Plan file:", applyPlan)

	return nil
}

// parsePlanFile reads a migration plan YAML and extracts mappings and the
// gateway name. Uses simple line-based parsing (no YAML library).
func parsePlanFile(path string) ([]applyMapping, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	var mappings []applyMapping
	scanner := bufio.NewScanner(f)

	var gatewayName string

	// Track parser state.
	const (
		sNone = iota
		sGateway
		sMappings
		sSource
		sTarget
	)
	state := sNone
	var current applyMapping
	currentConfidence := ""

	flush := func() {
		if current.sourceKind != "" || current.targetKind != "" {
			current.confidence = currentConfidence
			mappings = append(mappings, current)
		}
		current = applyMapping{}
		currentConfidence = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		// Detect top-level sections.
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if trimmed == "gateway:" {
				state = sGateway
				continue
			}
			if trimmed == "mappings:" {
				if state == sMappings || state == sSource || state == sTarget {
					flush()
				}
				state = sMappings
				continue
			}
		}

		if state == sGateway {
			if strings.HasPrefix(trimmed, "name:") {
				gatewayName = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
				gatewayName = strings.Trim(gatewayName, "\"'")
			}
			continue
		}

		if state == sMappings || state == sSource || state == sTarget {
			// New list item.
			if strings.HasPrefix(trimmed, "- ") {
				flush()
				rest := strings.TrimPrefix(trimmed, "- ")
				if rest == "source:" {
					state = sSource
				}
				continue
			}

			if trimmed == "source:" {
				state = sSource
				continue
			}
			if trimmed == "target:" {
				state = sTarget
				continue
			}

			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")

			if key == "confidence" {
				currentConfidence = val
				continue
			}

			if state == sSource {
				switch key {
				case "kind":
					current.sourceKind = val
				case "name":
					current.sourceName = val
				case "namespace":
					current.sourceNamespace = val
				}
			} else if state == sTarget {
				switch key {
				case "kind":
					current.targetKind = val
				case "name":
					current.targetName = val
				case "namespace":
					current.targetNamespace = val
				}
			}
		}
	}

	flush()

	if err := scanner.Err(); err != nil {
		return mappings, gatewayName, err
	}

	return mappings, gatewayName, nil
}
