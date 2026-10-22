package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tu-graz/kanboard-cli/internal/config"
	"github.com/zalando/go-keyring"
)

func TestAuthStatus(t *testing.T) {
	f := newFakeKanboard(t)
	out, err := runCLI(t, "--json", "auth", "status")
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		Authenticated bool   `json:"authenticated"`
		Checked       bool   `json:"checked"`
		Source        string `json:"source"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Authenticated || !status.Checked || status.Source != "environment (KANBOARD_TOKEN)" {
		t.Fatalf("incorrect status: %s", out)
	}
	f.calls = nil
	out, err = runCLI(t, "--json", "auth", "status", "--offline")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatal(err)
	}
	if status.Authenticated || status.Checked || len(f.calls) != 0 {
		t.Fatalf("offline status checked server: %s", out)
	}
}

func TestAuthStatusRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv(config.EnvURL, srv.URL)
	t.Setenv(config.EnvUsername, "alice")
	t.Setenv(config.EnvToken, "secret")
	out, err := runCLI(t, "--json", "auth", "status")
	if err == nil || !strings.Contains(out, `"authenticated": false`) ||
		!strings.Contains(out, `"checked": true`) {
		t.Fatalf("status = %s, error = %v", out, err)
	}
}

func TestAuthLoginVerifiesBeforeSaving(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		t.Run(map[bool]string{false: "rejected", true: "accepted"}[accepted], func(t *testing.T) {
			newFakeKanboard(t)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv(config.EnvToken, "")
			t.Setenv(config.EnvUsername, "")
			keyring.MockInit()
			if err := config.SaveCredentials("https://old.example", "alice", "old-token"); err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if !accepted {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					var req struct {
						ID int `json:"id"`
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					_ = json.NewEncoder(w).
						Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": map[string]string{"id": "7"}})
				}),
			)
			defer srv.Close()
			input, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = input.Close() }()
			_, _ = input.WriteString("new-token\n")
			_, _ = input.Seek(0, 0)
			oldStdin := os.Stdin
			os.Stdin = input
			defer func() { os.Stdin = oldStdin }()
			_, err = runCLI(t, "auth", "login", "--url", srv.URL, "-u", "alice")
			if (err == nil) != accepted {
				t.Fatalf("accepted = %v, error = %v", accepted, err)
			}
			_, token, err := config.Credentials()
			want := "old-token"
			if accepted {
				want = "new-token"
			}
			if err != nil || token != want {
				t.Fatal("saved credentials do not match verification result")
			}
		})
	}
}
