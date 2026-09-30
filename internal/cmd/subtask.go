package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
	"github.com/tu-graz/kanboard-cli/internal/config"
)

func newSubtaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtask",
		Short: "Manage task subtasks",
	}
	cmd.AddCommand(
		newSubtaskListCmd(),
		newSubtaskGetCmd(),
		newSubtaskAddCmd(),
		newSubtaskUpdateCmd(),
		newSubtaskDoneCmd(),
		newSubtaskDeleteCmd(),
		newSubtaskFromChecklistCmd(),
	)
	return cmd
}

// ---- helpers ----------------------------------------------------------------

// parseSubtaskStatus converts a user-supplied status into Kanboard's numeric
// subtask status.
func parseSubtaskStatus(s string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "0", "todo", "to-do", "open":
		return api.SubtaskStatusTodo, nil
	case "1", "in-progress", "inprogress", "progress", "started", "doing":
		return api.SubtaskStatusInProgress, nil
	case "2", "done", "closed", "finished":
		return api.SubtaskStatusDone, nil
	default:
		return 0, fmt.Errorf("invalid status %q: expected todo, in-progress, or done", s)
	}
}

func subtaskStatusLabel(status string) string {
	switch status {
	case "0":
		return "todo"
	case "1":
		return "in progress"
	case "2":
		return "done"
	default:
		return status
	}
}

func subtaskAssignee(s api.Subtask) string {
	switch {
	case s.Name != "" && s.Username != "":
		return s.Name + " (" + s.Username + ")"
	case s.Username != "":
		return s.Username
	case s.UserID.String() != "" && s.UserID.String() != "0":
		return "user " + s.UserID.String()
	default:
		return ""
	}
}

func subtaskHours(s api.Subtask) string {
	est, spent := s.TimeEstimated.String(), s.TimeSpent.String()
	if est == "" {
		est = "0"
	}
	if spent == "" {
		spent = "0"
	}
	if est == "0" && spent == "0" {
		return ""
	}
	return spent + "h spent / " + est + "h est."
}

func renderSubtaskTable(subtasks []api.Subtask) error {
	table := tablewriter.NewTable(os.Stdout)
	table.Header("ID", "Status", "Title", "Assignee", "Time")
	for _, s := range subtasks {
		if err := table.Append(
			s.ID.String(),
			subtaskStatusLabel(s.Status.String()),
			s.Title,
			subtaskAssignee(s),
			subtaskHours(s),
		); err != nil {
			return err
		}
	}
	return table.Render()
}

func parseID(arg, label string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s: %s", label, arg)
	}
	return id, nil
}

func getSubtaskOrErr(client *api.Client, id int) (*api.Subtask, error) {
	s, err := client.GetSubtask(id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("subtask %d not found", id)
	}
	return s, nil
}

// ---- commands ---------------------------------------------------------------

func newSubtaskListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <task-id>",
		Short: "List all subtasks of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseID(args[0], "task ID")
			if err != nil {
				return err
			}
			subtasks, err := newClient().GetAllSubtasks(taskID)
			if err != nil {
				return err
			}
			if jsonOutput {
				if subtasks == nil {
					subtasks = []api.Subtask{}
				}
				printJSON(subtasks)
				return nil
			}
			if len(subtasks) == 0 {
				fmt.Println("No subtasks found.")
				return nil
			}
			return renderSubtaskTable(subtasks)
		},
	}
}

func newSubtaskGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <subtask-id>",
		Short: "Show details of a subtask",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "subtask ID")
			if err != nil {
				return err
			}
			s, err := getSubtaskOrErr(newClient(), id)
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(s)
				return nil
			}
			fmt.Printf("ID:             %s\n", s.ID)
			fmt.Printf("Task:           %s\n", s.TaskID)
			fmt.Printf("Title:          %s\n", s.Title)
			fmt.Printf("Status:         %s\n", subtaskStatusLabel(s.Status.String()))
			fmt.Printf("User ID:        %s\n", s.UserID)
			fmt.Printf("Time estimated: %sh\n", s.TimeEstimated)
			fmt.Printf("Time spent:     %sh\n", s.TimeSpent)
			fmt.Printf("Position:       %s\n", s.Position)
			return nil
		},
	}
}

