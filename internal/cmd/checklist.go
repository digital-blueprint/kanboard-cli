package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ChecklistItem is a single Markdown task-list entry ("- [ ] ...") found in a
// task description.
type ChecklistItem struct {
	Title   string `json:"title"`
	Checked bool   `json:"checked"`
	Line    int    `json:"line"` // zero-based line index in the description
	// LinkedSubtaskID is the subtask ID referenced by a link already present
	// on the line (see subtaskLink), or 0 if the item is not linked yet.
	LinkedSubtaskID int `json:"linked_subtask_id,omitempty"`
	// Link is the subtask link found on the line, as written in the
	// description. Used to detect links in an outdated format.
	Link string `json:"-"`
}

// subtaskLinkRe matches a subtask link appended to a checklist item, with or
// without the surrounding parentheses. Two forms are recognised:
//
//   - the current format written by subtaskLink: a link with the text
//     "subtask #<id>" pointing to the parent task page;
//   - the earlier format: any link whose URL carries a subtask_id query
//     parameter (the subtask edit form), whatever its text.
var subtaskLinkRe = regexp.MustCompile(
	`[ \t]*\(?(?:` +
		`\[(?i:subtask) #(\d+)\]\([^)\s]*\)` +
		`|` +
		`\[[^\]\n]*\]\([^)\s]*[?&]subtask_id=(\d+)[^)\s]*\)` +
		`)\)?`,
)

// taskURL returns the URL of a task's page, where its subtasks are listed.
func taskURL(baseURL string, taskID int) string {
	return fmt.Sprintf("%s/task/%d", strings.TrimRight(baseURL, "/"), taskID)
}

// subtaskLink returns the Markdown snippet appended to a checklist item that
// was converted into a subtask. Kanboard subtasks have no page of their own,
// so the link opens the parent task; the link text carries the subtask ID.
func subtaskLink(baseURL string, taskID, subtaskID int) string {
	return fmt.Sprintf("([subtask #%d](%s))", subtaskID, taskURL(baseURL, taskID))
}

// extractSubtaskLink removes a subtask link from an item title and returns the
// cleaned title, the linked subtask ID (0 if there is none) and the link as
// it was written.
func extractSubtaskLink(title string) (string, int, string) {
	m := subtaskLinkRe.FindStringSubmatchIndex(title)
	if m == nil {
		return title, 0, ""
	}
	var idText string
	switch {
	case m[2] >= 0:
		idText = title[m[2]:m[3]]
	case m[4] >= 0:
		idText = title[m[4]:m[5]]
	}
	id, err := strconv.Atoi(idText)
	if err != nil {
		return title, 0, ""
	}
	cleaned := strings.TrimSpace(title[:m[0]] + title[m[1]:])
	return cleaned, id, strings.TrimSpace(title[m[0]:m[1]])
}

// setLinksOnLines sets the subtask link of the given zero-based lines of
// text: an existing subtask link on the line is removed and the new link is
// appended (separated by a space). Line endings are preserved.
func setLinksOnLines(text string, links map[int]string) string {
	if len(links) == 0 {
		return text
	}
	var b strings.Builder
	for i, line := range splitLinesKeepEnds(text) {
		link, ok := links[i]
		if !ok {
			b.WriteString(line)
			continue
		}
		content := strings.TrimRight(line, "\r\n")
		ending := line[len(content):]
		if loc := subtaskLinkRe.FindStringIndex(content); loc != nil {
			content = content[:loc[0]] + content[loc[1]:]
		}
		b.WriteString(strings.TrimRight(content, " \t"))
		b.WriteString(" ")
		b.WriteString(link)
		b.WriteString(ending)
	}
	return b.String()
}

// checklistItemRe matches Markdown task-list items: a bullet ("-", "*", "+")
// or ordered marker ("1.", "2)") followed by "[ ]", "[x]" or "[X]" and the
// item title. Leading indentation (nested lists) and blockquote markers are
// allowed.
var checklistItemRe = regexp.MustCompile(
	`^[\t >]*(?:[-*+]|\d{1,9}[.)])[ \t]+\[([ xX])\](?:[ \t]+(.*))?$`,
)

// parseChecklist extracts all Markdown checkbox items from text. Items inside
// fenced code blocks are ignored, as are items without a title.
func parseChecklist(text string) []ChecklistItem {
	var items []ChecklistItem
	var fence string

	for i, line := range splitLines(text) {
		trimmed := strings.TrimLeft(line, " \t")
		if marker := fenceMarker(trimmed); marker != "" {
			switch {
			case fence == "":
				fence = marker
			case strings.HasPrefix(trimmed, fence) &&
				strings.TrimSpace(strings.TrimLeft(trimmed, fence[:1])) == "":
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}

		m := checklistItemRe.FindStringSubmatch(strings.TrimRight(line, " \t\r"))
		if m == nil {
			continue
		}
		title, linkedID, link := extractSubtaskLink(strings.TrimSpace(m[2]))
		if title == "" {
			continue
		}
		items = append(items, ChecklistItem{
			Title:           title,
			Checked:         m[1] != " ",
			Line:            i,
			LinkedSubtaskID: linkedID,
			Link:            link,
		})
	}
	return items
}

// removeLines returns text with the given zero-based line indices removed.
// Line endings of the remaining lines are preserved.
func removeLines(text string, lines []int) string {
	if len(lines) == 0 {
		return text
	}
	drop := make(map[int]bool, len(lines))
	for _, l := range lines {
		drop[l] = true
	}
	var b strings.Builder
	for i, line := range splitLinesKeepEnds(text) {
		if !drop[i] {
			b.WriteString(line)
		}
	}
	result := strings.TrimRight(b.String(), "\r\n")
	if strings.TrimSpace(result) == "" {
		return ""
	}
	return result + trailingNewline(text)
}

// fenceMarker returns the opening run of ``` or ~~~ (3 or more) if line starts
// a fenced code block, otherwise "".
func fenceMarker(line string) string {
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(line) && line[n] == ch {
			n++
		}
		if n >= 3 {
			return line[:n]
		}
	}
	return ""
}

func splitLines(text string) []string {
	lines := splitLinesKeepEnds(text)
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, "\r\n")
	}
	return lines
}

func splitLinesKeepEnds(text string) []string {
	if text == "" {
		return nil
	}
	return strings.SplitAfter(text, "\n")
}

func trailingNewline(text string) string {
	switch {
	case strings.HasSuffix(text, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(text, "\n"):
		return "\n"
	default:
		return ""
	}
}
