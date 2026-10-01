package cmd

import (
	"fmt"
	"strings"

	"github.com/tu-graz/kanboard-cli/internal/api"
)

// printTaskDetails prints the task fields (without description/subtasks).
// Names are looked up on a best-effort basis and fall back to IDs.
func printTaskDetails(l *lookup, t *api.Task) {
	projectID := numberToInt(t.ProjectID)
	row := func(label, value string) {
		fmt.Printf("%-13s%s\n", label+":", value)
	}
	status := "open"
	if t.IsActive.String() == "0" {
		status = "closed"
	}

	row("ID", t.ID.String())
	row("Title", t.Title)
	row("Status", status)
	row("Project", l.projectLabel(projectID))
	row("Column", l.columnLabel(projectID, numberToInt(t.ColumnID)))
	if lane := numberToInt(t.SwimlaneID); lane != 0 {
		row("Swimlane", l.swimlaneLabel(projectID, lane))
	}
	row("Position", t.Position.String())
	row("Assignee", l.userLabel(projectID, numberToInt(t.OwnerID)))
	row("Category", l.categoryLabel(projectID, numberToInt(t.CategoryID)))
	row("Color", t.ColorID)
	row("Priority", t.Priority.String())
	if score := numberToInt(t.Score); score != 0 {
		row("Complexity", t.Score.String())
	}
	row("Start", formatTaskTime(t.DateStarted))
	row("Due", formatTaskTime(t.DateDue))
	if est, spent := numberToFloat(t.TimeEstimated), numberToFloat(t.TimeSpent); est != 0 ||
		spent != 0 {
		row("Time", fmt.Sprintf("%s spent of %s estimated", formatHours(spent), formatHours(est)))
	}
	if t.Reference != "" {
		row("Reference", t.Reference)
	}
	if status := numberToInt(t.RecurrenceStatus); status != recurrenceStatusNone {
		row("Recurrence", fmt.Sprintf(
			"%s, every %s, on %s, based on %s",
			recurrenceStatusLabel(status),
			recurrenceEveryLabel(
				numberToInt(t.RecurrenceFactor),
				numberToInt(t.RecurrenceTimeframe),
			),
			recurrenceTriggerLabel(numberToInt(t.RecurrenceTrigger)),
			recurrenceBaseLabel(numberToInt(t.RecurrenceBasedate)),
		))
	}
	if len(t.Tags) > 0 {
		row("Tags", strings.Join(sortedTagNames(t.Tags), ", "))
	}
	if t.URL != "" {
		row("URL", t.URL)
	}
}
