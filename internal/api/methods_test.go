package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecodeStringMap(t *testing.T) {
	tests := []struct {
		raw  string
		want map[string]string
	}{
		{`{"1":"bug","2":"ui"}`, map[string]string{"1": "bug", "2": "ui"}},
		{`[]`, map[string]string{}},
		{`null`, map[string]string{}},
		{`false`, map[string]string{}},
	}
	for _, tt := range tests {
		got, err := decodeStringMap(json.RawMessage(tt.raw), "task tags")
		if err != nil {
			t.Errorf("decodeStringMap(%s) error: %v", tt.raw, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("decodeStringMap(%s) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestCreateTaskParamsTags(t *testing.T) {
	data, err := json.Marshal(CreateTaskParams{Title: "x", ProjectID: 1, Tags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"title":"x","project_id":1}`; string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
	data, err = json.Marshal(CreateTaskParams{Title: "x", ProjectID: 1, Tags: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"title":"x","project_id":1,"tags":["a"]}`; string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestUpdateTaskParamsJSON(t *testing.T) {
	p := UpdateTaskParams{ID: 5}
	if !p.IsEmpty() {
		t.Error("IsEmpty() = false for params with only an ID")
	}
	zero, empty := 0, ""
	tags := []string{}
	p.OwnerID, p.DateDue, p.Tags = &zero, &empty, &tags
	if p.IsEmpty() {
		t.Error("IsEmpty() = true with fields set")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	// Explicit zero/empty values must be sent so fields can be cleared.
	if want := `{"id":5,"owner_id":0,"date_due":"","tags":[]}`; string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}
