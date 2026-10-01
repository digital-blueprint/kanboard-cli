package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTaskUpdateFields(t *testing.T) {
	f := newFakeKanboard(t)
	out, err := runCLI(t, "task", "update", "42",
		"--title", "New title",
		"--priority", "2",
		"--category", "bug",
		"--assignee", "me",
		"--color", "yellow", // unchanged, must not be sent
		"--due", "2026-10-15",
		"--estimate", "90m",
		"--recurrence", "on", "--recurrence-every", "2d",
		"--tag", "urgent",
	)
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	writes := f.writes()
	if got := f.methods(writes); !reflect.DeepEqual(got, []string{"updateTask"}) {
		t.Fatalf("write calls = %v, want [updateTask]", got)
	}
	want := map[string]interface{}{
		"id":                   float64(42),
		"title":                "New title",
		"priority":             float64(2),
		"category_id":          float64(3),
		"owner_id":             float64(7),
		"date_due":             "2026-10-15 00:00",
		"time_estimated":       1.5,
		"recurrence_status":    float64(1),
		"recurrence_factor":    float64(2),
		"recurrence_timeframe": float64(0),
		"tags":                 []interface{}{"bug", "urgent"},
	}
	if !reflect.DeepEqual(writes[0].Params, want) {
		t.Errorf("updateTask params =\n  %v\nwant\n  %v", writes[0].Params, want)
	}
	for _, s := range []string{"Task 42 updated:", "category:", "Bug (#3)", "assignee:", "Alice (#7)"} {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q:\n%s", s, out)
		}
	}
	if strings.Contains(out, "color") {
		t.Errorf("unchanged color reported as change:\n%s", out)
	}
}

func TestTaskUpdateDryRunWritesNothing(t *testing.T) {
	f := newFakeKanboard(t)
	out, err := runCLI(t, "task", "update", "42", "--title", "X", "--column", "Done",
		"--status", "closed", "--dry-run")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if w := f.writes(); len(w) != 0 {
		t.Errorf("dry run made write calls: %v", f.methods(w))
	}
	if !strings.Contains(out, "would change (dry run)") || !strings.Contains(out, "Done (#11)") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestTaskUpdatePlacementAndStatusOrder(t *testing.T) {
	f := newFakeKanboard(t)
	_, err := runCLI(t, "task", "update", "42", "--status", "closed", "--column", "done",
		"--untag", "bug")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	writes := f.writes()
	if got := f.methods(writes); !reflect.DeepEqual(got,
		[]string{"moveTaskPosition", "updateTask", "closeTask"}) {
		t.Fatalf("write calls = %v", got)
	}
	move := writes[0].Params
	if move["column_id"] != float64(11) || move["position"] != float64(1) ||
		move["swimlane_id"] != float64(1) || move["project_id"] != float64(1) {
		t.Errorf("moveTaskPosition params = %v", move)
	}
	// Removing the last tag must send an empty list, not omit the field.
	tags, ok := writes[1].Params["tags"].([]interface{})
	if !ok || len(tags) != 0 {
		t.Errorf("updateTask tags = %#v, want []", writes[1].Params["tags"])
	}
}

func TestTaskUpdateClearValues(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTask", map[string]interface{}{
		"id": "42", "project_id": "1", "owner_id": "7", "category_id": "3",
		"date_due": "1790000000", "reference": "T-1", "is_active": "1",
	})
	_, err := runCLI(t, "task", "update", "42", "--assignee", "none", "--category", "none",
		"--due", "none", "--reference", "")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	want := map[string]interface{}{
		"id": float64(42), "owner_id": float64(0), "category_id": float64(0),
		"date_due": "", "reference": "",
	}
	if w := f.writes(); len(w) != 1 || !reflect.DeepEqual(w[0].Params, want) {
		t.Errorf("writes = %+v, want updateTask %v", w, want)
	}
}

func TestTaskUpdateUnchanged(t *testing.T) {
	f := newFakeKanboard(t)
	out, err := runCLI(t, "task", "update", "42", "--title", "Old title", "--tag", "BUG")
	if err != nil {
		t.Fatal(err)
	}
	if w := f.writes(); len(w) != 0 {
		t.Errorf("unexpected writes: %v", f.methods(w))
	}
	if !strings.Contains(out, "Task 42 unchanged") {
		t.Errorf("output = %q", out)
	}
}

func TestTaskUpdateValidationErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"42"}, "nothing to update"},
		{[]string{"42", "--priority", "9"}, "outside the range 0..3"},
		{[]string{"42", "--category", "Nope"}, `category "Nope" not found`},
		{[]string{"42", "--color", "ultraviolet"}, "invalid --color"},
		{[]string{"42", "--due", "15.10.2026"}, "invalid date"},
		{[]string{"42", "43", "-e"}, "single task"},
		{[]string{"42", "-d", "x", "--append-description", "y"}, "none of the others can be"},
		{[]string{"42", "--assignee", "Zed"}, `user "Zed" is not assignable`},
	}
	for _, tt := range tests {
		f := newFakeKanboard(t)
		_, err := runCLI(t, append([]string{"task", "update"}, tt.args...)...)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("args %v: err = %v, want containing %q", tt.args, err, tt.want)
		}
		if w := f.writes(); len(w) != 0 {
			t.Errorf("args %v: unexpected writes %v", tt.args, f.methods(w))
		}
	}
}

func TestTaskUpdateMultipleTasksJSON(t *testing.T) {
	f := newFakeKanboard(t)
	out, err := runCLI(t, "--json", "task", "update", "42", "43", "--assignee", "bob")
	if err != nil {
		t.Fatal(err)
	}
	var results []taskUpdateResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if len(results) != 2 || !results[0].Updated || results[0].Changes[0].To != "Bob Builder (#8)" {
		t.Errorf("results = %+v", results)
	}
	if got := f.methods(f.writes()); len(got) != 2 {
		t.Errorf("writes = %v, want 2 updateTask calls", got)
	}
}

func TestTaskCreateWithFields(t *testing.T) {
	f := newFakeKanboard(t)
	_, err := runCLI(t, "task", "create", "Fix it", "--project-id", "1", "-c", "Done",
		"--category", "Feature", "--assignee", "8", "--priority", "0", "-t", "a,b", "--tag", "A")
	if err != nil {
		t.Fatal(err)
	}
	w := f.writes()
	if len(w) != 1 || w[0].Method != "createTask" {
		t.Fatalf("writes = %v", f.methods(w))
	}
	want := map[string]interface{}{
		"title": "Fix it", "project_id": float64(1), "column_id": float64(11),
		"category_id": float64(4), "owner_id": float64(8), "priority": float64(0),
		"tags": []interface{}{"a", "b"},
	}
	if !reflect.DeepEqual(w[0].Params, want) {
		t.Errorf("createTask params = %v, want %v", w[0].Params, want)
	}
}

func TestCommentEdit(t *testing.T) {
	f := newFakeKanboard(t)
	if _, err := runCLI(t, "comment", "edit", "17", "--append", "More"); err != nil {
		t.Fatal(err)
	}
	w := f.writes()
	want := map[string]interface{}{"id": float64(17), "content": "Old\nMore"}
	if len(w) != 1 || w[0].Method != "updateComment" || !reflect.DeepEqual(w[0].Params, want) {
		t.Errorf("writes = %+v", w)
	}

	f = newFakeKanboard(t)
	f.on("updateComment", false)
	_, err := runCLI(t, "comment", "edit", "17", "New")
	if err == nil || !strings.Contains(err.Error(), "your own comments") {
		t.Errorf("err = %v", err)
	}
}

func TestProjectUpdate(t *testing.T) {
	f := newFakeKanboard(t)
	_, err := runCLI(t, "project", "update", "1", "--name", "Demo", "--identifier", "sws",
		"--end-date", "2026-12-31", "--priority-end", "5", "--status", "inactive", "--public", "no")
	if err != nil {
		t.Fatal(err)
	}
	w := f.writes()
	if got := f.methods(w); !reflect.DeepEqual(got, []string{"updateProject", "disableProject"}) {
		t.Fatalf("writes = %v", got)
	}
	want := map[string]interface{}{
		"project_id": float64(1), "identifier": "SWS", "end_date": "2026-12-31",
		"priority_end": float64(5),
	}
	if !reflect.DeepEqual(w[0].Params, want) {
		t.Errorf("updateProject params = %v, want %v", w[0].Params, want)
	}

	newFakeKanboard(t)
	_, err = runCLI(t, "project", "update", "1", "--priority-default", "7")
	if err == nil || !strings.Contains(err.Error(), "outside the range") {
		t.Errorf("err = %v", err)
	}
}
