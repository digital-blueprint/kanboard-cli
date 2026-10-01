package cmd

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tu-graz/kanboard-cli/internal/api"
)

// Values accepted by "clearable" flags to reset a field.
func isNone(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return true
	}
	return false
}

// ---- dates ------------------------------------------------------------------

// kanboardDateTime is the date format sent to Kanboard. Date-only values must
// include a time, otherwise Kanboard fills in the current time of day.
const kanboardDateTime = "2006-01-02 15:04"

// parseDateTimeArg parses a task date flag. It accepts "YYYY-MM-DD",
// "YYYY-MM-DD HH:MM", "YYYY-MM-DDTHH:MM", "now", "today", "tomorrow", and
// "none"/"" (clear). It returns the value to send to Kanboard ("" clears).
func parseDateTimeArg(value string, now time.Time) (string, error) {
	v := strings.TrimSpace(value)
	switch strings.ToLower(v) {
	case "", "none":
		return "", nil
	case "now":
		return now.Format(kanboardDateTime), nil
	case "today":
		return now.Format("2006-01-02") + " 00:00", nil
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format("2006-01-02") + " 00:00", nil
	}
	for _, layout := range []string{
		kanboardDateTime, "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02T15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t.Format(kanboardDateTime), nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
		return t.Format(kanboardDateTime), nil
	}
	return "", fmt.Errorf(
		"invalid date %q: expected YYYY-MM-DD, \"YYYY-MM-DD HH:MM\", now, today, tomorrow, or none",
		value,
	)
}

// parseDateArg parses a date-only flag (used for project dates). It returns
// "YYYY-MM-DD" or "" (clear).
func parseDateArg(value string, now time.Time) (string, error) {
	v := strings.TrimSpace(value)
	switch strings.ToLower(v) {
	case "", "none":
		return "", nil
	case "today":
		return now.Format("2006-01-02"), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD, today, or none", value)
}

// formatTaskTime formats a task timestamp for display.
func formatTaskTime(t api.FlexibleTime) string {
	if t.Time().IsZero() {
		return "none"
	}
	return t.Time().Format(kanboardDateTime)
}

// ---- numbers ----------------------------------------------------------------

// parseHours parses a duration in hours: "1.5", "1.5h", "90m", "1h30m", or
// "none"/"" (zero).
func parseHours(value string) (float64, error) {
	v := strings.TrimSpace(strings.ToLower(value))
	if isNone(v) {
		return 0, nil
	}
	var hours float64
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		hours = f
	} else if d, err := time.ParseDuration(v); err == nil {
		hours = d.Hours()
	} else {
		return 0, fmt.Errorf("invalid duration %q: expected hours like 1.5, 1.5h, 90m, or 1h30m", value)
	}
	if hours < 0 || math.IsNaN(hours) || math.IsInf(hours, 0) {
		return 0, fmt.Errorf("invalid duration %q: must not be negative", value)
	}
	return math.Round(hours*100) / 100, nil
}

func formatHours(hours float64) string {
	return strconv.FormatFloat(hours, 'f', -1, 64) + "h"
}

func numberToFloat(n fmt.Stringer) float64 {
	f, _ := strconv.ParseFloat(n.String(), 64)
	return f
}

func numberToInt(n fmt.Stringer) int {
	s := n.String()
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	f, _ := strconv.ParseFloat(s, 64)
	return int(f)
}

// parseIntArg parses an integer flag; "none"/"" yields 0.
func parseIntArg(value, flag string) (int, error) {
	if isNone(value) {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid --%s %q: expected an integer", flag, value)
	}
	return n, nil
}

// ---- enumerations -----------------------------------------------------------

// Kanboard recurrence constants (see TaskModel).
const (
	recurrenceStatusNone    = 0
	recurrenceStatusPending = 1

	recurrenceTriggerFirstColumn = 0
	recurrenceTriggerLastColumn  = 1
	recurrenceTriggerClose       = 2

	recurrenceTimeframeDays   = 0
	recurrenceTimeframeMonths = 1
	recurrenceTimeframeYears  = 2

	recurrenceBaseDueDate    = 0
	recurrenceBaseActionDate = 1
)

