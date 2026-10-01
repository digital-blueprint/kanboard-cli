package cmd

import "testing"

func TestAppendDescription(t *testing.T) {
	tests := []struct {
		current, addition, want string
	}{
		{"", "new", "new"},
		{"old", "", "old"},
		{"old", "new", "old\nnew"},
		{"old\n", "new", "old\nnew"},
	}
	for _, tt := range tests {
		if got := appendDescription(tt.current, tt.addition); got != tt.want {
			t.Errorf(
				"appendDescription(%q, %q) = %q, want %q",
				tt.current,
				tt.addition,
				got,
				tt.want,
			)
		}
	}
}
