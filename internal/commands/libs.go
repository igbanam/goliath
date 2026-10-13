package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// LibsCmd shows all library dependencies
var LibsCmd = &cobra.Command{
	Use:   "libs",
	Short: "Show all libraries used in the workspace",
	Long: `Show all Go module dependencies used across the workspace.

Displays which projects use each library and flags version inconsistencies.

Examples:
  goliath libs
  goliath libs --compact`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		compact, _ := cmd.Flags().GetBool("compact")
		out, _ := cmd.Flags().GetString("out")

		libs := collectLibraries(ws)

		var output string
		if compact {
			output = formatLibsCompact(libs, ws)
		} else {
			output = formatLibs(libs, ws)
		}

		if out != "" {
			if err := os.WriteFile(out, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Output written to %s\n", out)
		} else {
			fmt.Print(output)
		}

		return nil
	},
}

func init() {
	LibsCmd.Flags().String("ws-dir", "", "Path to workspace directory")
	LibsCmd.Flags().Bool("compact", false, "Show compact output")
	LibsCmd.Flags().String("out", "", "Write output to file")
}

// libInfo holds information about a library
type libInfo struct {
	Name     string
	Version  string
	UsedBy   map[string]bool // project names
	InBrick  map[string]bool // brick names (components/bases)
}

// collectLibraries scans all go.mod files and collects library dependencies
func collectLibraries(ws *workspace.Workspace) map[string]*libInfo {
	libs := make(map[string]*libInfo)

	addLib := func(name, version, project, brick string) {
		key := name
		if version != "" {
			key = name + "@" + version
		}
		lib, ok := libs[key]
		if !ok {
			lib = &libInfo{
				Name:    name,
				Version: version,
				UsedBy:  make(map[string]bool),
				InBrick: make(map[string]bool),
			}
			libs[key] = lib
		}
		if project != "" {
			lib.UsedBy[project] = true
		}
		if brick != "" {
			lib.InBrick[brick] = true
		}
	}

	// Scan all go.mod files
	dirsToScan := []struct {
		dir     string
		project string
		brick   string
	}{
		{filepath.Join(ws.Dir, "development"), "development", ""},
	}

	for _, comp := range ws.Components {
		dirsToScan = append(dirsToScan, struct {
			dir     string
			project string
			brick   string
		}{comp.Dir, "", comp.Name})
	}
	for _, base := range ws.Bases {
		dirsToScan = append(dirsToScan, struct {
			dir     string
			project string
			brick   string
		}{base.Dir, "", base.Name})
	}
	for name, proj := range ws.Projects {
		if name == "development" {
			continue
		}
		dirsToScan = append(dirsToScan, struct {
			dir     string
			project string
			brick   string
		}{proj.Dir, name, ""})
	}

	requireRe := regexp.MustCompile(`^\s*(\S+)\s+v([\d][\w.\-+]*)`)

	for _, scan := range dirsToScan {
		goModPath := filepath.Join(scan.dir, "go.mod")
		content, err := os.ReadFile(goModPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		inRequire := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			if strings.HasSuffix(trimmed, "(") && strings.HasPrefix(trimmed, "require") {
				inRequire = true
				continue
			}
			if trimmed == ")" {
				inRequire = false
				continue
			}

			// Single-line require
			if strings.HasPrefix(trimmed, "require ") && !inRequire {
				rest := strings.TrimPrefix(trimmed, "require ")
				if m := requireRe.FindStringSubmatch(rest); m != nil {
					// Skip local/stdlib-looking paths
					if !strings.Contains(m[1], ".") || strings.HasPrefix(m[1], "github.com/") || strings.HasPrefix(m[1], "gopkg.in/") || strings.HasPrefix(m[1], "golang.org/") {
						addLib(m[1], m[2], scan.project, scan.brick)
					}
				}
				continue
			}

			if inRequire {
				if m := requireRe.FindStringSubmatch(trimmed); m != nil {
					if !strings.Contains(m[1], ".") || strings.HasPrefix(m[1], "github.com/") || strings.HasPrefix(m[1], "gopkg.in/") || strings.HasPrefix(m[1], "golang.org/") {
						addLib(m[1], m[2], scan.project, scan.brick)
					}
				}
			}
		}
	}

	return libs
}

// formatLibs formats the library table
func formatLibs(libs map[string]*libInfo, ws *workspace.Workspace) string {
	var sb strings.Builder

	// Get sorted project names
	projNames := ws.ProjectNames()
	// Also include brick names as columns
	brickNames := []string{}
	for _, n := range ws.ComponentNames() {
		brickNames = append(brickNames, n)
	}
	for _, n := range ws.BaseNames() {
		brickNames = append(brickNames, n)
	}

	// Sort libraries
	var keys []string
	for k := range libs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Header
	fmt.Fprintf(&sb, "%-40s %-12s", "library", "version")
	for _, p := range projNames {
		alias := p
		if proj := ws.Projects[p]; proj != nil && proj.Alias != "" {
			alias = proj.Alias
		}
		fmt.Fprintf(&sb, " %-5.5s", alias)
	}
	fmt.Fprintf(&sb, "\n")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("-", 54+len(projNames)*6))

	// Rows
	for _, k := range keys {
		lib := libs[k]
		fmt.Fprintf(&sb, "%-40s %-12s", lib.Name, lib.Version)
		for _, p := range projNames {
			marker := "  -  "
			if lib.UsedBy[p] {
				marker = "  x  "
			}
			fmt.Fprintf(&sb, " %s", marker)
		}
		fmt.Fprintf(&sb, "\n")
	}

	return sb.String()
}

// formatLibsCompact shows a more compact view
func formatLibsCompact(libs map[string]*libInfo, ws *workspace.Workspace) string {
	var sb strings.Builder

	var keys []string
	for k := range libs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		lib := libs[k]
		var users []string
		for p := range lib.UsedBy {
			users = append(users, p)
		}
		sort.Strings(users)
		fmt.Fprintf(&sb, "%s@%s (%s)\n", lib.Name, lib.Version, strings.Join(users, ", "))
	}

	return sb.String()
}
