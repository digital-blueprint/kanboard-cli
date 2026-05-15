package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage Kanboard tasks",
	}
	cmd.AddCommand(
		newTaskListCmd(),
		newTaskGetCmd(),
		newTaskCreateCmd(),
		newTaskDeleteCmd(),
		newTaskMoveCmd(),
		newTaskMoveProjectCmd(),
		newTaskMoveBoardCmd(),
		newTaskCloseCmd(),
		newTaskOpenCmd(),
		newTaskAssignCmd(),
	)
	return cmd
}

func newTaskListCmd() *cobra.Command {
	var projectID int
	var all bool
	var status, tag, column string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == 0 {
				return fmt.Errorf("--project-id is required")
			}
			if all {
				status = "all"
			}
			statusIDs, err := taskStatusIDs(status)
			if err != nil {
				return err
			}
			client := newClient()
			tasks, err := getTasksByStatuses(client, projectID, statusIDs)
			if err != nil {
				return err
			}

			if column != "" {
				columnID, err := resolveColumnID(client, projectID, column)
				if err != nil {
					return err
				}
				tasks = filterTasksByColumn(tasks, columnID)
			}

			if tag != "" {
				tasks, err = filterTasksByTag(client, tasks, tag)
				if err != nil {
					return err
				}
			}

			if jsonOutput {
				printJSON(tasks)
				return nil
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}
			table := tablewriter.NewTable(os.Stdout)
			table.Header("ID", "Title", "Column", "Position", "Color", "Due")
			for _, t := range tasks {
				title := t.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				if err := table.Append(
					t.ID.String(),
					title,
					t.ColumnID.String(),
					t.Position.String(),
					t.ColorID,
					t.DateDue.Format("2006-01-02"),
				); err != nil {
					return err
				}
			}
			if err := table.Render(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&projectID, "project-id", "p", 0, "Project ID (required)")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include closed/inactive tasks")
	cmd.Flags().StringVar(&status, "status", "open", "Task status: open, closed, or all")
	cmd.Flags().StringVar(&tag, "tag", "", "Only include tasks with this tag")
	cmd.Flags().
		StringVar(&column, "column", "", "Only include tasks in this column (ID or exact title)")
	return cmd
}

func taskStatusIDs(status string) ([]int, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "open":
		return []int{1}, nil
	case "closed":
		return []int{0}, nil
	case "all":
		return []int{1, 0}, nil
	default:
		return nil, fmt.Errorf("invalid --status %q: expected open, closed, or all", status)
	}
}

func getTasksByStatuses(client *api.Client, projectID int, statusIDs []int) ([]api.Task, error) {
	var tasks []api.Task
	for _, statusID := range statusIDs {
		statusTasks, err := client.GetAllTasks(projectID, statusID)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, statusTasks...)
	}
	return tasks, nil
}

func resolveColumnID(client *api.Client, projectID int, column string) (string, error) {
	column = strings.TrimSpace(column)
	if column == "" {
		return "", fmt.Errorf("--column cannot be empty")
	}
	if _, err := strconv.Atoi(column); err == nil {
		return column, nil
	}

	columns, err := client.GetColumns(projectID)
	if err != nil {
		return "", err
	}
	for _, candidate := range columns {
		if strings.EqualFold(candidate.Title, column) {
			return candidate.ID.String(), nil
		}
	}
	return "", fmt.Errorf("column %q not found in project %d", column, projectID)
}

func resolveColumnIDInt(client *api.Client, projectID int, column string) (int, error) {
	columnID, err := resolveColumnID(client, projectID, column)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(columnID)
	if err != nil {
		return 0, fmt.Errorf("invalid column ID %q: %w", columnID, err)
	}
	return id, nil
}

func resolveProjectID(client *api.Client, project string) (int, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return 0, fmt.Errorf("--project cannot be empty")
	}
	if id, err := strconv.Atoi(project); err == nil {
		return id, nil
	}

	matched, err := client.GetProjectByName(project)
	if err != nil {
		return 0, err
	}
	if matched == nil {
		return 0, fmt.Errorf("project %q not found", project)
	}
	return jsonNumberToInt(matched.ID, "project ID")
}