func parseRecurrenceStatus(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none", "off", "no", "false":
		return recurrenceStatusNone, nil
	case "pending", "on", "yes", "true":
		return recurrenceStatusPending, nil
	}
	return 0, fmt.Errorf("invalid --recurrence %q: expected on or off", value)
}

func recurrenceStatusLabel(n int) string {
	switch n {
	case recurrenceStatusNone:
		return "off"
	case recurrenceStatusPending:
		return "on"
	case 2:
		return "processed"
	}
	return strconv.Itoa(n)
}

func parseRecurrenceTrigger(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "first-column", "first":
		return recurrenceTriggerFirstColumn, nil
	case "last-column", "last":
		return recurrenceTriggerLastColumn, nil
	case "close", "closed":
		return recurrenceTriggerClose, nil
	}
	return 0, fmt.Errorf(
		"invalid --recurrence-trigger %q: expected close, first-column, or last-column",
		value,
	)
}

func recurrenceTriggerLabel(n int) string {
	switch n {
	case recurrenceTriggerFirstColumn:
		return "first-column"
	case recurrenceTriggerLastColumn:
		return "last-column"
	case recurrenceTriggerClose:
		return "close"
	}
	return strconv.Itoa(n)
}

// parseRecurrenceEvery parses an interval like "3d", "2 months", or "1y" into a
// Kanboard recurrence factor and timeframe.
func parseRecurrenceEvery(value string) (factor, timeframe int, err error) {
	v := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	i := 0
	for i < len(v) && v[i] >= '0' && v[i] <= '9' {
		i++
	}
	factor, convErr := strconv.Atoi(v[:i])
	if convErr != nil || factor <= 0 {
		return 0, 0, fmt.Errorf(
			"invalid --recurrence-every %q: expected a positive number with unit, e.g. 3d, 2m, 1y",
			value,
		)
	}
	switch v[i:] {
	case "d", "day", "days":
		return factor, recurrenceTimeframeDays, nil
	case "m", "mo", "month", "months":
		return factor, recurrenceTimeframeMonths, nil
	case "y", "year", "years":
		return factor, recurrenceTimeframeYears, nil
	}
	return 0, 0, fmt.Errorf(
		"invalid --recurrence-every %q: unit must be d (days), m (months), or y (years)",
		value,
	)
}

func recurrenceEveryLabel(factor, timeframe int) string {
	if factor == 0 {
		return "none"
	}
	unit := map[int]string{
		recurrenceTimeframeDays:   "d",
		recurrenceTimeframeMonths: "m",
		recurrenceTimeframeYears:  "y",
	}[timeframe]
	return strconv.Itoa(factor) + unit
}

func parseRecurrenceBase(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "due-date", "due":
		return recurrenceBaseDueDate, nil
	case "action-date", "action", "trigger-date":
		return recurrenceBaseActionDate, nil
	}
	return 0, fmt.Errorf("invalid --recurrence-base %q: expected due-date or action-date", value)
}

func recurrenceBaseLabel(n int) string {
	if n == recurrenceBaseActionDate {
		return "action-date"
	}
	return "due-date"
}

// parseOpenClosed parses --status for tasks and returns true for "open".
func parseOpenClosed(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open", "opened", "reopen", "active":
		return true, nil
	case "closed", "close", "done", "inactive":
		return false, nil
	}
	return false, fmt.Errorf("invalid --status %q: expected open or closed", value)
}

// parseBoolArg parses yes/no style flag values.
func parseBoolArg(value, flag string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "y", "true", "on", "1":
		return true, nil
	case "no", "n", "false", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("invalid --%s %q: expected yes or no", flag, value)
}

// taskColors are the color IDs built into Kanboard.
var taskColors = []string{
	"yellow", "blue", "green", "purple", "red", "orange", "grey", "brown",
	"deep_orange", "dark_grey", "pink", "teal", "cyan", "lime", "light_green", "amber",
}

