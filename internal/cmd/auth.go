package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

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
				return fmt.Errorf("kanboard URL cannot be empty")
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
				fd := int(os.Stdin.Fd())
				if term.IsTerminal(fd) {
					raw, err := readMasked(fd)
					fmt.Println()
					if err != nil {
						return err
					}
					token = strings.TrimSpace(raw)
				} else {
					// Fallback for non-terminal environments (pipes, tests).
					line, err := reader.ReadString('\n')
					if err != nil {
						return err
					}
					token = strings.TrimSpace(line)
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

// readMasked reads a line from the terminal in raw mode, echoing '*' for each
// character so the user gets visual feedback without revealing the secret.
func readMasked(fd int) (string, error) {
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var input []rune
	buf := make([]byte, 256)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		chunk := buf[:n]
		for len(chunk) > 0 {
			r, size := utf8.DecodeRune(chunk)
			chunk = chunk[size:]
			switch {
			case r == '\r' || r == '\n':
				return string(input), nil
			case r == 3: // Ctrl+C
				return "", errors.New("interrupted")
			case r == 4: // Ctrl+D
				if len(input) == 0 {
					return "", io.EOF
				}
			case r == 127 || r == 8: // Backspace
				if len(input) > 0 {
					input = input[:len(input)-1]
					fmt.Print("\b \b")
				}
			case r == 21: // Ctrl+U: clear line
				fmt.Print(strings.Repeat("\b \b", len(input)))
				input = input[:0]
			case r == 27: // Escape sequence (arrow keys etc.): ignore rest
				chunk = nil
			case r < 32 || r == utf8.RuneError:
				// Ignore other control characters.
			default:
				input = append(input, r)
				fmt.Print("*")
			}
		}
	}
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return strings.Repeat("*", len(t))
	}
	return t[:4] + strings.Repeat("*", len(t)-8) + t[len(t)-4:]
}
