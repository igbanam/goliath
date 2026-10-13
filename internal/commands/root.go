package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the goliath version.
// Overridden at build time via -ldflags "-X .../commands.Version=<tag>" (e.g. by GoReleaser).
var Version = "dev"

// Date is the build date.
// Overridden at build time via -ldflags "-X .../commands.Date=<date>".
var Date = "unknown"

// Commit is the git commit SHA the binary was built from.
// Overridden at build time via -ldflags "-X .../commands.Commit=<sha>".
var Commit = "unknown"

// RootCmd is the root command
var RootCmd = &cobra.Command{
	Use:   "goliath",
	Short: "Goliath — Polylith-style workspace management for Go",
	Long: `Goliath brings Polylith architecture to Go.

It manages workspaces containing:
  - Components: domain logic with public interfaces
  - Bases: entry points that wire components together
  - Projects: deployable artifacts

Get started:
  goliath create workspace --name myws --top-ns github.com/myorg`,
	SilenceUsage: true,
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	RootCmd.AddCommand(InfoCmd)
	RootCmd.AddCommand(CheckCmd)
	RootCmd.AddCommand(DepsCmd)
	RootCmd.AddCommand(DiffCmd)
	RootCmd.AddCommand(LibsCmd)
	RootCmd.AddCommand(TestCmd)
	RootCmd.AddCommand(WsCmd)
	RootCmd.AddCommand(VersionCmd)
}

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