func resolveSwimlaneID(client *api.Client, projectID int, swimlane string) (int, error) {
	swimlane = strings.TrimSpace(swimlane)
	if swimlane == "" {
		return 0, fmt.Errorf("--swimlane cannot be empty")
	}
	if id, err := strconv.Atoi(swimlane); err == nil {
		return id, nil
	}

	swimlanes, err := client.GetActiveSwimlanes(projectID)
	if err != nil {
		return 0, err
	}
	for _, candidate := range swimlanes {
		if strings.EqualFold(candidate.Name, swimlane) {
			return jsonNumberToInt(candidate.ID, "swimlane ID")
		}
	}
	return 0, fmt.Errorf("swimlane %q not found in project %d", swimlane, projectID)
}

func firstColumnID(client *api.Client, projectID int) (int, error) {
	columns, err := client.GetColumns(projectID)
	if err != nil {
		return 0, err
	}
	if len(columns) == 0 {
		return 0, fmt.Errorf("project %d has no columns", projectID)
	}
	return jsonNumberToInt(columns[0].ID, "column ID")
}

func jsonNumberToInt(value fmt.Stringer, label string) (int, error) {
	id, err := strconv.Atoi(value.String())
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", label, value.String(), err)
	}
	return id, nil
}

func filterTasksByColumn(tasks []api.Task, columnID string) []api.Task {
	filtered := tasks[:0]
	for _, task := range tasks {
		if task.ColumnID.String() == columnID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func filterTasksByTag(client *api.Client, tasks []api.Task, tag string) ([]api.Task, error) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return tasks, nil
	}

	filtered := tasks[:0]
	for _, task := range tasks {
		taskID, err := strconv.Atoi(task.ID.String())
		if err != nil {
			return nil, fmt.Errorf("invalid task ID %q: %w", task.ID.String(), err)
		}
		tags, err := client.GetTaskTags(taskID)
		if err != nil {
			return nil, err
		}
		if taskHasTag(tags, tag) {
			task.Tags = tags
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

func taskHasTag(tags map[string]string, tag string) bool {
	for _, value := range tags {
		if strings.EqualFold(value, tag) {
			return true
		}
	}
	return false
}

func newTaskGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <task-id>",
		Short: "Show details of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()
			t, err := client.GetTask(id)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(t)
				return nil
			}

			active := "open"
			if t.IsActive.String() == "0" {
				active = "closed"
			}
			fmt.Printf("ID:          %s\n", t.ID)
			fmt.Printf("Title:       %s\n", t.Title)
			fmt.Printf("Status:      %s\n", active)
			fmt.Printf("Project:     %s\n", t.ProjectID)
			fmt.Printf("Column:      %s\n", t.ColumnID)
			fmt.Printf("Position:    %s\n", t.Position)
			fmt.Printf("Color:       %s\n", t.ColorID)
			fmt.Printf("Due:         %s\n", t.DateDue.Format("2006-01-02"))
			fmt.Printf("Reference:   %s\n", t.Reference)
			if t.Description != "" {
				fmt.Printf("Description:\n%s\n", t.Description)
			}
			return nil
		},
	}
}

