package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// InfoCmd shows workspace information
var InfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show workspace information",
	Long: `Show an overview of the workspace including projects, components, bases, and interfaces.

Example:
  goliath info`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		printInfo(ws)
		return nil
	},
}

func init() {
	InfoCmd.Flags().String("ws-dir", "", "Path to workspace directory")
	InfoCmd.Flags().String("out", "", "Write output to file")
}

func printInfo(ws *workspace.Workspace) {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Workspace: %s\n", ws.Name))
	sb.WriteString(fmt.Sprintf("Directory: %s\n", ws.Dir))
	sb.WriteString(fmt.Sprintf("Top namespace: %s\n\n", ws.Config.TopNamespace))

	// Summary counts
	numInterfaces := len(ws.InterfaceNames())
	sb.WriteString(fmt.Sprintf("projects: %d   interfaces: %d\n", len(ws.Projects), numInterfaces))
	sb.WriteString(fmt.Sprintf("bases:    %d   components: %d\n\n", len(ws.Bases), len(ws.Components)))

	// Projects table
	if len(ws.Projects) > 0 {
		sb.WriteString("project       alias  module\n")
		sb.WriteString("--------------------------\n")
		for _, name := range ws.ProjectNames() {
			proj := ws.Projects[name]
			alias := proj.Alias
			if alias == "" {
				alias = "-"
			}
			sb.WriteString(fmt.Sprintf("%-13s %-6s %s\n", proj.Name, alias, proj.Module))
		}
		sb.WriteString("\n")
	}

	// Bricks table
	if len(ws.Components) > 0 || len(ws.Bases) > 0 {
		sb.WriteString("interface  brick    type       module\n")
		sb.WriteString("-----------------   ---------  ----------------------\n")

		// Components sorted by interface then name
		type compRow struct {
			iface string
			name  string
			mod   string
		}
		var rows []compRow
		for _, name := range ws.ComponentNames() {
			comp := ws.Components[name]
			rows = append(rows, compRow{comp.Interface, comp.Name, comp.Module})
		}
		// Sort by interface, then name
		for i := 0; i < len(rows); i++ {
			for j := i + 1; j < len(rows); j++ {
				if rows[i].iface > rows[j].iface ||
					(rows[i].iface == rows[j].iface && rows[i].name > rows[j].name) {
					rows[i], rows[j] = rows[j], rows[i]
				}
			}
		}

		for _, row := range rows {
			sb.WriteString(fmt.Sprintf("%-10s %-8s component  %s\n", row.iface, row.name, row.mod))
		}

		for _, name := range ws.BaseNames() {
			base := ws.Bases[name]
			sb.WriteString(fmt.Sprintf("%-10s %-8s %-10s %s\n", "-", base.Name, "base", base.Module))
		}
	}

	fmt.Print(sb.String())
}
