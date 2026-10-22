package config

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyringFailurePreservesConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	keyring.MockInit()
	if err := SaveCredentials("https://old.example", "alice", "token"); err != nil {
		t.Fatal(err)
	}
	keyring.MockInitWithError(errors.New("wallet unavailable"))
	if err := SaveCredentials("https://new.example", "bob", "new-token"); err == nil {
		t.Fatal("expected keyring save failure")
	}
	if err := DeleteCredentials(); err == nil {
		t.Fatal("expected keyring deletion failure")
	}
	cfg, err := loadFileConfig()
	if err != nil || cfg.URL != "https://old.example" || cfg.Username != "alice" {
		t.Fatalf("config changed after failure: %v, %v", cfg, err)
	}
}
