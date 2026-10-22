package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCredentialProcedures(t *testing.T) {
	for _, username := range []string{"alice", "jsonrpc"} {
		t.Run(username, func(t *testing.T) {
			var methods []string
			srv := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					user, token, ok := r.BasicAuth()
					if !ok || user != username || token != "secret" ||
						r.URL.Path != "/jsonrpc.php" {
						t.Error("incorrect endpoint or credentials")
					}
					var req rpcRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					methods = append(methods, req.Method)
					var result interface{}
					switch req.Method {
					case "getMe":
						result = map[string]string{"id": "7", "username": username}
					case "getVersion":
						result = "1.2.50"
					case "getAllProjects", "getMyProjects":
						result = []map[string]string{{"id": "1", "name": "Demo"}}
					default:
						t.Errorf("unexpected method %s", req.Method)
					}
					_ = json.NewEncoder(w).
						Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
				}),
			)
			defer srv.Close()
			client := NewClient(srv.URL, username, "secret")
			if err := client.VerifyAuthentication(); err != nil {
				t.Fatal(err)
			}
			projects, err := client.GetProjects()
			if err != nil || len(projects) != 1 || projects[0].Name != "Demo" {
				t.Fatalf("projects = %v, error = %v", projects, err)
			}
			want := "getMe,getMyProjects"
			if username == "jsonrpc" {
				want = "getVersion,getAllProjects"
			}
			if strings.Join(methods, ",") != want {
				t.Fatalf("methods = %v, want %s", methods, want)
			}
		})
	}
}

func TestAuthenticationErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		err := NewClient(srv.URL, "alice", "secret").VerifyAuthentication()
		srv.Close()
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe or missing error: %v", err)
		}
		if status == http.StatusForbidden && !strings.Contains(err.Error(), "getMe") {
			t.Fatalf("missing method context: %v", err)
		}
	}
}
