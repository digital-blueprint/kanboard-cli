package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// rpcCall is a JSON-RPC call received by the fake server.
type rpcCall struct {
	Method string
	Params map[string]interface{}
}

// fakeKanboard is a minimal Kanboard JSON-RPC server for command tests.
type fakeKanboard struct {
	t        *testing.T
	mu       sync.Mutex
	calls    []rpcCall
	handlers map[string]func(params map[string]interface{}) interface{}
}

func newFakeKanboard(t *testing.T) *fakeKanboard {
	t.Helper()
	f := &fakeKanboard{t: t, handlers: map[string]func(map[string]interface{}) interface{}{}}

	// Defaults describing project 1 with task 42.
	f.on("getMe", map[string]interface{}{"id": "7", "username": "alice", "name": "Alice"})
	f.on("getTask", map[string]interface{}{
		"id": "42", "title": "Old title", "description": "Old description",
		"project_id": "1", "column_id": "10", "swimlane_id": "1", "owner_id": "0",
		"category_id": "0", "is_active": "1", "position": "3", "color_id": "yellow",
		"priority": "0", "score": "0", "time_estimated": "0", "time_spent": "0",
		"date_due": "0", "date_started": "0", "reference": "",
		"recurrence_status": "0", "recurrence_trigger": "0", "recurrence_factor": "0",
		"recurrence_timeframe": "0", "recurrence_basedate": "0",
	})
	f.on("getTaskTags", map[string]string{"5": "bug"})
	f.on("getProjectById", map[string]interface{}{
		"id": "1", "name": "Demo", "is_active": "1", "is_public": "0", "description": "",
		"identifier": "", "owner_id": "7", "email": "", "start_date": "", "end_date": "",
		"priority_default": "0", "priority_start": "0", "priority_end": "3",
	})
	f.on("getProjectByName", false)
	f.on("getAllCategories", []map[string]string{
		{"id": "3", "name": "Bug", "project_id": "1"},
		{"id": "4", "name": "Feature", "project_id": "1"},
	})
	f.on("getAssignableUsers", map[string]string{"7": "Alice", "8": "Bob Builder"})
	f.on("getProjectUsers", map[string]string{"7": "Alice", "8": "Bob Builder"})
	f.on("getUserByName", false)
	f.on("getColumns", []map[string]string{
		{"id": "10", "title": "Backlog", "position": "1"},
		{"id": "11", "title": "Done", "position": "2"},
	})
	f.on("getActiveSwimlanes", []map[string]string{{"id": "1", "name": "Default swimlane"}})
	f.on(
		"getComment",
		map[string]interface{}{"id": "17", "task_id": "42", "user_id": "7", "comment": "Old"},
	)
	for _, m := range []string{
		"updateTask", "moveTaskPosition", "moveTaskToProject", "openTask", "closeTask",
		"updateComment", "updateProject", "enableProject", "disableProject",
		"enableProjectPublicAccess", "disableProjectPublicAccess",
	} {
		f.on(m, true)
	}
	f.on("createTask", 99)

	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	t.Setenv("KANBOARD_URL", srv.URL)
	t.Setenv("KANBOARD_USERNAME", "alice")
	t.Setenv("KANBOARD_TOKEN", "secret")
	return f
}

// on sets a static result for a method.
func (f *fakeKanboard) on(method string, result interface{}) {
	f.handlers[method] = func(map[string]interface{}) interface{} { return result }
}

func (f *fakeKanboard) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Method string          `json:"method"`
		ID     int             `json:"id"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		f.t.Errorf("bad request: %v", err)
		return
	}
	params := map[string]interface{}{}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	f.mu.Lock()
	f.calls = append(f.calls, rpcCall{Method: req.Method, Params: params})
	h, ok := f.handlers[req.Method]
	f.mu.Unlock()

	resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID}
	if ok {
		resp["result"] = h(params)
	} else {
		resp["error"] = map[string]interface{}{"code": -32601, "message": "Method not found: " + req.Method}
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writes returns the calls that modify data, in order.
func (f *fakeKanboard) writes() []rpcCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []rpcCall
	for _, c := range f.calls {
		if !strings.HasPrefix(c.Method, "get") {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeKanboard) methods(calls []rpcCall) []string {
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.Method
	}
	return out
}

// runCLI executes the root command with args and returns stdout.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	jsonOutput = false
	t.Cleanup(func() { jsonOutput = false })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	root := NewRootCmd()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	runErr := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	return <-done, runErr
}
