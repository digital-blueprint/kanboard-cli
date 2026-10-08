package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tu-graz/kanboard-cli/internal/api"
)

func newAttachmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachment",
		Aliases: []string{"attachments"},
		Short:   "List and download task attachments",
	}
	cmd.AddCommand(newAttachmentListCmd(), newAttachmentGetCmd(),
		newAttachmentDownloadCmd(), newAttachmentDownloadAllCmd())
	return cmd
}

func newAttachmentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <task-id>",
		Short: "List attachment metadata for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "task ID")
			if err != nil {
				return err
			}
			files, err := newClient().GetAllTaskFiles(id)
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(files)
				return nil
			}
			return renderAttachmentTable(files)
		},
	}
}

func newAttachmentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <file-id>",
		Short: "Show attachment metadata (not its content)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "file ID")
			if err != nil {
				return err
			}
			file, err := requireAttachment(newClient(), id)
			if err != nil {
				return err
			}
			if jsonOutput {
				printJSON(file)
				return nil
			}
			return renderAttachmentTable([]api.TaskFile{*file})
		},
	}
}

func requireAttachment(client *api.Client, id int) (*api.TaskFile, error) {
	file, err := client.GetTaskFile(id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("attachment %d not found", id)
	}
	return file, nil
}

func renderAttachmentTable(files []api.TaskFile) error {
	if len(files) == 0 {
		fmt.Println("No attachments found.")
		return nil
	}
	table := tablewriter.NewTable(os.Stdout)
	table.Header("ID", "Task", "Name", "Bytes", "Date", "Author")
	for _, file := range files {
		author := file.Username
		if file.UserName != "" {
			author = file.UserName
		}
		if author == "" {
			author = file.UserID.String()
		}
		if err := table.Append(file.ID.String(), file.TaskID.String(), file.Name,
			file.Size.String(), formatTaskTime(file.Date), author); err != nil {
			return err
		}
	}
	return table.Render()
}

type attachmentDownload struct {
	FileID    json.Number `json:"file_id"`
	TaskID    json.Number `json:"task_id"`
	Name      string      `json:"name"`
	LocalPath string      `json:"local_path,omitempty"`
	Bytes     int         `json:"bytes,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func newAttachmentDownloadCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "download <file-id>",
		Short: "Download an attachment to a file (or stdout with -o -)",
		Long: `Download the original attachment bytes. By default, save a safe version of
the original filename in the current directory. Existing files are never
overwritten. Use --output - for raw stdout (cannot be combined with --json).
With --json, return metadata and the absolute local_path, not binary content.`,
		Example: `  kanboard-cli attachment download 17
  kanboard-cli attachment download 17 -o report.pdf
  kanboard-cli --json attachment download 17 -o /tmp/report.pdf
  kanboard-cli attachment download 17 -o - | pdftotext - -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "file ID")
			if err != nil {
				return err
			}
			if output == "-" && jsonOutput {
				return fmt.Errorf("--json cannot be combined with --output - (raw bytes)")
			}
			if cmd.Flags().Changed("output") && output == "" {
				return fmt.Errorf("--output cannot be empty")
			}
			client := newClient()
			file, err := requireAttachment(client, id)
			if err != nil {
				return err
			}
			if output == "-" {
				data, err := client.DownloadTaskFile(id)
				if err != nil {
					return err
				}
				_, err = os.Stdout.Write(data)
				return err
			}
			if output == "" {
				output = safeAttachmentName(file.Name)
			}
			result, err := saveAttachment(client, *file, output)
			if err != nil {
				return err
			}
			printAttachmentDownloads([]attachmentDownload{result}, false)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", `Destination filename ("-" for raw stdout)`)
	return cmd
}

func newAttachmentDownloadAllCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "download-all <task-id>",
		Short: "Download every attachment of a task",
		Long: `Download all attachments into --output-dir (default: task-<id>-attachments).
Names are prefixed with the file ID to avoid duplicate-name collisions.
Existing files are never overwritten. All files are attempted; failures are
reported alongside successes and the command exits nonzero if any failed.
With --json, return an array with file_id, task_id, name, local_path and bytes
for successful downloads, or error for failed downloads.`,
		Example: `  kanboard-cli attachment download-all 42
  kanboard-cli --json attachment download-all 42 --output-dir ./attachments`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "task ID")
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("output-dir") && dir == "" {
				return fmt.Errorf("--output-dir cannot be empty")
			}
			client := newClient()
			files, err := client.GetAllTaskFiles(id)
			if err != nil {
				return err
			}
			if dir == "" {
				dir = fmt.Sprintf("task-%d-attachments", id)
			}
			if len(files) > 0 {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
			}
			results := make([]attachmentDownload, 0, len(files))
			failed := 0
			for _, file := range files {
				fileID, err := parseID(file.ID.String(), "attachment ID returned by API")
				var result attachmentDownload
				if err == nil {
					name := fmt.Sprintf("%d-%s", fileID, safeAttachmentName(file.Name))
					result, err = saveAttachment(client, file, filepath.Join(dir, name))
				}
				if err != nil {
					failed++
					result = attachmentDownload{
						FileID: file.ID,
						TaskID: file.TaskID,
						Name:   file.Name,
						Error:  err.Error(),
					}
				}
				results = append(results, result)
			}
			printAttachmentDownloads(results, true)
			if failed > 0 {
				return fmt.Errorf("%d attachment download(s) failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "output-dir", "", "Directory to save attachments into")
	return cmd
}

// Treat both Unix and Windows paths as untrusted, and keep generated names
// portable. Explicit --output paths are chosen by the caller, not the server.
func safeAttachmentName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, " .")
	if name == "" {
		return "attachment"
	}
	// Leave room for the ID prefix and avoid splitting a UTF-8 sequence.
	if len(name) > 200 {
		name = strings.TrimRight(strings.ToValidUTF8(name[:200], ""), " .")
	}
	stem, _, _ := strings.Cut(strings.ToUpper(name), ".")
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5",
		"COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5",
		"LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}

func saveAttachment(
	client *api.Client,
	file api.TaskFile,
	destination string,
) (attachmentDownload, error) {
	result := attachmentDownload{FileID: file.ID, TaskID: file.TaskID, Name: file.Name}
	id, err := parseID(file.ID.String(), "attachment ID returned by API")
	if err != nil {
		return result, err
	}
	path, err := filepath.Abs(destination)
	if err != nil {
		return result, err
	}
	// O_EXCL also refuses symlinks, preventing accidental writes outside the
	// destination through an existing link. Use private permissions for task data.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return result, fmt.Errorf("create %s (existing files are not overwritten): %w", path, err)
	}
	complete := false
	defer func() {
		_ = f.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	data, err := client.DownloadTaskFile(id)
	if err != nil {
		return result, err
	}
	if _, err := f.Write(data); err != nil {
		return result, fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return result, fmt.Errorf("close %s: %w", path, err)
	}
	complete = true
	result.LocalPath, result.Bytes = path, len(data)
	return result, nil
}

func printAttachmentDownloads(results []attachmentDownload, all bool) {
	if jsonOutput {
		if all {
			printJSON(results)
		} else {
			printJSON(results[0])
		}
		return
	}
	if len(results) == 0 {
		fmt.Println("No attachments found.")
	}
	for _, result := range results {
		if result.Error != "" {
			fmt.Fprintf(
				os.Stderr,
				"Attachment %s (%s): %s\n",
				result.FileID,
				result.Name,
				result.Error,
			)
		} else {
			fmt.Printf("Attachment %s saved to %s (%d bytes)\n", result.FileID, result.LocalPath, result.Bytes)
		}
	}
}
