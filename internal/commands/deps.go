package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// DepsCmd shows dependency information
var DepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Show dependencies",
	Long: `Show dependencies in the workspace.

Without arguments, shows the workspace dependency matrix.
Use --brick to show dependencies for a specific brick.
Use --project to show dependencies for a specific project.

Examples:
  goliath deps
  goliath deps --brick user
  goliath deps --project myapp
  goliath deps --project myapp --brick user`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		brick, _ := cmd.Flags().GetString("brick")
		project, _ := cmd.Flags().GetString("project")
		swapAxes, _ := cmd.Flags().GetBool("swap-axes")

		switch {
		case brick != "" && project != "":
			return printBrickProjectDeps(ws, brick, project)
		case brick != "":
			return printBrickDeps(ws, brick)
		case project != "":
			return printProjectDeps(ws, project)
		default:
			return printWorkspaceDeps(ws, swapAxes)
		}
	},
}

func init() {
	DepsCmd.Flags().String("ws-dir", "", "Path to workspace directory")
	DepsCmd.Flags().String("brick", "", "Show dependencies for a specific brick (component or base)")
	DepsCmd.Flags().String("project", "", "Show dependencies for a specific project")
	DepsCmd.Flags().Bool("swap-axes", false, "Swap the x and y axes of the diagram")
	DepsCmd.Flags().String("out", "", "Write output to file")
}

// brickDeps computes which bricks a given brick depends on and which depend on it
type brickDeps struct {
	Name      string
	Interface string
	IsBase    bool
	Uses      []string // bricks this brick uses
	UsedBy    []string // bricks that use this brick
}

// computeBrickDeps builds dependency information for all bricks
func computeBrickDeps(ws *workspace.Workspace) map[string]*brickDeps {
	result := make(map[string]*brickDeps)

	// Initialize all bricks
	for _, comp := range ws.Components {
		result[comp.Name] = &brickDeps{
			Name:      comp.Name,
			Interface: comp.Interface,
			IsBase:    false,
		}
	}
	for _, base := range ws.Bases {
		result[base.Name] = &brickDeps{
			Name:   base.Name,
			IsBase: true,
		}
	}

	// Compute dependencies from imports
	for _, comp := range ws.Components {
		for _, imp := range comp.Imports {
			for _, otherComp := range ws.Components {
				if otherComp.Name == comp.Name {
					continue
				}
				if workspace.ImportBelongsToComponent(imp, otherComp.Module) {
					result[comp.Name].Uses = appendUnique(result[comp.Name].Uses, otherComp.Name)
					result[otherComp.Name].UsedBy = appendUnique(result[otherComp.Name].UsedBy, comp.Name)
				}
			}
			// Check if base imports another component (base -> component)
		}
	}

	for _, base := range ws.Bases {
		for _, imp := range base.Imports {
			for _, comp := range ws.Components {
				if workspace.ImportBelongsToComponent(imp, comp.Module) {
					result[base.Name].Uses = appendUnique(result[base.Name].Uses, comp.Name)
					result[comp.Name].UsedBy = appendUnique(result[comp.Name].UsedBy, base.Name)
				}
			}
		}
	}

	// Sort
	for _, bd := range result {
		sort.Strings(bd.Uses)
		sort.Strings(bd.UsedBy)
	}

	return result
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// printWorkspaceDeps shows the full workspace dependency matrix
func printWorkspaceDeps(ws *workspace.Workspace, swapAxes bool) error {
	deps := computeBrickDeps(ws)

	// Get sorted brick names
	var names []string
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Println("No bricks found in workspace.")
		return nil
	}

	printDepsMatrix(names, deps)

	_ = swapAxes // TODO: implement axis swapping
	return nil
}

