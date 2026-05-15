package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage Kanboard projects",
	}
	cmd.AddCommand(
		newProjectListCmd(),
		newProjectCreateCmd(),
		newProjectDeleteCmd(),
	)
	return cmd
}

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			projects, err := client.GetAllProjects()
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(projects)
				return nil
			}

			if len(projects) == 0 {
				fmt.Println("No projects found.")
				return nil
			}
			table := tablewriter.NewTable(os.Stdout)
			table.Header("ID", "Name", "Active", "Identifier", "Description")
			for _, p := range projects {
				active := "yes"
				if p.IsActive.String() == "0" {
					active = "no"
				}
				desc := p.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				if err := table.Append(p.ID.String(), p.Name, active, p.Identifier, desc); err != nil {
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

func newProjectCreateCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			id, err := client.CreateProject(args[0], description)
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]int{"project_id": id})
				return nil
			}
			fmt.Printf("Project created with ID %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	return cmd
}

func newProjectDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid project ID: %s", args[0])
			}
			client := newClient()
			if err := client.RemoveProject(id); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]interface{}{"deleted": true, "project_id": id})
				return nil
			}
			fmt.Printf("Project %d deleted\n", id)
			return nil
		},
	}
}
