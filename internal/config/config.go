package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	// EnvURL is the environment variable for the Kanboard server URL.
	EnvURL = "KANBOARD_URL"
	// EnvUsername is the environment variable for overriding the username.
	EnvUsername = "KANBOARD_USERNAME"
	// EnvToken is the environment variable for overriding the API token.
	// Useful in CI/CD pipelines where a keyring is not available.
	EnvToken = "KANBOARD_TOKEN"

	keyringService = "kanboard-cli"
)

// fileConfig holds non-secret configuration persisted to disk.
type fileConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
}

// configPath returns the path to the config file, creating the directory if needed.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	configDir := filepath.Join(dir, "kanboard-cli")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return filepath.Join(configDir, "config.json"), nil
}

func loadFileConfig() (*fileConfig, error) {
	path, err := configPath()
	if err != nil {
		return &fileConfig{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &fileConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func saveFileConfig(cfg *fileConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// SaveCredentials stores the server URL and username in the config file and the token
// securely in the OS keyring.
func SaveCredentials(serverURL, username, token string) error {
	if err := keyring.Set(keyringService, username, token); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return saveFileConfig(&fileConfig{URL: serverURL, Username: username})
}

// CredentialSource describes the effective token source without exposing it.
func CredentialSource() string {
	if os.Getenv(EnvToken) != "" {
		return "environment (KANBOARD_TOKEN)"
	}
	return "keyring"
}

// KeyringLocation describes how to find the stored entry with a keyring manager.
func KeyringLocation(username string) string {
	if runtime.GOOS == "linux" {
		return fmt.Sprintf(
			"Secret Service login collection; label: Password for '%s' on '%s'; attributes: service=%s, username=%s (use Seahorse/KeepSecret if KWallet Manager does not show it)",
			username,
			keyringService,
			keyringService,
			username,
		)
	}
	return fmt.Sprintf("OS keyring; service: %s; username: %s", keyringService, username)
}

// DeleteCredentials removes the stored credentials.
func DeleteCredentials() error {
	cfg, err := loadFileConfig()
	if err != nil {
		return err
	}
	username := cfg.Username
	if username == "" {
		username = "jsonrpc"
	}
	if err := keyring.Delete(keyringService, username); err != nil &&
		!errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete token from keyring: %w", err)
	}
	cfg.Username = ""
	return saveFileConfig(cfg)
}

// URL returns the Kanboard server URL from the environment variable or config file.
func URL() (string, error) {
	u := os.Getenv(EnvURL)
	if u != "" {
		return u, nil
	}
	cfg, err := loadFileConfig()
	if err != nil {
		return "", err
	}
	if cfg.URL == "" {
		return "", fmt.Errorf(
			"kanboard URL is not configured; run 'kanboard-cli auth login' or set %s",
			EnvURL,
		)
	}
	return cfg.URL, nil
}

// Credentials returns (username, token) by checking env vars first, then the
// config file (username) and OS keyring (token).
func Credentials() (string, string, error) {
	// 1. Environment variables take precedence (useful in CI/CD).
	if envToken := os.Getenv(EnvToken); envToken != "" {
		username := os.Getenv(EnvUsername)
		if username == "" {
			username = "jsonrpc"
		}
		return username, envToken, nil
	}

	// 2. Load username from config file.
	cfg, err := loadFileConfig()
	if err != nil {
		return "", "", err
	}
	username := os.Getenv(EnvUsername)
	if username == "" {
		username = cfg.Username
	}
	if username == "" {
		username = "jsonrpc"
	}

	// 3. Load token from the OS keyring.
	token, err := keyring.Get(keyringService, username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", "", fmt.Errorf(
				"no API token found in keyring for user %q; run 'kanboard-cli auth login' or set %s",
				username,
				EnvToken,
			)
		}
		return "", "", fmt.Errorf("keyring lookup failed: %w", err)
	}
	return username, token, nil
}
