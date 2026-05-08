package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/config"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication credentials",
	}
	cmd.AddCommand(newAuthLoginCmd(), newAuthStatusCmd(), newAuthLogoutCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var serverURL, username, token string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store API credentials in the OS keyring",
		Long: `Store your Kanboard API token securely in the OS keyring
(libsecret/GNOME Keyring on Linux, Keychain on macOS,
Credential Manager on Windows).

The server URL and username are stored in a plain config file; only the token is
kept in the keyring.

For the application API use username "jsonrpc" and the token from
Settings > API.  For the user API use your username and a personal
access token generated in your profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			if serverURL == "" {
				currentURL, _ := config.URL()
				if currentURL == "" {
					fmt.Print("Kanboard URL: ")
				} else {
					fmt.Printf("Kanboard URL [%s]: ", currentURL)
				}
				u, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				serverURL = strings.TrimSpace(u)
				if serverURL == "" {
					serverURL = currentURL
				}
			}

			if serverURL == "" {
				return fmt.Errorf("Kanboard URL cannot be empty")
			}

			if username == "" {
				fmt.Print("Username [jsonrpc]: ")
				u, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				username = strings.TrimSpace(u)
				if username == "" {
					username = "jsonrpc"
				}
			}

			if token == "" {
				fmt.Print("API token: ")
				raw, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					// Fallback for non-terminal environments (pipes, tests).
					line, err2 := reader.ReadString('\n')
					if err2 != nil {
						return err2
					}
					token = strings.TrimSpace(line)
				} else {
					token = strings.TrimSpace(string(raw))
				}
			}

			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			if err := config.SaveCredentials(serverURL, username, token); err != nil {
				return err
			}
			fmt.Println("Settings saved and credentials stored in OS keyring.")
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "url", "", "Kanboard server URL")
	cmd.Flags().StringVarP(&username, "username", "u", "", "Kanboard username (default: jsonrpc)")
	// Intentionally not providing a --token flag to discourage passing secrets
	// on the command line (visible in shell history / process list).
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			serverURL, _ := config.URL()
			username, token, err := config.Credentials()

			if jsonOutput {
				out := map[string]interface{}{
					"server":        serverURL,
					"username":      username,
					"token_masked":  maskToken(token),
					"authenticated": err == nil,
				}
				if err != nil {
					out["error"] = err.Error()
				}
				printJSON(out)
				return nil
			}

			if serverURL == "" {
				fmt.Println("Server:   not configured")
			} else {
				fmt.Println("Server:  ", serverURL)
			}
			if err != nil {
				fmt.Println("Credentials:", err)
				return nil
			}
			fmt.Println("Username:", username)
			fmt.Println("Token:   ", maskToken(token))
			fmt.Println("Source:   keyring")
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials from the OS keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.DeleteCredentials(); err != nil {
				return err
			}
			fmt.Println("Credentials removed from keyring.")
			return nil
		},
	}
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return strings.Repeat("*", len(t))
	}
	return t[:4] + strings.Repeat("*", len(t)-8) + t[len(t)-4:]
}
