package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexibleTime unmarshals a JSON value that Kanboard returns as either:
//   - a bare integer   (Unix timestamp, e.g.  0  or  1720000000)
//   - a quoted string  (formatted date "2024-07-01 14:25", or "0", or "")
//
// It exposes the raw string representation via String() and a parsed
// time.Time (zero value when unset) via Time().
type FlexibleTime struct {
	raw string
}

func (f *FlexibleTime) UnmarshalJSON(b []byte) error {
	// Try quoted string first.
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		f.raw = s
		return nil
	}
	// Fall back to bare number.
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		f.raw = n.String()
		return nil
	}
	// null → zero value
	if string(b) == "null" {
		f.raw = ""
		return nil
	}
	return fmt.Errorf("FlexibleTime: cannot unmarshal %s", string(b))
}

func (f FlexibleTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.raw)
}

// String returns the raw value as received from the API.
func (f FlexibleTime) String() string { return f.raw }

// IsZero returns true when the timestamp is absent / zero.
func (f FlexibleTime) IsZero() bool {
	return f.raw == "" || f.raw == "0"
}

// Time parses the value into a time.Time.
// Returns zero time when unset.
func (f FlexibleTime) Time() time.Time {
	if f.IsZero() {
		return time.Time{}
	}
	// Unix timestamp (integer string).
	if n, err := strconv.ParseInt(f.raw, 10, 64); err == nil {
		return time.Unix(n, 0)
	}
	// Formatted date string.
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, f.raw, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

// Format returns a human-readable date string, or "-" when unset.
func (f FlexibleTime) Format(layout string) string {
	t := f.Time()
	if t.IsZero() {
		return "-"
	}
	return t.Format(layout)
}
