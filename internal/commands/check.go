package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// Severity represents the severity of a finding
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

// Finding represents a validation finding
type Finding struct {
	Code     int
	Severity Severity
	Message  string
	Brick    string
}

func (f Finding) String() string {
	prefix := fmt.Sprintf("%s %d", f.Severity, f.Code)
	if f.Brick != "" {
		prefix += fmt.Sprintf(" [%s]", f.Brick)
	}
	return fmt.Sprintf("%s: %s", prefix, f.Message)
}

// CheckResult holds all findings from validation
type CheckResult struct {
	Findings []Finding
}

// HasErrors returns true if there are any error-severity findings
func (r *CheckResult) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// OK returns true if there are no findings at all
func (r *CheckResult) OK() bool {
	return len(r.Findings) == 0
}

// CheckCmd validates the workspace
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if the workspace is valid",
	Long: `Validate the workspace structure and dependencies.

Checks include:
  - Error 101: Illegal dependency on non-interface namespace
  - Error 104: Circular dependencies between components
  - Error 105: Illegal name sharing (base same name as component)
  - Error 106: Multiple components with same interface in a project
  - Error 107: Missing components for interface in project

Example:
  goliath check`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		result := CheckWorkspace(ws)

		if result.OK() {
			fmt.Println("OK")
			return nil
		}

		for _, f := range result.Findings {
			fmt.Println(f.String())
		}

		if result.HasErrors() {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	CheckCmd.Flags().String("ws-dir", "", "Path to workspace directory")
}

// CheckWorkspace runs all validation rules on the workspace
func CheckWorkspace(ws *workspace.Workspace) *CheckResult {
	result := &CheckResult{}

	// Rule 105: Name collision between bases and components
	result.Findings = append(result.Findings, checkNameCollisions(ws)...)

	// Rule 101: Illegal dependency on non-API namespace
	result.Findings = append(result.Findings, checkIllegalDependencies(ws)...)

	// Rule 104: Circular dependencies
	result.Findings = append(result.Findings, checkCircularDependencies(ws)...)

	// Rule 106: Multiple components with same interface
	result.Findings = append(result.Findings, checkDuplicateInterfaces(ws)...)

	return result
}

// checkNameCollisions checks for bases with the same name as components (Error 105)
func checkNameCollisions(ws *workspace.Workspace) []Finding {
	var findings []Finding
	for baseName := range ws.Bases {
		if _, exists := ws.Components[baseName]; exists {
			findings = append(findings, Finding{
				Code:     105,
				Severity: SeverityError,
				Message:  fmt.Sprintf("base %q has the same name as a component", baseName),
				Brick:    baseName,
			})
		}
	}
	return findings
}

// checkIllegalDependencies checks for imports of non-API packages from other components (Error 101)
func checkIllegalDependencies(ws *workspace.Workspace) []Finding {
	var findings []Finding
	interfacePkg := ws.Config.InterfacePkg
	if interfacePkg == "" {
		interfacePkg = "api"
	}

	for _, comp := range ws.Components {
		for _, imp := range comp.Imports {
			// Check if import belongs to another component
			for _, otherComp := range ws.Components {
				if otherComp.Name == comp.Name {
					continue // Skip self
				}
				if workspace.ImportBelongsToComponent(imp, otherComp.Module) {
					// Check if it's the API package
					if !workspace.ImportIsComponentAPI(imp, otherComp.Module, interfacePkg) {
						findings = append(findings, Finding{
							Code:     101,
							Severity: SeverityError,
							Message: fmt.Sprintf(
								"illegal dependency on namespace %q — only the %q package of component %q may be imported",
								imp, interfacePkg, otherComp.Name,
							),
							Brick: comp.Name,
						})
					}
				}
			}
		}
	}

	// Also check bases
	for _, base := range ws.Bases {
		for _, imp := range base.Imports {
			for _, comp := range ws.Components {
				if workspace.ImportBelongsToComponent(imp, comp.Module) {
					if !workspace.ImportIsComponentAPI(imp, comp.Module, interfacePkg) {
						findings = append(findings, Finding{
							Code:     101,
							Severity: SeverityError,
							Message: fmt.Sprintf(
								"illegal dependency on namespace %q — only the %q package of component %q may be imported",
								imp, interfacePkg, comp.Name,
							),
							Brick: base.Name,
						})
					}
				}
			}
		}
	}

	return findings
}

// checkCircularDependencies detects circular dependencies between components (Error 104)
func checkCircularDependencies(ws *workspace.Workspace) []Finding {
	var findings []Finding

	// Build dependency map: component -> set of components it depends on
	deps := make(map[string]map[string]bool)
	for _, comp := range ws.Components {
		deps[comp.Name] = make(map[string]bool)
		for _, imp := range comp.Imports {
			for _, otherComp := range ws.Components {
				if otherComp.Name == comp.Name {
					continue
				}
				if workspace.ImportBelongsToComponent(imp, otherComp.Module) {
					deps[comp.Name][otherComp.Name] = true
				}
			}
		}
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var detectCycle func(node string, path []string) []string
	detectCycle = func(node string, path []string) []string {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for dep := range deps[node] {
			if !visited[dep] {
				if cycle := detectCycle(dep, path); cycle != nil {
					return cycle
				}
			} else if recStack[dep] {
				// Found a cycle
				cycleStart := 0
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				return append(path[cycleStart:], dep)
			}
		}

		recStack[node] = false
		return nil
	}

	checked := make(map[string]bool)
	for _, name := range ws.ComponentNames() {
		if !checked[name] {
			if cycle := detectCycle(name, nil); cycle != nil {
				findings = append(findings, Finding{
					Code:     104,
					Severity: SeverityError,
					Message:  fmt.Sprintf("circular dependency detected: %s", strings.Join(cycle, " → ")),
					Brick:    cycle[0],
				})
			}
			// Mark all visited nodes as checked
			for n := range visited {
				checked[n] = true
			}
		}
	}

	return findings
}

// checkDuplicateInterfaces checks for multiple components with the same interface (Error 106)
// This is only an error in projects, not in the workspace itself
func checkDuplicateInterfaces(ws *workspace.Workspace) []Finding {
	var findings []Finding

	// Group components by interface
	ifaceComps := make(map[string][]string)
	for _, comp := range ws.Components {
		ifaceComps[comp.Interface] = append(ifaceComps[comp.Interface], comp.Name)
	}

	// For each project, check if it includes multiple components with the same interface
	for _, projName := range ws.ProjectNames() {
		if projName == "development" {
			continue // Development can have multiple implementations
		}
		proj := ws.Projects[projName]

		for iface, comps := range ifaceComps {
			if len(comps) < 2 {
				continue
			}

			// Count how many of these components are in the project
			included := 0
			for _, compName := range comps {
				comp := ws.Components[compName]
				for _, req := range proj.Requires {
					if req == comp.Module {
						included++
						break
					}
				}
			}

			if included > 1 {
				findings = append(findings, Finding{
					Code:     106,
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"project %q contains multiple components (%s) that share interface %q",
						projName, strings.Join(comps, ", "), iface,
					),
					Brick: projName,
				})
			}
		}
	}

	return findings
}
