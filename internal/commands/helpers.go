package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// loadWorkspace loads the workspace from the current directory or returns an error
func loadWorkspace() (*workspace.Workspace, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	wsDir, err := workspace.FindWorkspaceDir(dir)
	if err != nil {
		return nil, err
	}

	return workspace.Load(wsDir)
}

// loadWorkspaceFromArgs checks for ws-dir flag and loads from there if present
func loadWorkspaceFromArgs(cmd *cobra.Command) (*workspace.Workspace, error) {
	wsDir, _ := cmd.Flags().GetString("ws-dir")
	if wsDir != "" {
		return workspace.Load(wsDir)
	}
	return loadWorkspace()
}
