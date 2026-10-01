package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
)

// taskUpdateResult is the outcome of updating one task.
type taskUpdateResult struct {
	TaskID  int           `json:"task_id"`
	Updated bool          `json:"updated"`
	DryRun  bool          `json:"dry_run,omitempty"`
	Changes []fieldChange `json:"changes"`
}

type taskUpdateOptions struct {
	fields   taskFieldFlags
	project  string
	column   string
	swimlane string
	position int
	status   string
	dryRun   bool
}

func newTaskUpdateCmd() *cobra.Command {
	var o taskUpdateOptions

	cmd := &cobra.Command{
		Use:     "update <task-id> [task-id...]",
		Aliases: []string{"edit"},
		Short:   "Change fields, tags, board placement, or status of one or more tasks",
		Long: `Change one or more tasks. Only the given flags are changed; everything else
is left as it is. Fields that already have the requested value are skipped.

Values that can be removed accept "none" (or ""), e.g. --due none,
--assignee none, --category none.

Users, categories, projects, columns, and swimlanes can be given by ID or by
name (case-insensitive). Assignees also accept "me" and usernames.

When several changes are requested they are applied in this order: move to
another project, move on the board, update fields and tags, open/close.`,
		Example: `  # Edit fields
  kanboard-cli task update 42 --title "New title" --priority 2 --due 2026-10-15
  kanboard-cli task update 42 --assignee me --category Bug --color red
  kanboard-cli task update 42 --estimate 1.5 --spent 30m

  # Description
  kanboard-cli task update 42 -d "New description"
  kanboard-cli task update 42 -F notes.md          # -F - reads stdin
  kanboard-cli task update 42 -e                   # opens $EDITOR
  kanboard-cli task update 42 --append-description "- [ ] Another item"

  # Tags
  kanboard-cli task update 42 --tag urgent --untag later
  kanboard-cli task update 42 --set-tags bug,ui

  # Board placement and status
  kanboard-cli task update 42 --column Done --status closed
  kanboard-cli task update 42 --project "Software Solutions" --swimlane "Team A"

  # Several tasks at once, preview first
  kanboard-cli task update 41 42 43 --tag sprint-7 --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskUpdate(cmd, &o, args)
		},
	}
	o.fields.register(cmd, true)
	fl := cmd.Flags()
	fl.StringVarP(&o.project, "project", "p", "", "Move to project (ID or name)")
	fl.StringVarP(&o.column, "column", "c", "", "Move to column (ID or name)")
	fl.StringVar(&o.swimlane, "swimlane", "", "Move to swimlane (ID or name)")
	fl.IntVar(&o.position, "position", 0, "Position in the column (1 = top)")
	fl.StringVar(&o.status, "status", "", "Set status: open or closed")
	fl.BoolVarP(&o.dryRun, "dry-run", "n", false, "Show what would change without saving")
	return cmd
}

var placementFlagNames = []string{"project", "column", "swimlane", "position"}

func runTaskUpdate(cmd *cobra.Command, o *taskUpdateOptions, args []string) error {
	taskIDs := make([]int, 0, len(args))
	for _, arg := range args {
		id, err := parseID(arg, "task ID")
		if err != nil {
			return err
		}
		taskIDs = append(taskIDs, id)
	}

	f := &o.fields
	if !f.changed(fieldFlagNames...) && !f.changed(placementFlagNames...) && !f.changed("status") {
		return errors.New("nothing to update: pass at least one field flag (see --help)")
	}
	if f.edit && len(taskIDs) > 1 {
		return errors.New("--edit can only be used with a single task")
	}
	if f.changed("position") && o.position < 1 {
		return errors.New("--position must be 1 or greater")
	}
	if f.changed("project") && strings.TrimSpace(o.project) == "" {
		return errors.New("--project cannot be empty")
	}
	var wantOpen bool
	if f.changed("status") {
		var err error
		if wantOpen, err = parseOpenClosed(o.status); err != nil {
			return err
		}
	}
	if err := f.validate(); err != nil {
		return err
	}
	if err := f.loadDescriptionFile(); err != nil {
		return err
	}

	client := newClient()
	l := newLookup(client)

	targetProject := 0
	if f.changed("project") {
		id, err := resolveProjectID(client, o.project)
		if err != nil {
			return err
		}
		targetProject = id
	}

	results := make([]taskUpdateResult, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		res, err := updateOneTask(client, l, o, taskID, targetProject, wantOpen)
		if res != nil {
			results = append(results, *res)
			if !jsonOutput {
				printTaskUpdateResult(*res)
			}
		}
		if err != nil {
			if jsonOutput && len(results) > 0 {
				printJSON(results)
			}
			return fmt.Errorf("task %d: %w", taskID, err)
		}
	}
	if jsonOutput {
		printJSON(results)
	}
	return nil
}

func updateOneTask(
	client *api.Client,
	l *lookup,
	o *taskUpdateOptions,
	taskID, targetProject int,
	wantOpen bool,
) (*taskUpdateResult, error) {
	f := &o.fields
	res := &taskUpdateResult{TaskID: taskID, DryRun: o.dryRun, Changes: []fieldChange{}}
	add := func(field, from, to string) {
		res.Changes = append(res.Changes, fieldChange{Field: field, From: from, To: to})
	}

	task, err := getTaskOrErr(client, taskID)
	if err != nil {
		return nil, err
	}
	projectID := numberToInt(task.ProjectID)

	// 1. Move to another project.
	movedProject := false
	if targetProject != 0 && targetProject != projectID {
		add("project", l.projectLabel(projectID), l.projectLabel(targetProject))
		if !o.dryRun {
			if err := client.MoveTaskToProject(taskID, targetProject); err != nil {
				return res, fmt.Errorf("move to project %d: %w", targetProject, err)
			}
			res.Updated = true
			if task, err = getTaskOrErr(client, taskID); err != nil {
				return res, err
			}
		}
		projectID = targetProject
		movedProject = true
	}

	// 2. Move on the board.
	if f.changed("column", "swimlane", "position") {
		curCol, curLane, curPos := numberToInt(task.ColumnID), numberToInt(task.SwimlaneID),
			numberToInt(task.Position)
		if movedProject && o.dryRun {
			// The column/swimlane in the new project is only known after moving.
			curCol, curLane, curPos = 0, 0, 0
		}
		col, lane, pos := curCol, curLane, curPos
		if f.changed("column") {
			if col, err = l.resolveColumn(projectID, o.column); err != nil {
				return res, err
			}
		}
		if f.changed("swimlane") {
			if lane, err = l.resolveSwimlane(projectID, o.swimlane); err != nil {
				return res, err
			}
		}
		if col == 0 {
			if col, err = l.firstColumn(projectID); err != nil {
				return res, err
			}
		}
		switch {
		case f.changed("position"):
			pos = o.position
		case col != curCol || lane != curLane:
			pos = 1
		}
		if col != curCol || lane != curLane || pos != curPos {
			if col != curCol {
				add(
					"column",
					labelOrUnknown(
						curCol,
						func(n int) string { return l.columnLabel(projectID, n) },
					),
					l.columnLabel(projectID, col),
				)
			}
			if lane != curLane && lane != 0 {
				add(
					"swimlane",
					labelOrUnknown(
						curLane,
						func(n int) string { return l.swimlaneLabel(projectID, n) },
					),
					l.swimlaneLabel(projectID, lane),
				)
			}
			if pos != curPos {
				add("position", labelOrUnknown(curPos, strconv.Itoa), strconv.Itoa(pos))
			}
			if !o.dryRun {
				if lane == 0 {
					lane = curLane
				}
				if err := client.MoveTaskPosition(api.MoveTaskPositionParams{
					ProjectID:  projectID,
					TaskID:     taskID,
					ColumnID:   col,
					Position:   pos,
					SwimlaneID: lane,
				}); err != nil {
					return res, fmt.Errorf("move on board: %w", err)
				}
				res.Updated = true
			}
		}
	}

	// 3. Fields and tags.
	params, changes, err := f.buildUpdate(l, task, projectID, func() ([]string, error) {
		tags, err := client.GetTaskTags(taskID)
		if err != nil {
			return nil, err
		}
		return sortedTagNames(tags), nil
	})
	if err != nil {
		return res, err
	}
	res.Changes = append(res.Changes, changes...)
	if !params.IsEmpty() && !o.dryRun {
		if err := client.UpdateTask(params); err != nil {
			return res, fmt.Errorf(
				"%w (Kanboard rejected the values; e.g. the start date must not be after the due date)",
				err,
			)
		}
		res.Updated = true
		warnIfTimeNotSaved(client, taskID, params)
	}

	// 4. Open / close.
	if f.changed("status") {
		isOpen := task.IsActive.String() != "0"
		if isOpen != wantOpen {
			add("status", openClosedLabel(isOpen), openClosedLabel(wantOpen))
			if !o.dryRun {
				if wantOpen {
					err = client.OpenTask(taskID)
				} else {
					err = client.CloseTask(taskID)
				}
				if err != nil {
					return res, err
				}
				res.Updated = true
			}
		}
	}
	return res, nil
}

// warnIfTimeNotSaved warns when Kanboard silently ignored time fields, which
// older Kanboard versions do not accept in updateTask.
func warnIfTimeNotSaved(client *api.Client, taskID int, p api.UpdateTaskParams) {
	if p.TimeEstimated == nil && p.TimeSpent == nil {
		return
	}
	task, err := client.GetTask(taskID)
	if err != nil || task.ID.String() == "" {
		return
	}
	if p.TimeEstimated != nil && numberToFloat(task.TimeEstimated) != *p.TimeEstimated ||
		p.TimeSpent != nil && numberToFloat(task.TimeSpent) != *p.TimeSpent {
		fmt.Fprintf(os.Stderr,
			"Warning: task %d: the server did not save the estimated/spent time "+
				"(this Kanboard version may not support changing them via the API)\n", taskID)
	}
}

func getTaskOrErr(client *api.Client, taskID int) (*api.Task, error) {
	task, err := client.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	if task.ID.String() == "" {
		return nil, fmt.Errorf("task %d not found", taskID)
	}
	return task, nil
}

func labelOrUnknown(n int, label func(int) string) string {
	if n == 0 {
		return "?"
	}
	return label(n)
}

func openClosedLabel(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}

func printTaskUpdateResult(res taskUpdateResult) {
	if len(res.Changes) == 0 {
		fmt.Printf("Task %d unchanged\n", res.TaskID)
		return
	}
	switch {
	case res.DryRun:
		fmt.Printf("Task %d would change (dry run):\n", res.TaskID)
	case res.Updated:
		fmt.Printf("Task %d updated:\n", res.TaskID)
	default:
		fmt.Printf("Task %d:\n", res.TaskID)
	}
	for _, c := range res.Changes {
		from, to := c.From, c.To
		if c.Field == "description" {
			from, to = summarizeText(from), summarizeText(to)
		}
		fmt.Printf("  %-19s %s → %s\n", c.Field+":", from, to)
	}
}

// summarizeText shortens long multi-line text for display.
func summarizeText(s string) string {
	if s == "" {
		return "(empty)"
	}
	first := strings.SplitN(s, "\n", 2)[0]
	runes := []rune(first)
	if len(runes) > 40 {
		first = string(runes[:37]) + "..."
	} else if len(first) < len(s) {
		first += " ..."
	}
	return fmt.Sprintf("%q (%d chars)", first, len([]rune(s)))
}
