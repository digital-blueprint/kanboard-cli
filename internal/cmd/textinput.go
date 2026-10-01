package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// stdinIsTerminal reports whether stdin is an interactive terminal.
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// readTextFile reads text from a file, or from stdin when path is "-".
func readTextFile(path string) (string, error) {
	if path == "-" {
		return readAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func readAll(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(data), nil
}

// appendDescription appends addition to current, separated by a newline.
func appendDescription(current, addition string) string {
	if current == "" {
		return addition
	}
	if addition == "" {
		return current
	}
	if !strings.HasSuffix(current, "\n") {
		current += "\n"
	}
	return current + addition
}

// editInEditor opens $VISUAL / $EDITOR (fallback: vi) on a temporary file
// containing initial and returns the edited content.
func editInEditor(initial, pattern string) (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	if !stdinIsTerminal() {
		return "", fmt.Errorf("cannot open editor: stdin is not a terminal")
	}

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()

	if _, err := f.WriteString(initial); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	// Run through the shell so EDITOR values with arguments (e.g. "code --wait")
	// work as expected.
	c := exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %w", editor, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read temp file: %w", err)
	}
	edited := string(data)
	// Editors usually add a trailing newline; don't treat that as a change.
	if !strings.HasSuffix(initial, "\n") {
		edited = strings.TrimSuffix(edited, "\n")
	}
	return edited, nil
}