func newTaskAssignCmd() *cobra.Command {
	var userID int

	cmd := &cobra.Command{
		Use:   "assign <task-id> [task-id...]",
		Short: "Assign one or more tasks to a user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskIDs := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid task ID: %s", arg)
				}
				taskIDs = append(taskIDs, id)
			}

			client := newClient()
			if userID == 0 {
				me, err := client.GetMe()
				if err != nil {
					return fmt.Errorf("could not determine user ID (use --user-id): %w", err)
				}
				uid, err := strconv.Atoi(me.ID.String())
				if err != nil {
					return fmt.Errorf("invalid user ID returned from API: %w", err)
				}
				userID = uid
			}

			assigned := make([]map[string]int, 0, len(taskIDs))
			for _, taskID := range taskIDs {
				if err := client.AssignTask(taskID, userID); err != nil {
					return fmt.Errorf("assign task %d: %w", taskID, err)
				}
				assigned = append(assigned, map[string]int{"task_id": taskID, "user_id": userID})
			}

			if jsonOutput {
				printJSON(assigned)
				return nil
			}
			for _, taskID := range taskIDs {
				fmt.Printf("Task %d assigned to user %d\n", taskID, userID)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&userID, "user-id", "u", 0, "User ID (auto-detected for user API)")
	return cmd
}

func newTaskCreateCmd() *cobra.Command {
	var projectID, columnID int
	var description, color, dateDue string

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == 0 {
				return fmt.Errorf("--project-id is required")
			}
			client := newClient()
			id, err := client.CreateTask(api.CreateTaskParams{
				Title:       args[0],
				ProjectID:   projectID,
				ColumnID:    columnID,
				Description: description,
				ColorID:     color,
				DateDue:     dateDue,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]int{"task_id": id})
				return nil
			}
			fmt.Printf("Task created with ID %d\n", id)
			return nil
		},
	}
	cmd.Flags().IntVarP(&projectID, "project-id", "p", 0, "Project ID (required)")
	cmd.Flags().IntVarP(&columnID, "column-id", "c", 0, "Column ID")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Task description (Markdown)")
	cmd.Flags().StringVar(&color, "color", "", "Color ID (e.g. blue, green, red, yellow)")
	cmd.Flags().StringVar(&dateDue, "due", "", "Due date (YYYY-MM-DD HH:MM)")
	return cmd
}

func newTaskDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()
			if err := client.RemoveTask(id); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{"deleted": true, "task_id": id})
				return nil
			}
			fmt.Printf("Task %d deleted\n", id)
			return nil
		},
	}
}

func newTaskMoveCmd() *cobra.Command {
	var projectID, columnID, position, swimlaneID int

	cmd := &cobra.Command{
		Use:   "move <task-id>",
		Short: "Move a task to a different column/position within the same project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			if projectID == 0 || columnID == 0 {
				return fmt.Errorf("--project-id and --column-id are required")
			}
			if position == 0 {
				position = 1
			}
			client := newClient()
			if err := client.MoveTaskPosition(api.MoveTaskPositionParams{
				ProjectID:  projectID,
				TaskID:     taskID,
				ColumnID:   columnID,
				Position:   position,
				SwimlaneID: swimlaneID,
			}); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{
					"moved":      true,
					"task_id":    taskID,
					"column_id":  columnID,
					"position":   position,
					"project_id": projectID,
				})
				return nil
			}
			fmt.Printf("Task %d moved to column %d, position %d\n", taskID, columnID, position)
			return nil
		},
	}
	cmd.Flags().IntVarP(&projectID, "project-id", "p", 0, "Project ID (required)")
	cmd.Flags().IntVarP(&columnID, "column-id", "c", 0, "Target column ID (required)")
	cmd.Flags().IntVar(&position, "position", 1, "Position in column")
	cmd.Flags().IntVar(&swimlaneID, "swimlane-id", 0, "Swimlane ID (0 = default)")
	return cmd
}

func newTaskMoveProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move-project <task-id> <target-project-id>",
		Short: "Move a task to a different project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			projectID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid project ID: %s", args[1])
			}
			client := newClient()
			if err := client.MoveTaskToProject(taskID, projectID); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{
					"moved":      true,
					"task_id":    taskID,
					"project_id": projectID,
				})
				return nil
			}
			fmt.Printf("Task %d moved to project %d\n", taskID, projectID)
			return nil
		},
	}
}