func newSubtaskAddCmd() *cobra.Command {
	var userID int
	var timeEstimated, timeSpent float64
	var status string

	cmd := &cobra.Command{
		Use:   "add <task-id> <title> [title...]",
		Short: "Add one or more subtasks to a task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseID(args[0], "task ID")
			if err != nil {
				return err
			}
			statusID, err := parseSubtaskStatus(status)
			if err != nil {
				return err
			}
			client := newClient()

			created := make([]map[string]interface{}, 0, len(args)-1)
			for _, title := range args[1:] {
				title = strings.TrimSpace(title)
				if title == "" {
					return fmt.Errorf("subtask title cannot be empty")
				}
				id, err := client.CreateSubtask(api.CreateSubtaskParams{
					TaskID:        taskID,
					Title:         title,
					UserID:        userID,
					TimeEstimated: timeEstimated,
					TimeSpent:     timeSpent,
					Status:        statusID,
				})
				if err != nil {
					return fmt.Errorf("create subtask %q: %w", title, err)
				}
				created = append(created, map[string]interface{}{
					"subtask_id": id, "task_id": taskID, "title": title,
				})
				if !jsonOutput {
					fmt.Printf("Subtask %d added to task %d: %s\n", id, taskID, title)
				}
			}
			if jsonOutput {
				printJSON(created)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&userID, "user-id", "u", 0, "Assign subtask to this user ID")
	cmd.Flags().Float64Var(&timeEstimated, "time-estimated", 0, "Estimated time in hours")
	cmd.Flags().Float64Var(&timeSpent, "time-spent", 0, "Time spent in hours")
	cmd.Flags().StringVar(&status, "status", "todo", "Status: todo, in-progress, or done")
	return cmd
}

func newSubtaskUpdateCmd() *cobra.Command {
	var title, status string
	var userID int
	var timeEstimated, timeSpent float64

	cmd := &cobra.Command{
		Use:   "update <subtask-id>",
		Short: "Update a subtask (title, status, assignee, time)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "subtask ID")
			if err != nil {
				return err
			}
			flags := cmd.Flags()
			if !flags.Changed("title") && !flags.Changed("status") && !flags.Changed("user-id") &&
				!flags.Changed("time-estimated") && !flags.Changed("time-spent") {
				return fmt.Errorf(
					"nothing to update: pass at least one of --title, --status, --user-id, --time-estimated, --time-spent",
				)
			}

			client := newClient()
			s, err := getSubtaskOrErr(client, id)
			if err != nil {
				return err
			}
			taskID, err := jsonNumberToInt(s.TaskID, "task ID")
			if err != nil {
				return err
			}

			p := api.UpdateSubtaskParams{ID: id, TaskID: taskID}
			if flags.Changed("title") {
				title = strings.TrimSpace(title)
				if title == "" {
					return fmt.Errorf("--title cannot be empty")
				}
				p.Title = &title
			}
			if flags.Changed("status") {
				st, err := parseSubtaskStatus(status)
				if err != nil {
					return err
				}
				p.Status = &st
			}
			if flags.Changed("user-id") {
				p.UserID = &userID
			}
			if flags.Changed("time-estimated") {
				p.TimeEstimated = &timeEstimated
			}
			if flags.Changed("time-spent") {
				p.TimeSpent = &timeSpent
			}

			if err := client.UpdateSubtask(p); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(
					map[string]interface{}{"updated": true, "subtask_id": id, "task_id": taskID},
				)
				return nil
			}
			fmt.Printf("Subtask %d updated\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&status, "status", "", "New status: todo, in-progress, or done")
	cmd.Flags().IntVarP(&userID, "user-id", "u", 0, "Assign to this user ID (0 to unassign)")
	cmd.Flags().Float64Var(&timeEstimated, "time-estimated", 0, "Estimated time in hours")
	cmd.Flags().Float64Var(&timeSpent, "time-spent", 0, "Time spent in hours")
	return cmd
}

func newSubtaskDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <subtask-id> [subtask-id...]",
		Short: "Mark one or more subtasks as done",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := parseID(arg, "subtask ID")
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}
			client := newClient()
			done := api.SubtaskStatusDone
			result := make([]map[string]interface{}, 0, len(ids))
			for _, id := range ids {
				s, err := getSubtaskOrErr(client, id)
				if err != nil {
					return err
				}
				taskID, err := jsonNumberToInt(s.TaskID, "task ID")
				if err != nil {
					return err
				}
				if err := client.UpdateSubtask(api.UpdateSubtaskParams{
					ID: id, TaskID: taskID, Status: &done,
				}); err != nil {
					return fmt.Errorf("subtask %d: %w", id, err)
				}
				result = append(result, map[string]interface{}{"subtask_id": id, "status": "done"})
				if !jsonOutput {
					fmt.Printf("Subtask %d marked as done\n", id)
				}
			}
			if jsonOutput {
				printJSON(result)
			}
			return nil
		},
	}
}

func newSubtaskDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <subtask-id> [subtask-id...]",
		Short: "Delete one or more subtasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := parseID(arg, "subtask ID")
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}
			client := newClient()
			result := make([]map[string]interface{}, 0, len(ids))
			for _, id := range ids {
				if err := client.RemoveSubtask(id); err != nil {
					return fmt.Errorf("subtask %d: %w", id, err)
				}
				result = append(result, map[string]interface{}{"deleted": true, "subtask_id": id})
				if !jsonOutput {
					fmt.Printf("Subtask %d deleted\n", id)
				}
			}
			if jsonOutput {
				printJSON(result)
			}
			return nil
		},
	}
}

// Actions reported for each checklist item by subtask from-checklist.
const (
	actionLinked         = "linked"            // line already links to an existing subtask
	actionMissing        = "missing"           // line links to a subtask that no longer exists
	actionSkippedChecked = "skipped-checked"   // checked item ignored via --skip-checked
	actionWouldLink      = "would-link"        // dry run: would link to an existing subtask
	actionLinkedExisting = "linked-existing"   // linked to an existing subtask with the same title
	actionWouldCreate    = "would-create"      // dry run: would create a new subtask
	actionCreated        = "created"           // new subtask created
	actionWouldRelink    = "would-update-link" // dry run: link is in an outdated format
	actionRelinked       = "link-updated"      // outdated link replaced by the current format
)

// checklistResult describes what happens to one checklist item.
type checklistResult struct {
	ChecklistItem
	Action    string `json:"action"`
	SubtaskID int    `json:"subtask_id,omitempty"`
	URL       string `json:"url,omitempty"`
}

type checklistPlanOptions struct {
	SkipChecked     bool
	AllowDuplicates bool
	// When UpdateLinks is set, existing links that differ from
	// subtaskLink(BaseURL, TaskID, id) are planned for rewriting.
	UpdateLinks bool
	BaseURL     string
	TaskID      int
}

// planChecklistSync decides, without side effects, what to do with every
// checklist item given the task's current subtasks. Items that need a new
// subtask get actionWouldCreate, items that can be matched by title to an
// existing, not yet linked subtask get actionWouldLink.
func planChecklistSync(
	items []ChecklistItem,
	subtasks []api.Subtask,
	opts checklistPlanOptions,
) []checklistResult {
	exists := make(map[int]bool, len(subtasks))
	for _, s := range subtasks {
		if id, err := jsonNumberToInt(s.ID, "subtask ID"); err == nil {
			exists[id] = true
		}
	}

	// Subtasks already referenced from the description can't be claimed again.
	claimed := map[int]bool{}
	for _, item := range items {
		if item.LinkedSubtaskID != 0 {
			claimed[item.LinkedSubtaskID] = true
		}
	}

	// Unclaimed subtasks by normalized title, in position order.
	byTitle := map[string][]int{}
	if !opts.AllowDuplicates {
		for _, s := range subtasks {
			id, err := jsonNumberToInt(s.ID, "subtask ID")
			if err != nil || claimed[id] {
				continue
			}
			key := normalizeTitle(s.Title)
			byTitle[key] = append(byTitle[key], id)
		}
	}

	results := make([]checklistResult, 0, len(items))
	for _, item := range items {
		r := checklistResult{ChecklistItem: item}
		switch {
		case item.LinkedSubtaskID != 0 && exists[item.LinkedSubtaskID]:
			r.Action = actionLinked
			r.SubtaskID = item.LinkedSubtaskID
			if opts.UpdateLinks &&
				item.Link != subtaskLink(opts.BaseURL, opts.TaskID, item.LinkedSubtaskID) {
				r.Action = actionWouldRelink
			}
		case item.LinkedSubtaskID != 0:
			r.Action = actionMissing
			r.SubtaskID = item.LinkedSubtaskID
		case item.Checked && opts.SkipChecked:
			r.Action = actionSkippedChecked
		case len(byTitle[normalizeTitle(item.Title)]) > 0:
			key := normalizeTitle(item.Title)
			r.Action = actionWouldLink
			r.SubtaskID = byTitle[key][0]
			byTitle[key] = byTitle[key][1:]
		default:
			r.Action = actionWouldCreate
		}
		results = append(results, r)
	}
	return results
}

