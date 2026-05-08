package cmd

import (
	"fmt"
	"os"
	"strconv"

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
		newTaskCloseCmd(),
		newTaskOpenCmd(),
	)
	return cmd
}

func newTaskListCmd() *cobra.Command {
	var projectID int
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == 0 {
				return fmt.Errorf("--project-id is required")
			}
			statusID := 1
			if all {
				statusID = 0
			}
			client := newClient()
			tasks, err := client.GetAllTasks(projectID, statusID)
			if err != nil {
				return err
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
				table.Append(t.ID.String(), title, t.ColumnID.String(), t.Position.String(), t.ColorID,
					t.DateDue.Format("2006-01-02"))
			}
			table.Render()
			return nil
		},
	}
	cmd.Flags().IntVarP(&projectID, "project-id", "p", 0, "Project ID (required)")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include closed/inactive tasks")
	return cmd
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

func newTaskCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <task-id>",
		Short: "Close (complete) a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()
			if err := client.CloseTask(id); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{"closed": true, "task_id": id})
				return nil
			}
			fmt.Printf("Task %d closed\n", id)
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
