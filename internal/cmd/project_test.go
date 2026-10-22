package cmd

import (
	"strings"
	"testing"
)

func TestProjectListPersonalAccount(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getMyProjects", []map[string]string{{"id": "1", "name": "Demo", "is_active": "1"}})
	out, err := runCLI(t, "--json", "project", "list")
	if err != nil || !strings.Contains(out, "Demo") {
		t.Fatalf("output = %s, error = %v", out, err)
	}
	if len(f.calls) != 1 || f.calls[0].Method != "getMyProjects" {
		t.Fatalf("wrong project procedure: %v", f.calls)
	}
}