// printDepsMatrix renders a dependency matrix with a vertical header. Each
// column name prints one rune per line, bottom-aligned, one rune wide, with a
// single space between columns — so long brick names make the matrix taller,
// never wider.
func printDepsMatrix(names []string, deps map[string]*brickDeps) {
	// Header block height = the longest brick name. Columns are one rune wide
	// plus a single separating space.
	height := 0
	for _, n := range names {
		if len(n) > height {
			height = len(n)
		}
	}
	const colW = 2 // one rune of the name + one separating space

	// Row-label gutter: as wide as the longest name plus one space (at least 5,
	// to fit the "brick" label) so long names don't push the matrix sideways.
	gutter := 5
	for _, n := range names {
		if len(n)+1 > gutter {
			gutter = len(n) + 1
		}
	}

	// Vertical header, bottom-aligned: the top lines are blank for shorter
	// names, and the "brick" label occupies the gutter on the bottom line.
	for i := 0; i < height; i++ {
		// i == 0 is the top line; the bottom line is i == height-1, where each
		// name prints its final rune.
		label := ""
		if i == height-1 {
			label = "brick"
		}
		fmt.Printf("%-*s", gutter, label)
		for _, n := range names {
			cell := " "
			// Bottom-aligned: name character index from the bottom.
			fromBottom := height - 1 - i
			if fromBottom < len(n) {
				cell = string(n[len(n)-1-fromBottom])
			}
			fmt.Printf("%-*s", colW-1, cell)
			fmt.Print(" ")
		}
		fmt.Println()
	}
	fmt.Println(strings.Repeat("-", gutter-1+len(names)*colW))

	// Data rows: one cell per column, one rune wide ("."/"x"/blank) + one
	// space, so the cells line up under the single-rune header columns.
	for _, rowName := range names {
		row := deps[rowName]
		fmt.Printf("%-*s", gutter, rowName)
		for _, colName := range names {
			if rowName == colName {
				fmt.Print(".")
			} else {
				marker := " "
				// Check if row uses col
				for _, u := range row.Uses {
					if u == colName {
						marker = "x"
						break
					}
				}
				fmt.Print(marker)
			}
			fmt.Print(" ")
		}
		fmt.Println()
	}
}

// printBrickDeps shows dependencies for a specific brick
func printBrickDeps(ws *workspace.Workspace, brickName string) error {
	deps := computeBrickDeps(ws)

	bd, ok := deps[brickName]
	if !ok {
		return fmt.Errorf("brick %q not found", brickName)
	}

	fmt.Printf("  used by  <  %s  >  uses\n", brickName)
	fmt.Printf("  -------              ----\n")

	// Print used-by and uses side by side
	maxRows := len(bd.UsedBy)
	if len(bd.Uses) > maxRows {
		maxRows = len(bd.Uses)
	}

	for i := 0; i < maxRows; i++ {
		usedBy := ""
		if i < len(bd.UsedBy) {
			usedBy = bd.UsedBy[i]
		}
		uses := ""
		if i < len(bd.Uses) {
			uses = bd.Uses[i]
		}
		fmt.Printf("  %-8s          %s\n", usedBy, uses)
	}

	return nil
}

// printProjectDeps shows dependencies within a specific project
func printProjectDeps(ws *workspace.Workspace, projectName string) error {
	// Resolve alias
	proj, ok := ws.Projects[projectName]
	if !ok {
		// Try to find by alias
		for _, p := range ws.Projects {
			if p.Alias == projectName {
				proj = p
				ok = true
				break
			}
		}
	}
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	deps := computeBrickDeps(ws)

	// Determine which bricks are in this project
	inProject := make(map[string]bool)
	for _, req := range proj.Requires {
		for _, comp := range ws.Components {
			if comp.Module == req {
				inProject[comp.Name] = true
			}
		}
		for _, base := range ws.Bases {
			if base.Module == req {
				inProject[base.Name] = true
			}
		}
	}

	// If no requires found, include all bricks (for development)
	if len(inProject) == 0 && proj.Name == "development" {
		for name := range deps {
			inProject[name] = true
		}
	}

	// Get sorted brick names in project
	var names []string
	for name := range inProject {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Printf("Project %q has no bricks.\n", proj.Name)
		return nil
	}

	// Print matrix (same vertical header as the workspace matrix)
	printDepsMatrix(names, deps)

	return nil
}

// printBrickProjectDeps shows dependencies for a brick within a project context
func printBrickProjectDeps(ws *workspace.Workspace, brickName, projectName string) error {
	// For now, delegate to brick deps with project context noted
	fmt.Printf("Dependencies for brick %q in project %q:\n\n", brickName, projectName)
	return printBrickDeps(ws, brickName)
}
