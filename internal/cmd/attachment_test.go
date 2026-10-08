package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func attachmentFixture(id, name string) map[string]interface{} {
	return map[string]interface{}{
		"id": id, "task_id": "42", "name": name, "path": "tasks/42/storage-key",
		"size": "5", "is_image": "0", "date": "1720000000", "user_id": "7",
		"username": "alice", "user_name": "Alice",
	}
}

func TestAttachmentList(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getAllTaskFiles", []interface{}{attachmentFixture("17", "notes.txt")})
	out, err := runCLI(t, "attachment", "list", "42")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"17", "notes.txt", "Alice", "BYTES"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	out, err = runCLI(t, "--json", "attachment", "list", "42")
	if err != nil {
		t.Fatal(err)
	}
	var files []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &files); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0]["name"] != "notes.txt" {
		t.Fatalf("unexpected metadata: %s", out)
	}
	if files[0]["path"] != "tasks/42/storage-key" {
		t.Errorf("storage path missing: %s", out)
	}
	if got := f.calls[0].Params["task_id"]; got != float64(42) {
		t.Errorf("task_id = %v", got)
	}
}

func TestAttachmentEmptyList(t *testing.T) {
	for _, empty := range []interface{}{nil, false, []interface{}{}, map[string]interface{}{}} {
		t.Run("empty", func(t *testing.T) {
			f := newFakeKanboard(t)
			f.on("getAllTaskFiles", empty)
			out, err := runCLI(t, "--json", "attachment", "list", "42")
			if err != nil || strings.TrimSpace(out) != "[]" {
				t.Fatalf("out=%q err=%v", out, err)
			}
			out, err = runCLI(t, "attachment", "list", "42")
			if err != nil || !strings.Contains(out, "No attachments") {
				t.Fatalf("out=%q err=%v", out, err)
			}
		})
	}
}

