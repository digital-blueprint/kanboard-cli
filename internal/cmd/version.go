package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				printJSON(map[string]string{
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				})
				return
			}
			fmt.Printf("kanboard-cli %s (commit %s, built %s)\n",
				version.Version, version.Commit, version.Date)
		},
	}
}