func newTaskMoveBoardCmd() *cobra.Command {
	var project, column, swimlane string
	var position int

	cmd := &cobra.Command{
		Use:   "move-board <task-id> [task-id...]",
		Short: "Move one or more tasks to a project board, column, or swimlane",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(project) == "" {
				return fmt.Errorf("--project is required")
			}
			if position == 0 {
				position = 1
			}

			taskIDs := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid task ID: %s", arg)
				}
				taskIDs = append(taskIDs, id)
			}

			client := newClient()
			projectID, err := resolveProjectID(client, project)
			if err != nil {
				return err
			}

			columnID := 0
			if strings.TrimSpace(column) != "" {
				columnID, err = resolveColumnIDInt(client, projectID, column)
				if err != nil {
					return err
				}
			}

			swimlaneID := 0
			if strings.TrimSpace(swimlane) != "" {
				swimlaneID, err = resolveSwimlaneID(client, projectID, swimlane)
				if err != nil {
					return err
				}
			}

			moved := make([]map[string]int, 0, len(taskIDs))
			for _, taskID := range taskIDs {
				if err := client.MoveTaskToProject(taskID, projectID); err != nil {
					return fmt.Errorf("move task %d to project %d: %w", taskID, projectID, err)
				}

				targetColumnID := columnID
				if targetColumnID == 0 && swimlaneID != 0 {
					task, err := client.GetTask(taskID)
					if err != nil {
						return fmt.Errorf("get task %d after project move: %w", taskID, err)
					}
					targetColumnID, err = jsonNumberToInt(task.ColumnID, "column ID")
					if err != nil || targetColumnID == 0 {
						targetColumnID, err = firstColumnID(client, projectID)
						if err != nil {
							return err
						}
					}
				}

				if targetColumnID != 0 {
					if err := client.MoveTaskPosition(api.MoveTaskPositionParams{
						ProjectID:  projectID,
						TaskID:     taskID,
						ColumnID:   targetColumnID,
						Position:   position,
						SwimlaneID: swimlaneID,
					}); err != nil {
						return fmt.Errorf("move task %d on board %d: %w", taskID, projectID, err)
					}
				}

				moved = append(moved, map[string]int{
					"task_id":     taskID,
					"project_id":  projectID,
					"column_id":   targetColumnID,
					"swimlane_id": swimlaneID,
					"position":    position,
				})
			}

			if jsonOutput {
				printJSON(moved)
				return nil
			}
			for _, taskID := range taskIDs {
				fmt.Printf("Task %d moved to project %d", taskID, projectID)
				if columnID != 0 {
					fmt.Printf(", column %d", columnID)
				}
				if swimlaneID != 0 {
					fmt.Printf(", swimlane %d", swimlaneID)
				}
				fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().
		StringVarP(&project, "project", "p", "", "Target project ID or exact name (required)")
	cmd.Flags().StringVarP(&column, "column", "c", "", "Target column ID or exact title")
	cmd.Flags().StringVar(&swimlane, "swimlane", "", "Target swimlane ID or exact name")
	cmd.Flags().IntVar(&position, "position", 1, "Position in column")
	return cmd
}

func newTaskCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <task-id> [task-id...]",
		Short: "Close (complete) one or more tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskIDs := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid task ID: %s", arg)
				}
				taskIDs = append(taskIDs, id)
			}

			client := newClient()
			closed := make([]map[string]interface{}, 0, len(taskIDs))
			for _, taskID := range taskIDs {
				if err := client.CloseTask(taskID); err != nil {
					return fmt.Errorf("close task %d: %w", taskID, err)
				}
				closed = append(closed, map[string]interface{}{"closed": true, "task_id": taskID})
			}

			if jsonOutput {
				printJSON(closed)
				return nil
			}
			for _, taskID := range taskIDs {
				fmt.Printf("Task %d closed\n", taskID)
			}
			return nil
		},
	}
}

func newTaskOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <task-id>",
		Short: "Re-open a closed task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()
			if err := client.OpenTask(id); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{"opened": true, "task_id": id})
				return nil
			}
			fmt.Printf("Task %d re-opened\n", id)
			return nil
		},
	}
}
