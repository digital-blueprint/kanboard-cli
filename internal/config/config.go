package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// SaveCredentials stores the username in the config file and the token
// securely in the OS keyring.
func SaveCredentials(username, token string) error {
	if err := saveFileConfig(&fileConfig{Username: username}); err != nil {
		return err
	}
	if err := keyring.Set(keyringService, username, token); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return nil
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
	// Best-effort keyring deletion; ignore "not found" errors.
	_ = keyring.Delete(keyringService, username)
	return saveFileConfig(&fileConfig{})
}

// URL returns the Kanboard server URL from the environment variable.
func URL() (string, error) {
	u := os.Getenv(EnvURL)
	if u == "" {
		return "", fmt.Errorf("environment variable %s is not set", EnvURL)
	}
	return u, nil
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
				username, EnvToken,
			)
		}
		return "", "", fmt.Errorf("keyring lookup failed: %w", err)
	}
	return username, token, nil
}
