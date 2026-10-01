package cmd

import (
	"reflect"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{" bug ", "", "Bug", "ui", "  ", "UI", "backend"})
	want := []string{"bug", "ui", "backend"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeTags = %q, want %q", got, want)
	}
	if got := normalizeTags(nil); got == nil || len(got) != 0 {
		t.Errorf("normalizeTags(nil) = %#v, want empty non-nil slice", got)
	}
}

func TestRemoveTags(t *testing.T) {
	remaining, missing := removeTags(
		[]string{"bug", "UI", "backend"},
		[]string{"ui", "frontend", "BUG"},
	)
	if want := []string{"backend"}; !reflect.DeepEqual(remaining, want) {
		t.Errorf("remaining = %q, want %q", remaining, want)
	}
	if want := []string{"frontend"}; !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %q, want %q", missing, want)
	}
}

func TestSameTags(t *testing.T) {
	tests := []struct {
		a, b []string
		want bool
	}{
		{nil, []string{}, true},
		{[]string{"a", "b"}, []string{"b", "a"}, true},
		{[]string{"a"}, []string{"A"}, false},
		{[]string{"a"}, []string{"a", "b"}, false},
	}
	for _, tt := range tests {
		if got := sameTags(tt.a, tt.b); got != tt.want {
			t.Errorf("sameTags(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSortedTagNames(t *testing.T) {
	got := sortedTagNames(map[string]string{"3": "beta", "1": "Alpha", "2": "gamma"})
	want := []string{"Alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedTagNames = %q, want %q", got, want)
	}
}
