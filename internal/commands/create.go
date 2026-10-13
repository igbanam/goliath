package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// CreateCmd is the parent command for create subcommands
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a workspace, component, base, or project",
	Long: `Create a new workspace, component, base, or project.

Examples:
  goliath create workspace --name myws --top-ns github.com/myorg
  goliath create component user
  goliath create component admin --interface user
  goliath create base cli
  goliath create project myapp`,
}

var createWorkspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Create a new workspace",
	Long: `Create a new goliath workspace in the current directory (or a named subdirectory).

Examples:
  goliath create workspace --top-ns github.com/myorg
  goliath create workspace --name myws --top-ns github.com/myorg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		topNS, _ := cmd.Flags().GetString("top-ns")

		dir := "."
		if name != "" {
			dir = name
		}

		if err := workspace.CreateWorkspace(dir, name, topNS); err != nil {
			return err
		}

		if name != "" {
			fmt.Printf("Workspace '%s' created successfully!\n", name)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %s\n", name)
			fmt.Printf("  goliath create component mycomponent\n")
		} else {
			fmt.Printf("Workspace created successfully in current directory!\n")
		}
		return nil
	},
}

var createComponentCmd = &cobra.Command{
	Use:   "component [name]",
	Short: "Create a new component",
	Long: `Create a new component brick with an interface and implementation.

Examples:
  goliath create component user
  goliath create component admin --interface user`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		iface, _ := cmd.Flags().GetString("interface")

		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		if err := workspace.CreateComponent(ws, name, iface); err != nil {
			return err
		}

		fmt.Printf("Component '%s' created successfully!\n", name)
		if iface != "" && iface != name {
			fmt.Printf("  Interface: %s\n", iface)
		}
		fmt.Printf("  Location: components/%s\n", name)
		return nil
	},
}

var createBaseCmd = &cobra.Command{
	Use:   "base [name]",
	Short: "Create a new base",
	Long: `Create a new base brick (an entry point that wires components together).

Example:
  goliath create base cli`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		if err := workspace.CreateBase(ws, name); err != nil {
			return err
		}

		fmt.Printf("Base '%s' created successfully!\n", name)
		fmt.Printf("  Location: bases/%s\n", name)
		return nil
	},
}

var createProjectCmd = &cobra.Command{
	Use:   "project [name]",
	Short: "Create a new project",
	Long: `Create a new deployable project that combines bases and components.

Example:
  goliath create project myapp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		if err := workspace.CreateProject(ws, name); err != nil {
			return err
		}

		fmt.Printf("Project '%s' created successfully!\n", name)
		fmt.Printf("  Location: projects/%s\n", name)
		return nil
	},
}

func init() {
	createWorkspaceCmd.Flags().String("name", "", "Workspace name (creates subdirectory if given)")
	createWorkspaceCmd.Flags().String("top-ns", "", "Top namespace / module path prefix (required)")
	createWorkspaceCmd.MarkFlagRequired("top-ns")

	createComponentCmd.Flags().String("interface", "", "Interface name (defaults to component name)")

	CreateCmd.AddCommand(createWorkspaceCmd)
	CreateCmd.AddCommand(createComponentCmd)
	CreateCmd.AddCommand(createBaseCmd)
	CreateCmd.AddCommand(createProjectCmd)
}