func TestAttachmentGet(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTaskFile", attachmentFixture("17", "notes.txt"))
	out, err := runCLI(t, "--json", "attachment", "get", "17")
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]interface{}
	if err := json.Unmarshal([]byte(out), &file); err != nil {
		t.Fatal(err)
	}
	if file["name"] != "notes.txt" {
		t.Errorf("unexpected output: %s", out)
	}
	f.on("getTaskFile", false)
	_, err = runCLI(t, "attachment", "get", "17")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestAttachmentDownloadBinaryAndJSON(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTaskFile", attachmentFixture("17", "image.png"))
	data := []byte{0, 1, 255, '\n', 42}
	f.on("downloadTaskFile", base64.StdEncoding.EncodeToString(data))
	path := filepath.Join(t.TempDir(), "image.png")
	out, err := runCLI(t, "--json", "attachment", "download", "17", "-o", path)
	if err != nil {
		t.Fatal(err)
	}
	var result attachmentDownload
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.LocalPath != path || result.Bytes != len(data) || result.FileID.String() != "17" {
		t.Fatalf("unexpected manifest: %s", out)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(data) {
		t.Fatalf("got=%v err=%v", got, err)
	}
	out, err = runCLI(t, "attachment", "download", "17", "-o", "-")
	if err != nil || out != string(data) {
		t.Fatalf("raw stdout=%q err=%v", out, err)
	}
	if got := f.calls[len(f.calls)-1].Params["file_id"]; got != float64(17) {
		t.Errorf("file_id=%v", got)
	}
	_, err = runCLI(t, "--json", "attachment", "download", "17", "-o", "-")
	if err == nil {
		t.Fatal("expected error combining JSON and raw stdout")
	}
}

func TestAttachmentDownloadNoOverwrite(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTaskFile", attachmentFixture("17", "notes.txt"))
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := runCLI(t, "attachment", "download", "17", "-o", path)
	if err == nil {
		t.Fatal("expected overwrite refusal")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("existing file modified: %q, %v", got, err)
	}
	for _, call := range f.calls {
		if call.Method == "downloadTaskFile" {
			t.Fatal("should refuse before fetching content")
		}
	}
}

func TestAttachmentDownloadRemovesFailedFile(t *testing.T) {
	for _, content := range []interface{}{"", "not base64!", false} {
		t.Run("failure", func(t *testing.T) {
			f := newFakeKanboard(t)
			f.on("getTaskFile", attachmentFixture("17", "notes.txt"))
			f.on("downloadTaskFile", content)
			path := filepath.Join(t.TempDir(), "notes.txt")
			_, err := runCLI(t, "attachment", "download", "17", "-o", path)
			if err == nil {
				t.Fatal("expected download error")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed download left file: %v", err)
			}
		})
	}
}

func TestAttachmentDownloadRefusesSymlink(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTaskFile", attachmentFixture("17", "notes.txt"))
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := runCLI(t, "attachment", "download", "17", "-o", link); err == nil {
		t.Fatal("expected symlink refusal")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "original" {
		t.Fatalf("symlink target changed: %q, %v", got, err)
	}
}

func TestAttachmentDefaultDestinations(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getTaskFile", attachmentFixture("17", "../../notes.txt"))
	f.on("getAllTaskFiles", []interface{}{attachmentFixture("17", "notes.txt")})
	f.on("downloadTaskFile", base64.StdEncoding.EncodeToString([]byte("hello")))
	dir := t.TempDir()
	t.Chdir(dir)
	out, err := runCLI(t, "attachment", "download", "17")
	if err != nil || !strings.Contains(out, "saved to") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, "attachment", "download-all", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "task-42-attachments", "17-notes.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestAttachmentDownloadAll(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getAllTaskFiles", []interface{}{
		attachmentFixture("17", "../../notes.txt"), attachmentFixture("18", `C:\dir\notes.txt`),
	})
	f.on("downloadTaskFile", base64.StdEncoding.EncodeToString([]byte("hello")))
	dir := filepath.Join(t.TempDir(), "attachments")
	out, err := runCLI(t, "--json", "attachment", "download-all", "42", "--output-dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	var results []attachmentDownload
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("unexpected results: %s", out)
	}
	for i, name := range []string{"17-notes.txt", "18-notes.txt"} {
		path := filepath.Join(dir, name)
		if results[i].LocalPath != path {
			t.Errorf("path=%s want=%s", results[i].LocalPath, path)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != "hello" {
			t.Fatalf("file %s=%q err=%v", name, got, err)
		}
	}
	// A rerun must report failures as valid JSON and leave files alone.
	out, err = runCLI(t, "--json", "attachment", "download-all", "42", "--output-dir", dir)
	if err == nil {
		t.Fatal("expected failure for existing files")
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if results[0].Error == "" || results[1].Error == "" {
		t.Errorf("missing errors: %s", out)
	}
}

func TestAttachmentDownloadAllPartialFailure(t *testing.T) {
	f := newFakeKanboard(t)
	f.on(
		"getAllTaskFiles",
		[]interface{}{attachmentFixture("17", "a.txt"), attachmentFixture("18", "b.txt")},
	)
	f.handlers["downloadTaskFile"] = func(params map[string]interface{}) interface{} {
		if params["file_id"] == float64(17) {
			return ""
		}
		return base64.StdEncoding.EncodeToString([]byte("hello"))
	}
	dir := t.TempDir()
	out, err := runCLI(t, "--json", "attachment", "download-all", "42", "--output-dir", dir)
	if err == nil {
		t.Fatal("expected partial failure")
	}
	var results []attachmentDownload
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Error == "" || results[1].LocalPath == "" {
		t.Fatalf("unexpected partial manifest: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "17-a.txt")); !os.IsNotExist(err) {
		t.Errorf("failed file left behind: %v", err)
	}
}

func TestAttachmentDownloadAllEmpty(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getAllTaskFiles", []interface{}{})
	dir := filepath.Join(t.TempDir(), "unused")
	out, err := runCLI(t, "--json", "attachment", "download-all", "42", "--output-dir", dir)
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("empty list created directory: %v", err)
	}
}

func TestAttachmentInvalidIDs(t *testing.T) {
	for _, command := range []string{"list", "get", "download", "download-all"} {
		for _, id := range []string{"0", "-1", "abc"} {
			_, err := runCLI(t, "attachment", command, id)
			if err == nil {
				t.Errorf("%s accepted invalid ID %s", command, id)
			}
		}
	}
}

func TestSafeAttachmentName(t *testing.T) {
	for name, want := range map[string]string{
		"../../report.pdf": "report.pdf", `C:\dir\report.pdf`: "report.pdf",
		"/tmp/report.pdf": "report.pdf", "..": "attachment", "": "attachment",
		"a\nb?.txt": "a_b_.txt", "report.pdf": "report.pdf",
		"CON.txt": "_CON.txt", "nul": "_nul",
	} {
		if got := safeAttachmentName(name); got != want {
			t.Errorf("safeAttachmentName(%q)=%q want=%q", name, got, want)
		}
	}
}

func TestTaskGetIncludesAttachments(t *testing.T) {
	f := newFakeKanboard(t)
	f.on("getAllSubtasks", []interface{}{})
	f.on("getAllTaskFiles", []interface{}{attachmentFixture("17", "report.pdf")})
	out, err := runCLI(t, "--json", "task", "get", "42")
	if err != nil {
		t.Fatal(err)
	}
	var task struct {
		Attachments []map[string]interface{} `json:"attachments"`
	}
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatal(err)
	}
	if len(task.Attachments) != 1 || task.Attachments[0]["name"] != "report.pdf" {
		t.Errorf("attachments missing: %s", out)
	}
	out, err = runCLI(t, "task", "get", "42")
	if err != nil || !strings.Contains(out, "report.pdf") ||
		!strings.Contains(out, "attachment download") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	delete(f.handlers, "getAllTaskFiles")
	if _, err := runCLI(t, "--json", "task", "get", "42"); err != nil {
		t.Errorf("informational fetch failed task get: %v", err)
	}
}
