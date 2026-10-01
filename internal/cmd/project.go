package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage Kanboard projects",
	}
	cmd.AddCommand(
		newProjectListCmd(),
		newProjectCreateCmd(),
		newProjectUpdateCmd(),
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

func newProjectUpdateCmd() *cobra.Command {
	var name, description, descriptionFile, identifier, email, owner string
	var startDate, endDate, status, public string
	var priorityDefault, priorityStart, priorityEnd int
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "update <project-id|name>",
		Aliases: []string{"edit"},
		Short:   "Change project settings",
		Long: `Change project settings. Only the given flags are changed.

--identifier, --email, and dates accept "none" (or "") to clear them; the
description is cleared with -d "". --owner accepts a user ID, name,
username, me, or none.

Requires the project manager role. Note: Kanboard's updateProject API always
turns off the "per-swimlane task limits" setting; re-enable it in the web UI
if you use it. --status and --public do not touch this setting.`,
		Example: `  kanboard-cli project update 12 --name "Software Solutions" --identifier SWS
  kanboard-cli project update "Software Solutions" -d "New description"
  kanboard-cli project update 12 --start-date 2026-10-01 --end-date none
  kanboard-cli project update 12 --priority-start 0 --priority-end 5 --priority-default 2
  kanboard-cli project update 12 --status inactive`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fl := cmd.Flags()
			changed := func(names ...string) bool {
				for _, n := range names {
					if fl.Changed(n) {
						return true
					}
				}
				return false
			}
			if !changed("name", "description", "description-file", "identifier", "email", "owner",
				"start-date", "end-date", "priority-default", "priority-start", "priority-end",
				"status", "public") {
				return fmt.Errorf("nothing to update: pass at least one flag (see --help)")
			}
			if changed("name") && strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name cannot be empty")
			}
			if changed("description-file") {
				text, err := readTextFile(descriptionFile)
				if err != nil {
					return err
				}
				description = text
			}
			var wantActive, wantPublic bool
			if changed("status") {
				switch strings.ToLower(strings.TrimSpace(status)) {
				case "active", "enabled", "enable", "open":
					wantActive = true
				case "inactive", "disabled", "disable", "closed":
					wantActive = false
				default:
					return fmt.Errorf("invalid --status %q: expected active or inactive", status)
				}
			}
			if changed("public") {
				var err error
				if wantPublic, err = parseBoolArg(public, "public"); err != nil {
					return err
				}
			}
			now := time.Now()
			for flag, value := range map[string]*string{"start-date": &startDate, "end-date": &endDate} {
				if changed(flag) {
					v, err := parseDateArg(*value, now)
					if err != nil {
						return fmt.Errorf("--%s: %w", flag, err)
					}
					*value = v
				}
			}

			client := newClient()
			l := newLookup(client)
			projectID, err := resolveProjectID(client, args[0])
			if err != nil {
				return err
			}
			cur, err := client.GetProjectByID(projectID)
			if err != nil {
				return err
			}
			if cur == nil {
				return fmt.Errorf("project %d not found", projectID)
			}

			p := api.UpdateProjectParams{ProjectID: projectID}
			var changes []fieldChange
			add := func(field, from, to string) {
				changes = append(changes, fieldChange{Field: field, From: from, To: to})
			}
			setString := func(flag, current, value string, target **string) {
				if changed(flag) && current != value {
					v := value
					*target = &v
					add(flag, emptyAsNone(current), emptyAsNone(value))
				}
			}
			setString("name", cur.Name, strings.TrimSpace(name), &p.Name)
			if changed("description", "description-file") && cur.Description != description {
				p.Description = &description
				add("description", cur.Description, description)
			}
			identifier = strings.ToUpper(strings.TrimSpace(identifier))
			if isNone(identifier) {
				identifier = ""
			}
			setString("identifier", cur.Identifier, identifier, &p.Identifier)
			if isNone(email) {
				email = ""
			}
			setString("email", cur.Email, strings.TrimSpace(email), &p.Email)
			setString("start-date", cur.StartDate, startDate, &p.StartDate)
			setString("end-date", cur.EndDate, endDate, &p.EndDate)
			if changed("owner") {
				id, err := l.resolveMember(projectID, owner)
				if err != nil {
					return err
				}
				if curOwner := numberToInt(cur.OwnerID); curOwner != id {
					p.OwnerID = &id
					add("owner", l.memberLabel(projectID, curOwner), l.memberLabel(projectID, id))
				}
			}

			start, end, def := numberToInt(cur.PriorityStart), numberToInt(cur.PriorityEnd),
				numberToInt(cur.PriorityDefault)
			setPriority := func(flag string, current, value int, target **int) {
				if changed(flag) && current != value {
					v := value
					*target = &v
					add(flag, strconv.Itoa(current), strconv.Itoa(value))
				}
			}
			setPriority("priority-start", start, priorityStart, &p.PriorityStart)
			setPriority("priority-end", end, priorityEnd, &p.PriorityEnd)
			setPriority("priority-default", def, priorityDefault, &p.PriorityDefault)
			if changed("priority-start", "priority-end", "priority-default") {
				if changed("priority-start") {
					start = priorityStart
				}
				if changed("priority-end") {
					end = priorityEnd
				}
				if changed("priority-default") {
					def = priorityDefault
				}
				if start > end {
					return fmt.Errorf(
						"priority start (%d) must not be greater than priority end (%d)",
						start,
						end,
					)
				}
				if def < start || def > end {
					return fmt.Errorf(
						"default priority %d is outside the range %d..%d",
						def,
						start,
						end,
					)
				}
			}

			isActive := cur.IsActive.String() != "0"
			activeChange := changed("status") && isActive != wantActive
			if activeChange {
				add("status", activeLabel(isActive), activeLabel(wantActive))
			}
			isPublic := cur.IsPublic.String() == "1"
			publicChange := changed("public") && isPublic != wantPublic
			if publicChange {
				add("public", yesNo(isPublic), yesNo(wantPublic))
			}

			updated := false
			if !dryRun {
				if !p.IsEmpty() {
					if err := client.UpdateProject(p); err != nil {
						return fmt.Errorf(
							"%w (requires the project manager role; identifier and email must be unique)",
							err,
						)
					}
					updated = true
				}
				if activeChange {
					if err := client.SetProjectActive(projectID, wantActive); err != nil {
						return err
					}
					updated = true
				}
				if publicChange {
					if err := client.SetProjectPublic(projectID, wantPublic); err != nil {
						return err
					}
					updated = true
				}
			}

			if changes == nil {
				changes = []fieldChange{}
			}
			if jsonOutput {
				printJSON(map[string]interface{}{
					"project_id": projectID, "updated": updated, "dry_run": dryRun, "changes": changes,
				})
				return nil
			}
			switch {
			case len(changes) == 0:
				fmt.Printf("Project %d unchanged\n", projectID)
				return nil
			case dryRun:
				fmt.Printf("Project %d would change (dry run):\n", projectID)
			default:
				fmt.Printf("Project %d updated:\n", projectID)
			}
			for _, c := range changes {
				from, to := c.From, c.To
				if c.Field == "description" {
					from, to = summarizeText(from), summarizeText(to)
				}
				fmt.Printf("  %-19s %s → %s\n", c.Field+":", from, to)
			}
			return nil
		},
	}
	fl := cmd.Flags()
	fl.StringVar(&name, "name", "", "Project name")
	fl.StringVarP(&description, "description", "d", "", "Description (Markdown)")
	fl.StringVarP(
		&descriptionFile,
		"description-file",
		"F",
		"",
		`Read the description from a file ("-" for stdin)`,
	)
	fl.StringVar(&identifier, "identifier", "", "Unique alphanumeric identifier, e.g. SWS")
	fl.StringVar(&email, "email", "", "Project email address (for creating tasks by email)")
	fl.StringVar(&owner, "owner", "", "Owner: user ID, name, username, me, or none")
	fl.StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD) or none")
	fl.StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD) or none")
	fl.IntVar(&priorityDefault, "priority-default", 0, "Default task priority")
	fl.IntVar(&priorityStart, "priority-start", 0, "Lowest task priority")
	fl.IntVar(&priorityEnd, "priority-end", 0, "Highest task priority")
	fl.StringVar(&status, "status", "", "Set status: active or inactive")
	fl.StringVar(&public, "public", "", "Public read-only access: yes or no")
	fl.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would change without saving")
	cmd.MarkFlagsMutuallyExclusive("description", "description-file")
	return cmd
}

func emptyAsNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func activeLabel(active bool) string {
	if active {
		return "active"
	}
	return "inactive"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
