package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage task comments",
	}
	cmd.AddCommand(
		newCommentListCmd(),
		newCommentAddCmd(),
		newCommentDeleteCmd(),
	)
	return cmd
}

func newCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <task-id>",
		Short: "List all comments on a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()
			comments, err := client.GetAllComments(taskID)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(comments)
				return nil
			}

			if len(comments) == 0 {
				fmt.Println("No comments found.")
				return nil
			}
			table := tablewriter.NewTable(os.Stdout)
			table.Header("ID", "Author", "Date", "Comment")
			for _, c := range comments {
				ts := c.DateCreation.String()
				if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
					ts = time.Unix(n, 0).Format("2006-01-02 15:04")
				}
				author := c.Username
				if c.Name != "" {
					author = c.Name + " (" + c.Username + ")"
				}
				table.Append(c.ID.String(), author, ts, c.Comment)
			}
			table.Render()
			return nil
		},
	}
}

func newCommentAddCmd() *cobra.Command {
	var userID int

	cmd := &cobra.Command{
		Use:   "add <task-id> <content>",
		Short: "Add a comment to a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			client := newClient()

			// If user-id not provided, try to look it up via getMe (user API only).
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

			id, err := client.CreateComment(taskID, userID, args[1])
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]int{"comment_id": id, "task_id": taskID})
				return nil
			}
			fmt.Printf("Comment %d added to task %d\n", id, taskID)
			return nil
		},
	}
	cmd.Flags().IntVarP(&userID, "user-id", "u", 0, "User ID (auto-detected for user API)")
	return cmd
}

func newCommentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", args[0])
			}
			client := newClient()
			if err := client.RemoveComment(id); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{"deleted": true, "comment_id": id})
				return nil
			}
			fmt.Printf("Comment %d deleted\n", id)
			return nil
		},
	}
}
