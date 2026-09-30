package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/tu-graz/kanboard-cli/internal/api"
)

func TestParseChecklist(t *testing.T) {
	desc := "Intro text\n" +
		"\n" +
		"## Todo\n" +
		"- [ ] First item\n" +
		"* [x] Second item  \n" +
		"+ [X] Third item\n" +
		"  - [ ] Nested item\n" +
		"1. [ ] Numbered item\n" +
		"2) [ ]   Padded   item\n" +
		"> - [ ] Quoted item\n" +
		"- [ ]\n" +
		"- [ ]    \n" +
		"- [] not a checkbox\n" +
		"-[ ] no space\n" +
		"- normal bullet\n" +
		"```\n" +
		"- [ ] inside code\n" +
		"```\n" +
		"~~~~md\n" +
		"- [ ] inside tilde code\n" +
		"~~~\n" +
		"- [ ] still inside (fence not closed by shorter run)\n" +
		"~~~~\n" +
		"- [ ] After code\r\n"

	got := parseChecklist(desc)
	want := []ChecklistItem{
		{Title: "First item", Checked: false, Line: 3},
		{Title: "Second item", Checked: true, Line: 4},
		{Title: "Third item", Checked: true, Line: 5},
		{Title: "Nested item", Checked: false, Line: 6},
		{Title: "Numbered item", Checked: false, Line: 7},
		{Title: "Padded   item", Checked: false, Line: 8},
		{Title: "Quoted item", Checked: false, Line: 9},
		{Title: "After code", Checked: false, Line: 23},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseChecklist mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseChecklistEmpty(t *testing.T) {
	if got := parseChecklist(""); len(got) != 0 {
		t.Fatalf("expected no items, got %#v", got)
	}
	if got := parseChecklist("no checkboxes here\n- plain"); len(got) != 0 {
		t.Fatalf("expected no items, got %#v", got)
	}
}

func TestRemoveLines(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		lines []int
		want  string
	}{
		{
			name:  "removes middle lines",
			text:  "Intro\n- [ ] a\n- [x] b\nOutro\n",
			lines: []int{1, 2},
			want:  "Intro\nOutro\n",
		},
		{
			name:  "removes trailing lines and keeps final newline",
			text:  "Intro\n\n- [ ] a\n- [ ] b\n",
			lines: []int{2, 3},
			want:  "Intro\n",
		},
		{
			name:  "no trailing newline",
			text:  "Intro\n- [ ] a",
			lines: []int{1},
			want:  "Intro",
		},
		{
			name:  "everything removed",
			text:  "- [ ] a\n- [ ] b\n",
			lines: []int{0, 1},
			want:  "",
		},
		{
			name:  "nothing removed",
			text:  "Intro\n- [ ] a\n",
			lines: nil,
			want:  "Intro\n- [ ] a\n",
		},
		{
			name:  "crlf preserved",
			text:  "Intro\r\n- [ ] a\r\nOutro\r\n",
			lines: []int{1},
			want:  "Intro\r\nOutro\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeLines(tt.text, tt.lines); got != tt.want {
				t.Fatalf("removeLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoveLinesMatchesParsedLines(t *testing.T) {
	desc := "Intro\n- [ ] a\n```\n- [ ] code\n```\n- [x] b\nOutro\n"
	var lines []int
	for _, item := range parseChecklist(desc) {
		lines = append(lines, item.Line)
	}
	want := "Intro\n```\n- [ ] code\n```\nOutro\n"
	if got := removeLines(desc, lines); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const testBase = "https://plan.example.org"

const legacyLink = "([subtask #5](https://plan.example.org/?controller=SubtaskController&action=edit&task_id=42&subtask_id=5))"

func TestSubtaskLinkRoundTrip(t *testing.T) {
	link := subtaskLink(testBase+"/", 42, 101)
	want := "([subtask #101](https://plan.example.org/task/42))"
	if link != want {
		t.Fatalf("subtaskLink() = %q, want %q", link, want)
	}
	title, id, got := extractSubtaskLink("Write tests " + link)
	if title != "Write tests" || id != 101 || got != link {
		t.Fatalf("extractSubtaskLink() = %q, %d, %q", title, id, got)
	}
}

func TestExtractSubtaskLink(t *testing.T) {
	tests := []struct {
		in    string
		title string
		id    int
	}{
		{"Plain item", "Plain item", 0},
		{"Item [docs](https://example.org/docs)", "Item [docs](https://example.org/docs)", 0},
		{"Item [docs](https://example.org/task/42)", "Item [docs](https://example.org/task/42)", 0},
		// Current format, parentheses dropped and case changed by hand.
		{"Item [Subtask #12](https://x/task/1)", "Item", 12},
		// Legacy edit-form links, whatever their text.
		{"Item " + legacyLink, "Item", 5},
		{
			"Item [see here](https://x/?controller=SubtaskController&action=edit&task_id=1&subtask_id=7)",
			"Item",
			7,
		},
		{"Item ([#8](https://x/?subtask_id=8&task_id=1))", "Item", 8},
		// Only an exact subtask_id parameter counts.
		{"Item ([x](https://x/?my_subtask_id=9))", "Item ([x](https://x/?my_subtask_id=9))", 0},
	}
	for _, tt := range tests {
		title, id, _ := extractSubtaskLink(tt.in)
		if title != tt.title || id != tt.id {
			t.Errorf(
				"extractSubtaskLink(%q) = %q, %d; want %q, %d",
				tt.in,
				title,
				id,
				tt.title,
				tt.id,
			)
		}
	}
}

func TestParseChecklistLinkedItems(t *testing.T) {
	link := subtaskLink(testBase, 42, 5)
	desc := "- [ ] Old item " + link + "\n- [x] New item\n"
	got := parseChecklist(desc)
	want := []ChecklistItem{
		{Title: "Old item", Line: 0, LinkedSubtaskID: 5, Link: link},
		{Title: "New item", Checked: true, Line: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestSetLinksOnLines(t *testing.T) {
	text := "Intro\r\n- [ ] a  \r\n- [ ] b\n- [ ] c " + legacyLink + " trailing"
	got := setLinksOnLines(text, map[int]string{1: "(A)", 3: "(C)"})
	want := "Intro\r\n- [ ] a (A)\r\n- [ ] b\n- [ ] c trailing (C)"
	if got != want {
		t.Fatalf("setLinksOnLines() = %q, want %q", got, want)
	}
}

func subtask(id, title string) api.Subtask {
	return api.Subtask{ID: json.Number(id), Title: title}
}

func TestPlanChecklistSync(t *testing.T) {
	desc := "- [ ] Linked " + subtaskLink(testBase, 42, 5) + "\n" + // 0: linked
		"- [ ] Deleted " + subtaskLink(testBase, 42, 99) + "\n" + // 1: missing
		"- [ ] existing ONE\n" + // 2: matches subtask 6 by title
		"- [ ] Existing one\n" + // 3: subtask 6 already claimed -> create
		"- [ ] linked\n" + // 4: subtask 5 claimed by line 0 -> create
		"- [x] Done item\n" + // 5: create (or skip with SkipChecked)
		"- [ ] Brand new\n" + // 6: create
		"- [ ] Legacy ([s](https://plan.example.org/?task_id=42&subtask_id=7))\n" // 7: outdated link
	items := parseChecklist(desc)
	subtasks := []api.Subtask{
		subtask("5", "Linked"),
		subtask("6", "Existing one"),
		subtask("7", "Legacy"),
	}

	actions := func(rs []checklistResult) []string {
		var out []string
		for _, r := range rs {
			out = append(out, fmt.Sprintf("%s:%d", r.Action, r.SubtaskID))
		}
		return out
	}

	got := actions(planChecklistSync(items, subtasks, checklistPlanOptions{}))
	want := []string{
		"linked:5", "missing:99", "would-link:6", "would-create:0",
		"would-create:0", "would-create:0", "would-create:0", "linked:7",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default plan\n got: %v\nwant: %v", got, want)
	}

	got = actions(
		planChecklistSync(
			items,
			subtasks,
			checklistPlanOptions{SkipChecked: true, AllowDuplicates: true},
		),
	)
	want = []string{
		"linked:5", "missing:99", "would-create:0", "would-create:0",
		"would-create:0", "skipped-checked:0", "would-create:0", "linked:7",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skip-checked/allow-duplicates plan\n got: %v\nwant: %v", got, want)
	}
}

func TestPlanChecklistSyncUpdatesLinks(t *testing.T) {
	desc := "- [ ] Current " + subtaskLink(testBase, 42, 5) + "\n" +
		"- [ ] Legacy ([subtask #6](https://plan.example.org/?controller=SubtaskController&action=edit&task_id=42&subtask_id=6))\n" +
		"- [ ] Other host " + subtaskLink("https://other.example.org", 42, 7) + "\n"
	subtasks := []api.Subtask{
		subtask("5", "Current"),
		subtask("6", "Legacy"),
		subtask("7", "Other host"),
	}
	rs := planChecklistSync(parseChecklist(desc), subtasks, checklistPlanOptions{
		UpdateLinks: true, BaseURL: testBase, TaskID: 42,
	})
	var got []string
	for _, r := range rs {
		got = append(got, fmt.Sprintf("%s:%d", r.Action, r.SubtaskID))
	}
	want := []string{"linked:5", "would-update-link:6", "would-update-link:7"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseSubtaskStatus(t *testing.T) {
	cases := map[string]int{
		"todo": 0, "0": 0, "TODO": 0,
		"in-progress": 1, "progress": 1, "1": 1,
		"done": 2, " Done ": 2, "2": 2,
	}
	for in, want := range cases {
		got, err := parseSubtaskStatus(in)
		if err != nil || got != want {
			t.Errorf("parseSubtaskStatus(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseSubtaskStatus("bogus"); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestNormalizeTitle(t *testing.T) {
	if normalizeTitle("  Write   Tests ") != normalizeTitle("write tests") {
		t.Fatal("expected normalized titles to match")
	}
}