// parseColor normalizes and validates a Kanboard color ID.
func parseColor(value string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.NewReplacer("-", "_", " ", "_", "gray", "grey").Replace(v)
	for _, c := range taskColors {
		if v == c {
			return c, nil
		}
	}
	return "", fmt.Errorf(
		"invalid --color %q: expected one of %s",
		value,
		strings.Join(taskColors, ", "),
	)
}

// ---- name lookups -----------------------------------------------------------

// lookup resolves user, category, column, and swimlane names to IDs. Results
// are cached per project so updating many tasks does not repeat API calls.
type lookup struct {
	client     *api.Client
	meID       *int
	projects   map[int]*api.Project
	assignable map[int]map[string]string
	members    map[int]map[string]string
	categories map[int][]api.Category
	columns    map[int][]api.Column
	swimlanes  map[int][]api.Swimlane
}

func newLookup(client *api.Client) *lookup {
	return &lookup{
		client:     client,
		projects:   map[int]*api.Project{},
		assignable: map[int]map[string]string{},
		members:    map[int]map[string]string{},
		categories: map[int][]api.Category{},
		columns:    map[int][]api.Column{},
		swimlanes:  map[int][]api.Swimlane{},
	}
}

func (l *lookup) me() (int, error) {
	if l.meID != nil {
		return *l.meID, nil
	}
	me, err := l.client.GetMe()
	if err != nil {
		return 0, fmt.Errorf("could not determine the current user (use a user ID): %w", err)
	}
	id, err := jsonNumberToInt(me.ID, "user ID")
	if err != nil {
		return 0, err
	}
	l.meID = &id
	return id, nil
}

func (l *lookup) project(projectID int) (*api.Project, error) {
	if p, ok := l.projects[projectID]; ok {
		return p, nil
	}
	p, err := l.client.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project %d not found", projectID)
	}
	l.projects[projectID] = p
	return p, nil
}

func (l *lookup) projectLabel(projectID int) string {
	if p, err := l.project(projectID); err == nil && p.Name != "" {
		return fmt.Sprintf("%s (#%d)", p.Name, projectID)
	}
	return fmt.Sprintf("#%d", projectID)
}

func (l *lookup) assignableUsers(projectID int) (map[string]string, error) {
	if users, ok := l.assignable[projectID]; ok {
		return users, nil
	}
	users, err := l.client.GetAssignableUsers(projectID)
	if err != nil {
		return nil, err
	}
	l.assignable[projectID] = users
	return users, nil
}

func (l *lookup) projectMembers(projectID int) (map[string]string, error) {
	if users, ok := l.members[projectID]; ok {
		return users, nil
	}
	users, err := l.client.GetProjectUsers(projectID)
	if err != nil {
		return nil, err
	}
	l.members[projectID] = users
	return users, nil
}

// resolveAssignee resolves a user for task assignment in a project. It
// accepts a user ID, "me", "none"/"nobody"/"unassigned", a display name (or
// unique part of it), or a username.
func (l *lookup) resolveAssignee(projectID int, value string) (int, error) {
	return l.resolveUser(value, func() (map[string]string, error) {
		return l.assignableUsers(projectID)
	}, fmt.Sprintf("assignable in project %d", projectID))
}

// resolveMember resolves a user among all members of a project.
func (l *lookup) resolveMember(projectID int, value string) (int, error) {
	return l.resolveUser(value, func() (map[string]string, error) {
		return l.projectMembers(projectID)
	}, fmt.Sprintf("a member of project %d", projectID))
}

func (l *lookup) resolveUser(
	value string,
	load func() (map[string]string, error),
	scope string,
) (int, error) {
	v := strings.TrimSpace(value)
	switch strings.ToLower(v) {
	case "", "none", "nobody", "unassigned", "0":
		return 0, nil
	case "me":
		return l.me()
	}
	if id, err := strconv.Atoi(v); err == nil && id > 0 {
		return id, nil
	}

	users, loadErr := load()
	if loadErr == nil {
		if id, err := matchName(v, users, "user"); err == nil {
			return id, nil
		} else if !strings.Contains(err.Error(), "not found") {
			return 0, err
		}
	}

	// Usernames are not part of the user lists above; getUserByName only works
	// with administrator rights, so failures are ignored.
	if u, err := l.client.GetUserByName(v); err == nil && u != nil {
		return jsonNumberToInt(u.ID, "user ID")
	}

	if loadErr != nil {
		return 0, fmt.Errorf(
			"user %q not found (could not list users: %v); use a user ID",
			v,
			loadErr,
		)
	}
	return 0, fmt.Errorf("user %q is not %s; known users: %s", v, scope, namesList(users))
}

