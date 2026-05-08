package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
	"github.com/tu-graz/kanboard-cli/internal/config"
)

// jsonOutput is the process-wide flag set by --json on the root command.
var jsonOutput bool

// newClient builds an API client from config/env, printing error and exiting on failure.
func newClient() *api.Client {
	serverURL, err := config.URL()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	username, token, err := config.Credentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return api.NewClient(serverURL, username, token)
}

// die prints a formatted error and exits with code 1.
func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// printJSON marshals v to indented JSON and writes it to stdout.
// On error it writes to stderr and exits 1.
func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
		os.Exit(1)
	}
}

// NewRootCmd builds and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kanboard-cli",
		Short: "A CLI client for Kanboard",
		Long: `kanboard-cli lets you manage projects, tasks, and comments on a
Kanboard instance from the command line.

The server URL and authentication credentials are configured by running
'kanboard-cli auth login'. KANBOARD_URL can be used to override the stored
server URL.`,
	}

	// --json is a persistent flag: available on every sub-command.
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output result as JSON")

	root.AddCommand(
		newAuthCmd(),
		newProjectCmd(),
		newTaskCmd(),
		newCommentCmd(),
		newVersionCmd(),
	)

	return root
}
