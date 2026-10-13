package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VersionCmd shows the goliath version
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the goliath version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("goliath %s (commit: %s, built: %s) - https://github.com/igbanam/goliath\n", Version, Commit, Date)
	},
}