func (l *lookup) userLabel(projectID, userID int) string {
	if userID == 0 {
		return "nobody"
	}
	if users, err := l.assignableUsers(projectID); err == nil {
		if name, ok := users[strconv.Itoa(userID)]; ok {
			return fmt.Sprintf("%s (#%d)", name, userID)
		}
	}
	return fmt.Sprintf("#%d", userID)
}

func (l *lookup) categoryList(projectID int) ([]api.Category, error) {
	if cats, ok := l.categories[projectID]; ok {
		return cats, nil
	}
	cats, err := l.client.GetAllCategories(projectID)
	if err != nil {
		return nil, err
	}
	l.categories[projectID] = cats
	return cats, nil
}

// resolveCategory resolves a category ID or name in a project; "none"
// yields 0 (no category).
func (l *lookup) resolveCategory(projectID int, value string) (int, error) {
	v := strings.TrimSpace(value)
	if isNone(v) || v == "0" {
		return 0, nil
	}
	cats, err := l.categoryList(projectID)
	if err != nil {
		return 0, err
	}
	names := make(map[string]string, len(cats))
	for _, c := range cats {
		names[c.ID.String()] = c.Name
	}
	if id, err := strconv.Atoi(v); err == nil {
		if _, ok := names[v]; !ok {
			return 0, fmt.Errorf("category %d does not exist in project %d; categories: %s",
				id, projectID, namesList(names))
		}
		return id, nil
	}
	id, err := matchName(v, names, "category")
	if err != nil {
		return 0, fmt.Errorf("%w in project %d; categories: %s", err, projectID, namesList(names))
	}
	return id, nil
}

func (l *lookup) categoryLabel(projectID, categoryID int) string {
	if categoryID == 0 {
		return "none"
	}
	if cats, err := l.categoryList(projectID); err == nil {
		for _, c := range cats {
			if c.ID.String() == strconv.Itoa(categoryID) {
				return fmt.Sprintf("%s (#%d)", c.Name, categoryID)
			}
		}
	}
	return fmt.Sprintf("#%d", categoryID)
}

func (l *lookup) columnList(projectID int) ([]api.Column, error) {
	if cols, ok := l.columns[projectID]; ok {
		return cols, nil
	}
	cols, err := l.client.GetColumns(projectID)
	if err != nil {
		return nil, err
	}
	l.columns[projectID] = cols
	return cols, nil
}

// resolveColumn resolves a column ID or exact title (case-insensitive).
func (l *lookup) resolveColumn(projectID int, value string) (int, error) {
	cols, err := l.columnList(projectID)
	if err != nil {
		return 0, err
	}
	names := make(map[string]string, len(cols))
	for _, c := range cols {
		names[c.ID.String()] = c.Title
	}
	v := strings.TrimSpace(value)
	if id, err := strconv.Atoi(v); err == nil {
		if _, ok := names[v]; !ok {
			return 0, fmt.Errorf("column %d does not exist in project %d; columns: %s",
				id, projectID, namesList(names))
		}
		return id, nil
	}
	id, err := matchExactName(v, names, "column")
	if err != nil {
		return 0, fmt.Errorf("%w in project %d; columns: %s", err, projectID, namesList(names))
	}
	return id, nil
}

func (l *lookup) firstColumn(projectID int) (int, error) {
	cols, err := l.columnList(projectID)
	if err != nil {
		return 0, err
	}
	if len(cols) == 0 {
		return 0, fmt.Errorf("project %d has no columns", projectID)
	}
	return jsonNumberToInt(cols[0].ID, "column ID")
}

