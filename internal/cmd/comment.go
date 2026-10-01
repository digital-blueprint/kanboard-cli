package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
		newCommentEditCmd(),
		newCommentDeleteCmd(),
	)
	return cmd
}

func newCommentEditCmd() *cobra.Command {
	var file string
	var appendText string

	cmd := &cobra.Command{
		Use:     "edit <comment-id> [content]",
		Aliases: []string{"update"},
		Short:   "Change the text of a comment",
		Long: `Change the text of a comment.

The new text is taken from, in order of precedence: the [content] argument,
--file (use "-" for stdin), --append, piped stdin, or $VISUAL/$EDITOR
pre-filled with the current text.

Kanboard only lets you edit your own comments when using a personal token.`,
		Example: `  kanboard-cli comment edit 17 "Fixed typo"
  kanboard-cli comment edit 17 --file comment.md
  kanboard-cli comment edit 17 --append "Update: deployed."
  kanboard-cli comment edit 17                # opens $EDITOR`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "comment ID")
			if err != nil {
				return err
			}
			sources := 0
			for _, set := range []bool{len(args) == 2, cmd.Flags().Changed("file"), cmd.Flags().Changed("append")} {
				if set {
					sources++
				}
			}
			if sources > 1 {
				return fmt.Errorf("use only one of [content], --file, or --append")
			}

			client := newClient()
			comment, err := client.GetComment(id)
			if err != nil {
				return err
			}
			if comment == nil {
				return fmt.Errorf("comment %d not found", id)
			}

			var text string
			switch {
			case len(args) == 2:
				text = args[1]
			case cmd.Flags().Changed("file"):
				if text, err = readTextFile(file); err != nil {
					return err
				}
			case cmd.Flags().Changed("append"):
				text = appendDescription(comment.Comment, appendText)
			case !stdinIsTerminal():
				if text, err = readAll(os.Stdin); err != nil {
					return err
				}
			default:
				if text, err = editInEditor(comment.Comment, "kanboard-comment-*.md"); err != nil {
					return err
				}
			}
			if strings.TrimSpace(text) == "" {
				return fmt.Errorf(
					"comment text cannot be empty (use 'comment delete' to remove it)",
				)
			}

			changed := text != comment.Comment
			if changed {
				if err := client.UpdateComment(id, text); err != nil {
					return fmt.Errorf("%w (you can only edit your own comments)", err)
				}
			}
			if jsonOutput {
				printJSON(map[string]interface{}{
					"comment_id": id, "task_id": comment.TaskID, "updated": changed, "comment": text,
				})
				return nil
			}
			if changed {
				fmt.Printf("Comment %d updated\n", id)
			} else {
				fmt.Printf("Comment %d unchanged\n", id)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "F", "", `Read the text from a file ("-" for stdin)`)
	cmd.Flags().StringVar(&appendText, "append", "", "Append text to the comment")
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
				if err := table.Append(c.ID.String(), author, ts, c.Comment); err != nil {
					return err
				}
			}
			if err := table.Render(); err != nil {
				return err
			}
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