func newSubtaskFromChecklistCmd() *cobra.Command {
	var dryRun, skipChecked, allowDuplicates, removeFromDescription, noLinks bool
	var userID int

	cmd := &cobra.Command{
		Use:     "from-checklist <task-id>",
		Aliases: []string{"split", "sync"},
		Short:   "Create subtasks from the Markdown checkbox list in a task description",
		Long: `Detect Markdown checkbox items in the task description, such as

  - [ ] Write tests
  - [x] Update docs

and create one subtask per item. Unchecked items become "todo" subtasks,
checked items become "done" subtasks (use --skip-checked to ignore them).

After creating the subtasks, a link is appended to each checklist line in the
task description. Kanboard subtasks have no page of their own, so the link
opens the task page, where the subtask is listed; the link text carries the
subtask ID:

  - [ ] Write tests ([subtask #101](https://…/task/42))

These links mark items as converted. Run the command again after adding new
checkbox items and only the new (unlinked) items become subtasks. An unlinked
item whose title matches an existing, unlinked subtask is linked to that
subtask instead of creating a duplicate (disable with --allow-duplicates).
Items that link to a deleted subtask are reported as "missing" and left
unchanged. Links in an older format (pointing to the subtask edit form) are
rewritten to the current format. Checkboxes inside fenced code blocks are
ignored.

Use --dry-run to preview the result without changing anything, --no-links to
leave the description untouched, or --remove-from-description to delete the
converted lines from the description instead of linking them.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseID(args[0], "task ID")
			if err != nil {
				return err
			}
			if noLinks && removeFromDescription {
				return fmt.Errorf("--no-links and --remove-from-description cannot be combined")
			}
			baseURL, err := config.URL()
			if err != nil {
				return err
			}
			client := newClient()
			task, err := client.GetTask(taskID)
			if err != nil {
				return err
			}
			if task.ID.String() == "" {
				return fmt.Errorf("task %d not found", taskID)
			}

			items := parseChecklist(task.Description)
			if len(items) == 0 {
				if jsonOutput {
					printJSON(map[string]interface{}{
						"task_id": taskID, "dry_run": dryRun,
						"description_updated": false, "items": []checklistResult{},
					})
					return nil
				}
				fmt.Printf("No checkbox items found in the description of task %d.\n", taskID)
				return nil
			}

			subtasks, err := client.GetAllSubtasks(taskID)
			if err != nil {
				return err
			}
			results := planChecklistSync(items, subtasks, checklistPlanOptions{
				SkipChecked:     skipChecked,
				AllowDuplicates: allowDuplicates,
				UpdateLinks:     !noLinks && !removeFromDescription,
				BaseURL:         baseURL,
				TaskID:          taskID,
			})

			// Execute the plan.
			if !dryRun {
				for i := range results {
					r := &results[i]
					switch r.Action {
					case actionWouldLink:
						r.Action = actionLinkedExisting
					case actionWouldRelink:
						r.Action = actionRelinked
					case actionWouldCreate:
						status := api.SubtaskStatusTodo
						if r.Checked {
							status = api.SubtaskStatusDone
						}
						id, err := client.CreateSubtask(api.CreateSubtaskParams{
							TaskID: taskID,
							Title:  r.Title,
							UserID: userID,
							Status: status,
						})
						if err != nil {
							return fmt.Errorf("create subtask %q: %w", r.Title, err)
						}
						r.Action = actionCreated
						r.SubtaskID = id
					}
				}
			}
			for i := range results {
				if results[i].SubtaskID != 0 {
					results[i].URL = taskURL(baseURL, taskID)
				}
			}

			// Work out the new description.
			newDesc := task.Description
			switch {
			case removeFromDescription:
				var lines []int
				for _, r := range results {
					switch r.Action {
					case actionLinked, actionLinkedExisting, actionCreated,
						actionWouldLink, actionWouldCreate, actionWouldRelink, actionRelinked:
						lines = append(lines, r.Line)
					}
				}
				newDesc = removeLines(task.Description, lines)
			case !noLinks:
				links := map[int]string{}
				for _, r := range results {
					switch r.Action {
					case actionLinkedExisting, actionCreated, actionWouldRelink, actionRelinked:
						links[r.Line] = subtaskLink(baseURL, taskID, r.SubtaskID)
					}
				}
				newDesc = setLinksOnLines(task.Description, links)
			}

			descriptionUpdated := false
			if !dryRun && newDesc != task.Description {
				// Guard against overwriting edits made while we were working.
				current, err := client.GetTask(taskID)
				if err != nil {
					return fmt.Errorf("subtasks created, but re-reading the task failed: %w", err)
				}
				if current.Description != task.Description {
					return fmt.Errorf("subtasks created, but the task description was changed " +
						"concurrently; not updating it. Run the command again to add the links")
				}
				if err := client.UpdateTaskDescription(taskID, newDesc); err != nil {
					return fmt.Errorf(
						"subtasks created, but updating the description failed: %w",
						err,
					)
				}
				descriptionUpdated = true
			}

			if jsonOutput {
				out := map[string]interface{}{
					"task_id":             taskID,
					"dry_run":             dryRun,
					"description_updated": descriptionUpdated,
					"items":               results,
				}
				if dryRun && newDesc != task.Description {
					out["new_description"] = newDesc
				}
				printJSON(out)
				return nil
			}
			return printChecklistResults(taskID, results, dryRun, descriptionUpdated,
				removeFromDescription, newDesc != task.Description)
		},
	}
	cmd.Flags().
		BoolVarP(&dryRun, "dry-run", "n", false, "Only show what would happen, change nothing")
	cmd.Flags().
		BoolVar(&skipChecked, "skip-checked", false, "Ignore items that are already checked")
	cmd.Flags().BoolVar(&allowDuplicates, "allow-duplicates", false,
		"Always create new subtasks instead of linking unlinked items to existing subtasks with the same title")
	cmd.Flags().
		BoolVar(&noLinks, "no-links", false, "Do not add subtask links to the task description")
	cmd.Flags().BoolVar(&removeFromDescription, "remove-from-description", false,
		"Remove converted checklist lines from the task description instead of linking them")
	cmd.Flags().IntVarP(&userID, "user-id", "u", 0, "Assign created subtasks to this user ID")
	return cmd
}

func printChecklistResults(
	taskID int,
	results []checklistResult,
	dryRun, descriptionUpdated, remove, descriptionChanges bool,
) error {
	table := tablewriter.NewTable(os.Stdout)
	table.Header("Checked", "Title", "Action", "Subtask ID")
	counts := map[string]int{}
	for _, r := range results {
		checked := "[ ]"
		if r.Checked {
			checked = "[x]"
		}
		sid := ""
		if r.SubtaskID != 0 {
			sid = strconv.Itoa(r.SubtaskID)
		}
		counts[r.Action]++
		if err := table.Append(checked, r.Title, r.Action, sid); err != nil {
			return err
		}
	}
	if err := table.Render(); err != nil {
		return err
	}

	if dryRun {
		fmt.Printf(
			"Dry run for task %d: %d subtask(s) would be created, %d item(s) linked to existing subtasks.\n",
			taskID,
			counts[actionWouldCreate],
			counts[actionWouldLink],
		)
		if n := counts[actionWouldRelink]; n > 0 {
			fmt.Printf("%d outdated link(s) would be updated.\n", n)
		}
		if descriptionChanges {
			fmt.Println("The task description would be updated.")
		}
	} else {
		fmt.Printf("Task %d: created %d subtask(s), linked %d item(s) to existing subtasks.\n",
			taskID, counts[actionCreated], counts[actionLinkedExisting])
		if n := counts[actionRelinked]; n > 0 {
			fmt.Printf("Updated %d outdated link(s).\n", n)
		}
	}
	if n := counts[actionLinked]; n > 0 {
		fmt.Printf("%d item(s) were already linked to subtasks.\n", n)
	}
	if n := counts[actionMissing]; n > 0 {
		fmt.Fprintf(
			os.Stderr,
			"Warning: %d item(s) link to subtasks that no longer exist; they were left unchanged.\n",
			n,
		)
	}
	if descriptionUpdated {
		if remove {
			fmt.Println("Removed converted checklist lines from the task description.")
		} else {
			fmt.Println("Updated subtask links in the task description.")
		}
	}
	return nil
}

func normalizeTitle(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
