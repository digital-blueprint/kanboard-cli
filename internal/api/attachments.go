package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// TaskFile is attachment metadata. Path is Kanboard's storage key, not a URL
// or local filename. The original filename is Name.
type TaskFile struct {
	ID       json.Number  `json:"id"`
	TaskID   json.Number  `json:"task_id"`
	Name     string       `json:"name"`
	Path     string       `json:"path"`
	Size     json.Number  `json:"size"`
	IsImage  json.Number  `json:"is_image"`
	Date     FlexibleTime `json:"date"`
	UserID   json.Number  `json:"user_id"`
	Username string       `json:"username,omitempty"`
	UserName string       `json:"user_name,omitempty"`
	ETag     string       `json:"etag,omitempty"`
}

func (c *Client) GetAllTaskFiles(taskID int) ([]TaskFile, error) {
	var raw json.RawMessage
	if err := c.Call("getAllTaskFiles", map[string]int{"task_id": taskID}, &raw); err != nil {
		return nil, err
	}
	if isEmptyResult(raw) {
		return []TaskFile{}, nil
	}
	var files []TaskFile
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, fmt.Errorf("decode task files: %w", err)
	}
	return files, nil
}

// GetTaskFile returns nil if the attachment does not exist.
func (c *Client) GetTaskFile(fileID int) (*TaskFile, error) {
	var file TaskFile
	found, err := c.callOptional("getTaskFile", map[string]int{"file_id": fileID}, &file)
	if err != nil || !found || file.ID.String() == "" {
		return nil, err
	}
	return &file, nil
}

// DownloadTaskFile decodes Kanboard's base64 response into the original bytes.
// Kanboard returns an empty string when the stored file cannot be retrieved.
func (c *Client) DownloadTaskFile(fileID int) ([]byte, error) {
	var encoded string
	if err := c.Call("downloadTaskFile", map[string]int{"file_id": fileID}, &encoded); err != nil {
		return nil, err
	}
	if encoded == "" {
		return nil, fmt.Errorf("attachment %d is missing or its content is unavailable", fileID)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode attachment %d: %w", fileID, err)
	}
	return data, nil
}
