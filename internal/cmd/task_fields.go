package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
)

// taskFieldFlags holds the task field flags shared by "task create" and
// "task update". All values are strings so that "none" can clear a field.
type taskFieldFlags struct {
	title             string
	description       string
	descriptionFile   string
	edit              bool
	appendDescription string
	color             string
	assignee          string
	category          string
	priority          string
	complexity        string
	reference         string
	due               string
	start             string
	estimate          string
	spent             string
	recurrence        string
	recurrenceTrigger string
	recurrenceEvery   string
	recurrenceBase    string
	tags              []string
	untags            []string
	setTags           []string
	clearTags         bool

	cmd *cobra.Command
	now func() time.Time
}

// register adds the field flags to cmd. update enables flags that only make
// sense for existing tasks.
func (f *taskFieldFlags) register(cmd *cobra.Command, update bool) {
	f.cmd = cmd
	f.now = time.Now
	fl := cmd.Flags()

	if update {
		fl.StringVar(&f.title, "title", "", "New title")
	}
	fl.StringVarP(&f.description, "description", "d", "", "Description (Markdown); \"\" clears it")
	fl.StringVarP(&f.descriptionFile, "description-file", "F", "",
		`Read the description from a file ("-" for stdin)`)
	if update {
		fl.BoolVarP(&f.edit, "edit", "e", false,
			"Edit the description in $VISUAL/$EDITOR (single task only)")
		fl.StringVar(&f.appendDescription, "append-description", "",
			"Append text to the description")
	} else {
		fl.BoolVarP(&f.edit, "edit", "e", false, "Write the description in $VISUAL/$EDITOR")
	}
	fl.StringVar(&f.color, "color", "", "Color: "+strings.Join(taskColors, ", "))
	fl.StringVarP(&f.assignee, "assignee", "a", "",
		"Assignee: user ID, name, username, me, or none")
	fl.StringVar(&f.category, "category", "", "Category ID or name, or none")
	fl.StringVar(&f.priority, "priority", "", "Priority (within the project's priority range)")
	fl.StringVar(&f.complexity, "complexity", "", "Complexity (story points)")
	fl.StringVar(&f.complexity, "score", "", "Alias for --complexity")
	_ = fl.MarkHidden("score")
	fl.StringVar(
		&f.reference,
		"reference",
		"",
		"External reference, e.g. a ticket ID; \"\" clears it",
	)
	fl.StringVar(
		&f.due,
		"due",
		"",
		`Due date: YYYY-MM-DD, "YYYY-MM-DD HH:MM", today, tomorrow, or none`,
	)
	fl.StringVar(
		&f.start,
		"start",
		"",
		`Start date: YYYY-MM-DD, "YYYY-MM-DD HH:MM", now, today, or none`,
	)
	fl.StringVar(&f.estimate, "estimate", "", "Estimated time in hours (1.5, 90m, 1h30m) or none")
	fl.StringVar(&f.spent, "spent", "", "Time spent in hours (1.5, 90m, 1h30m) or none")
	fl.StringVar(&f.recurrence, "recurrence", "", "Recurrence: on or off")
	fl.StringVar(&f.recurrenceTrigger, "recurrence-trigger", "",
		"Create the next task when the task is: close, first-column, last-column")
	fl.StringVar(&f.recurrenceEvery, "recurrence-every", "",
		"Recurrence interval, e.g. 7d, 1m, 1y (d=days, m=months, y=years)")
	fl.StringVar(&f.recurrenceBase, "recurrence-base", "",
		"Compute the next due date from: due-date or action-date")

	if update {
		fl.StringSliceVarP(&f.tags, "tag", "t", nil, "Add tags (repeatable or comma-separated)")
		fl.StringSliceVar(&f.untags, "untag", nil, "Remove tags (repeatable or comma-separated)")
		fl.StringSliceVar(&f.setTags, "set-tags", nil, "Replace all tags (comma-separated)")
		fl.BoolVar(&f.clearTags, "clear-tags", false, "Remove all tags")
		cmd.MarkFlagsMutuallyExclusive("set-tags", "clear-tags")
		cmd.MarkFlagsMutuallyExclusive(
			"description",
			"description-file",
			"edit",
			"append-description",
		)
	} else {
		fl.StringSliceVarP(&f.tags, "tag", "t", nil, "Tags (repeatable or comma-separated)")
		cmd.MarkFlagsMutuallyExclusive("description", "description-file", "edit")
	}
}