func (l *lookup) columnLabel(projectID, columnID int) string {
	if cols, err := l.columnList(projectID); err == nil {
		for _, c := range cols {
			if c.ID.String() == strconv.Itoa(columnID) {
				return fmt.Sprintf("%s (#%d)", c.Title, columnID)
			}
		}
	}
	return fmt.Sprintf("#%d", columnID)
}

func (l *lookup) swimlaneList(projectID int) ([]api.Swimlane, error) {
	if lanes, ok := l.swimlanes[projectID]; ok {
		return lanes, nil
	}
	lanes, err := l.client.GetActiveSwimlanes(projectID)
	if err != nil {
		return nil, err
	}
	l.swimlanes[projectID] = lanes
	return lanes, nil
}

// resolveSwimlane resolves an active swimlane ID or exact name
// (case-insensitive).
func (l *lookup) resolveSwimlane(projectID int, value string) (int, error) {
	lanes, err := l.swimlaneList(projectID)
	if err != nil {
		return 0, err
	}
	names := make(map[string]string, len(lanes))
	for _, s := range lanes {
		names[s.ID.String()] = s.Name
	}
	v := strings.TrimSpace(value)
	if id, err := strconv.Atoi(v); err == nil {
		return id, nil
	}
	id, err := matchExactName(v, names, "swimlane")
	if err != nil {
		return 0, fmt.Errorf("%w in project %d; swimlanes: %s", err, projectID, namesList(names))
	}
	return id, nil
}

func (l *lookup) swimlaneLabel(projectID, swimlaneID int) string {
	if lanes, err := l.swimlaneList(projectID); err == nil {
		for _, s := range lanes {
			if s.ID.String() == strconv.Itoa(swimlaneID) {
				return fmt.Sprintf("%s (#%d)", s.Name, swimlaneID)
			}
		}
	}
	return fmt.Sprintf("#%d", swimlaneID)
}

// matchExactName finds the ID whose name equals value case-insensitively.
func matchExactName(value string, names map[string]string, what string) (int, error) {
	var matches []string
	for id, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), value) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%s %q not found", what, value)
	case 1:
		return strconv.Atoi(matches[0])
	}
	return 0, fmt.Errorf("%s %q is ambiguous (IDs %s); use the ID", what, value,
		strings.Join(sortedIDs(matches), ", "))
}

// matchName finds the ID whose name equals value case-insensitively, falling
// back to a unique case-insensitive substring match.
func matchName(value string, names map[string]string, what string) (int, error) {
	id, err := matchExactName(value, names, what)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		return id, err
	}
	needle := strings.ToLower(value)
	var matches []string
	for id, name := range names {
		if strings.Contains(strings.ToLower(name), needle) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%s %q not found", what, value)
	case 1:
		return strconv.Atoi(matches[0])
	}
	labels := make([]string, 0, len(matches))
	for _, id := range sortedIDs(matches) {
		labels = append(labels, fmt.Sprintf("%s (#%s)", names[id], id))
	}
	return 0, fmt.Errorf("%s %q is ambiguous: %s", what, value, strings.Join(labels, ", "))
}

func sortedIDs(ids []string) []string {
	sort.Slice(ids, func(i, j int) bool {
		a, _ := strconv.Atoi(ids[i])
		b, _ := strconv.Atoi(ids[j])
		return a < b
	})
	return ids
}

// namesList formats an ID→name map as "name (#id), ..." sorted by name.
func namesList(names map[string]string) string {
	if len(names) == 0 {
		return "(none)"
	}
	labels := make([]string, 0, len(names))
	for id, name := range names {
		labels = append(labels, fmt.Sprintf("%s (#%s)", name, id))
	}
	sort.Slice(labels, func(i, j int) bool {
		return strings.ToLower(labels[i]) < strings.ToLower(labels[j])
	})
	return strings.Join(labels, ", ")
}

func (l *lookup) memberLabel(projectID, userID int) string {
	if userID == 0 {
		return "nobody"
	}
	if users, err := l.projectMembers(projectID); err == nil {
		if name, ok := users[strconv.Itoa(userID)]; ok {
			return fmt.Sprintf("%s (#%d)", name, userID)
		}
	}
	return fmt.Sprintf("#%d", userID)
}
