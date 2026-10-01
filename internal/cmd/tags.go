package cmd

import (
	"sort"
	"strings"
)

// normalizeTags trims tag names, drops empty ones and removes
// case-insensitive duplicates while preserving the original order.
func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, tag)
	}
	return result
}

// removeTags removes the given tags (case-insensitively) from current. It
// returns the remaining tags and the requested tags that were not present.
func removeTags(current, remove []string) (remaining, missing []string) {
	toRemove := make(map[string]bool, len(remove))
	for _, tag := range normalizeTags(remove) {
		toRemove[strings.ToLower(tag)] = true
	}
	remaining = make([]string, 0, len(current))
	found := make(map[string]bool, len(remove))
	for _, tag := range current {
		key := strings.ToLower(tag)
		if toRemove[key] {
			found[key] = true
			continue
		}
		remaining = append(remaining, tag)
	}
	for _, tag := range normalizeTags(remove) {
		if !found[strings.ToLower(tag)] {
			missing = append(missing, tag)
		}
	}
	return remaining, missing
}

// sameTags reports whether a and b contain the same tag names, ignoring order
// but respecting case (so changing the case of a tag counts as a change).
func sameTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// sortedTagNames returns the tag names of a getTaskTags result, sorted
// case-insensitively.
func sortedTagNames(tags map[string]string) []string {
	names := make([]string, 0, len(tags))
	for _, name := range tags {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	return names
}