func (f *taskFieldFlags) changed(names ...string) bool {
	for _, name := range names {
		if fl := f.cmd.Flags().Lookup(name); fl != nil && fl.Changed {
			return true
		}
	}
	return false
}

// fieldFlagNames lists the flags handled by taskFieldFlags.
var fieldFlagNames = []string{
	"title", "description", "description-file", "edit", "append-description", "color",
	"assignee", "category", "priority", "complexity", "score", "reference", "due", "start",
	"estimate", "spent", "recurrence", "recurrence-trigger", "recurrence-every",
	"recurrence-base", "tag", "untag", "set-tags", "clear-tags",
}

// validate parses all set values once, so errors are reported before any task
// is modified.
func (f *taskFieldFlags) validate() error {
	if f.changed("title") && strings.TrimSpace(f.title) == "" {
		return fmt.Errorf("--title cannot be empty")
	}
	if f.changed("color") {
		c, err := parseColor(f.color)
		if err != nil {
			return err
		}
		f.color = c
	}
	checks := []struct {
		flag string
		fn   func() error
	}{
		{"priority", func() error { _, err := parseIntArg(f.priority, "priority"); return err }},
		{
			"complexity",
			func() error { _, err := parseIntArg(f.complexity, "complexity"); return err },
		},
		{"score", func() error { _, err := parseIntArg(f.complexity, "complexity"); return err }},
		{"due", func() error { _, err := parseDateTimeArg(f.due, f.now()); return err }},
		{"start", func() error { _, err := parseDateTimeArg(f.start, f.now()); return err }},
		{"estimate", func() error { _, err := parseHours(f.estimate); return err }},
		{"spent", func() error { _, err := parseHours(f.spent); return err }},
		{"recurrence", func() error { _, err := parseRecurrenceStatus(f.recurrence); return err }},
		{
			"recurrence-trigger",
			func() error { _, err := parseRecurrenceTrigger(f.recurrenceTrigger); return err },
		},
		{
			"recurrence-every",
			func() error { _, _, err := parseRecurrenceEvery(f.recurrenceEvery); return err },
		},
		{
			"recurrence-base",
			func() error { _, err := parseRecurrenceBase(f.recurrenceBase); return err },
		},
	}
	for _, c := range checks {
		if f.changed(c.flag) {
			if err := c.fn(); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadDescriptionFile reads --description-file once and stores it as
// --description.
func (f *taskFieldFlags) loadDescriptionFile() error {
	if !f.changed("description-file") {
		return nil
	}
	text, err := readTextFile(f.descriptionFile)
	if err != nil {
		return err
	}
	f.description = text
	return nil
}

func (f *taskFieldFlags) descriptionSet() bool {
	return f.changed("description", "description-file")
}

// checkPriority verifies a priority against the project's configured range.
func checkPriority(l *lookup, projectID, priority int) error {
	p, err := l.project(projectID)
	if err != nil {
		// Range check is best effort; Kanboard itself does not enforce it.
		return nil //nolint:nilerr
	}
	start, end := numberToInt(p.PriorityStart), numberToInt(p.PriorityEnd)
	if start > end {
		start, end = end, start
	}
	if start == end && start == 0 {
		return nil
	}
	if priority < start || priority > end {
		return fmt.Errorf("--priority %d is outside the range %d..%d of project %d",
			priority, start, end, projectID)
	}
	return nil
}

// buildCreate converts the flags into createTask parameters.
func (f *taskFieldFlags) buildCreate(l *lookup, p *api.CreateTaskParams) error {
	projectID := p.ProjectID
	if f.descriptionSet() {
		p.Description = f.description
	}
	if f.edit {
		text, err := editInEditor("", "kanboard-description-*.md")
		if err != nil {
			return err
		}
		p.Description = text
	}
	if f.changed("color") {
		p.ColorID = f.color
	}
	if f.changed("assignee") {
		id, err := l.resolveAssignee(projectID, f.assignee)
		if err != nil {
			return err
		}
		p.OwnerID = id
	}
	if f.changed("category") {
		id, err := l.resolveCategory(projectID, f.category)
		if err != nil {
			return err
		}
		p.CategoryID = id
	}
	if f.changed("priority") {
		n, _ := parseIntArg(f.priority, "priority")
		if err := checkPriority(l, projectID, n); err != nil {
			return err
		}
		p.Priority = &n
	}
	if f.changed("complexity", "score") {
		p.Score, _ = parseIntArg(f.complexity, "complexity")
	}
	if f.changed("reference") {
		p.Reference = f.reference
	}
	if f.changed("due") {
		p.DateDue, _ = parseDateTimeArg(f.due, f.now())
	}
	if f.changed("start") {
		p.DateStarted, _ = parseDateTimeArg(f.start, f.now())
	}
	if f.changed("estimate") {
		h, _ := parseHours(f.estimate)
		p.TimeEstimated = &h
	}
	if f.changed("spent") {
		h, _ := parseHours(f.spent)
		p.TimeSpent = &h
	}
	if f.changed("recurrence") {
		p.RecurrenceStatus, _ = parseRecurrenceStatus(f.recurrence)
	}
	if f.changed("recurrence-trigger") {
		p.RecurrenceTrigger, _ = parseRecurrenceTrigger(f.recurrenceTrigger)
	}
	if f.changed("recurrence-every") {
		p.RecurrenceFactor, p.RecurrenceTimeframe, _ = parseRecurrenceEvery(f.recurrenceEvery)
	}
	if f.changed("recurrence-base") {
		p.RecurrenceBasedate, _ = parseRecurrenceBase(f.recurrenceBase)
	}
	p.Tags = normalizeTags(f.tags)
	return nil
}

// fieldChange describes a change of one task field.
type fieldChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// buildUpdate compares the flags against task and returns updateTask
// parameters containing only fields that actually change. currentTags is only
// used when tag flags are set.
func (f *taskFieldFlags) buildUpdate(
	l *lookup,
	task *api.Task,
	projectID int,
	currentTags func() ([]string, error),
) (api.UpdateTaskParams, []fieldChange, error) {
	taskID := numberToInt(task.ID)
	p := api.UpdateTaskParams{ID: taskID}
	var changes []fieldChange
	add := func(field, from, to string) {
		changes = append(changes, fieldChange{Field: field, From: from, To: to})
	}
	setString := func(field, current, value string, target **string) {
		if current != value {
			v := value
			*target = &v
			add(field, current, value)
		}
	}
	setInt := func(field string, current, value int, target **int, label func(int) string) {
		if current != value {
			v := value
			*target = &v
			add(field, label(current), label(value))
		}
	}
	itoa := strconv.Itoa

	if f.changed("title") {
		setString("title", task.Title, strings.TrimSpace(f.title), &p.Title)
	}

	// Description.
	switch {
	case f.descriptionSet():
		setString("description", task.Description, f.description, &p.Description)
	case f.changed("append-description"):
		setString("description", task.Description,
			appendDescription(task.Description, f.appendDescription), &p.Description)
	case f.edit:
		text, err := editInEditor(task.Description, "kanboard-description-*.md")
		if err != nil {
			return p, nil, err
		}
		setString("description", task.Description, text, &p.Description)
	}

	if f.changed("color") {
		setString("color", task.ColorID, f.color, &p.ColorID)
	}
	if f.changed("assignee") {
		id, err := l.resolveAssignee(projectID, f.assignee)
		if err != nil {
			return p, nil, err
		}
		setInt("assignee", numberToInt(task.OwnerID), id, &p.OwnerID, func(n int) string {
			return l.userLabel(projectID, n)
		})
	}
	if f.changed("category") {
		id, err := l.resolveCategory(projectID, f.category)
		if err != nil {
			return p, nil, err
		}
		setInt("category", numberToInt(task.CategoryID), id, &p.CategoryID, func(n int) string {
			return l.categoryLabel(projectID, n)
		})
	}
	if f.changed("priority") {
		n, _ := parseIntArg(f.priority, "priority")
		if err := checkPriority(l, projectID, n); err != nil {
			return p, nil, err
		}
		setInt("priority", numberToInt(task.Priority), n, &p.Priority, itoa)
	}
	if f.changed("complexity", "score") {
		n, _ := parseIntArg(f.complexity, "complexity")
		setInt("complexity", numberToInt(task.Score), n, &p.Score, itoa)
	}
	if f.changed("reference") {
		setString("reference", task.Reference, strings.TrimSpace(f.reference), &p.Reference)
	}
	setDate := func(field, value string, current api.FlexibleTime, target **string) {
		v, _ := parseDateTimeArg(value, f.now())
		from, to := formatTaskTime(current), v
		if to == "" {
			to = "none"
		}
		if from != to {
			*target = &v
			add(field, from, to)
		}
	}
	if f.changed("due") {
		setDate("due", f.due, task.DateDue, &p.DateDue)
	}
	if f.changed("start") {
		setDate("start", f.start, task.DateStarted, &p.DateStarted)
	}
	setHours := func(field, value string, current fmt.Stringer, target **float64) {
		h, _ := parseHours(value)
		cur := numberToFloat(current)
		if cur != h {
			*target = &h
			add(field, formatHours(cur), formatHours(h))
		}
	}
	if f.changed("estimate") {
		setHours("estimate", f.estimate, task.TimeEstimated, &p.TimeEstimated)
	}
	if f.changed("spent") {
		setHours("spent", f.spent, task.TimeSpent, &p.TimeSpent)
	}
	if f.changed("recurrence") {
		n, _ := parseRecurrenceStatus(f.recurrence)
		setInt("recurrence", numberToInt(task.RecurrenceStatus), n, &p.RecurrenceStatus,
			recurrenceStatusLabel)
	}
	if f.changed("recurrence-trigger") {
		n, _ := parseRecurrenceTrigger(f.recurrenceTrigger)
		setInt("recurrence-trigger", numberToInt(task.RecurrenceTrigger), n, &p.RecurrenceTrigger,
			recurrenceTriggerLabel)
	}
	if f.changed("recurrence-every") {
		factor, timeframe, _ := parseRecurrenceEvery(f.recurrenceEvery)
		curFactor, curTimeframe := numberToInt(
			task.RecurrenceFactor,
		), numberToInt(
			task.RecurrenceTimeframe,
		)
		if factor != curFactor || timeframe != curTimeframe {
			p.RecurrenceFactor, p.RecurrenceTimeframe = &factor, &timeframe
			add("recurrence-every", recurrenceEveryLabel(curFactor, curTimeframe),
				recurrenceEveryLabel(factor, timeframe))
		}
	}
	if f.changed("recurrence-base") {
		n, _ := parseRecurrenceBase(f.recurrenceBase)
		setInt("recurrence-base", numberToInt(task.RecurrenceBasedate), n, &p.RecurrenceBasedate,
			recurrenceBaseLabel)
	}

	if f.changed("tag", "untag", "set-tags", "clear-tags") {
		current, err := currentTags()
		if err != nil {
			return p, nil, err
		}
		updated := current
		switch {
		case f.clearTags:
			updated = []string{}
		case f.changed("set-tags"):
			updated = normalizeTags(f.setTags)
		}
		updated = normalizeTags(append(append([]string{}, updated...), f.tags...))
		if len(f.untags) > 0 {
			var missing []string
			updated, missing = removeTags(updated, f.untags)
			for _, name := range missing {
				fmt.Fprintf(os.Stderr, "Warning: task %d has no tag %q\n", taskID, name)
			}
		}
		if !sameTags(current, updated) {
			p.Tags = &updated
			add("tags", tagsLabel(current), tagsLabel(updated))
		}
	}
	return p, changes, nil
}

func tagsLabel(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, ", ")
}
