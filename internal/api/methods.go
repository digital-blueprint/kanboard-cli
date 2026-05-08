package api

import (
	"encoding/json"
	"fmt"
)

// Project represents a Kanboard project.
type Project struct {
	ID          json.Number `json:"id"`
	Name        string      `json:"name"`
	IsActive    json.Number `json:"is_active"`
	Description string      `json:"description"`
	Identifier  string      `json:"identifier"`
}

// Task represents a Kanboard task.
type Task struct {
	ID           json.Number       `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	ProjectID    json.Number       `json:"project_id"`
	ColumnID     json.Number       `json:"column_id"`
	SwimlaneID   json.Number       `json:"swimlane_id"`
	OwnerID      json.Number       `json:"owner_id"`
	IsActive     json.Number       `json:"is_active"`
	Position     json.Number       `json:"position"`
	ColorID      string            `json:"color_id"`
	DateDue      FlexibleTime      `json:"date_due"`
	DateStarted  FlexibleTime      `json:"date_started"`
	DateCreation FlexibleTime      `json:"date_creation"`
	DateMoved    FlexibleTime      `json:"date_moved"`
	Reference    string            `json:"reference"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// Comment represents a Kanboard comment.
type Comment struct {
	ID           json.Number `json:"id"`
	TaskID       json.Number `json:"task_id"`
	UserID       json.Number `json:"user_id"`
	DateCreation json.Number `json:"date_creation"`
	Comment      string      `json:"comment"`
	Username     string      `json:"username"`
	Name         string      `json:"name"`
}

// Column represents a board column.
type Column struct {
	ID       json.Number `json:"id"`
	Title    string      `json:"title"`
	Position json.Number `json:"position"`
}

// Me represents the current user.
type Me struct {
	ID       json.Number `json:"id"`
	Username string      `json:"username"`
	Name     string      `json:"name"`
	Email    string      `json:"email"`
}

// ---- Project methods --------------------------------------------------------

func (c *Client) GetAllProjects() ([]Project, error) {
	var result []Project
	if err := c.Call("getAllProjects", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetProjectByID(id int) (*Project, error) {
	var result Project
	if err := c.Call("getProjectById", map[string]int{"project_id": id}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateProject(name, description string) (int, error) {
	params := map[string]string{"name": name}
	if description != "" {
		params["description"] = description
	}
	var id int
	if err := c.Call("createProject", params, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) RemoveProject(id int) error {
	var ok bool
	if err := c.Call("removeProject", map[string]int{"project_id": id}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("removeProject")
	}
	return nil
}

// ---- Column methods ---------------------------------------------------------

func (c *Client) GetColumns(projectID int) ([]Column, error) {
	var result []Column
	if err := c.Call("getColumns", map[string]int{"project_id": projectID}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ---- Task methods -----------------------------------------------------------

func (c *Client) GetAllTasks(projectID, statusID int) ([]Task, error) {
	var result []Task
	params := map[string]int{"project_id": projectID, "status_id": statusID}
	if err := c.Call("getAllTasks", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetTask(taskID int) (*Task, error) {
	var result Task
	if err := c.Call("getTask", map[string]int{"task_id": taskID}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetTaskTags(taskID int) (map[string]string, error) {
	var result map[string]string
	if err := c.Call("getTaskTags", map[string]int{"task_id": taskID}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateTaskParams holds parameters for creating a task.
type CreateTaskParams struct {
	Title       string `json:"title"`
	ProjectID   int    `json:"project_id"`
	ColumnID    int    `json:"column_id,omitempty"`
	Description string `json:"description,omitempty"`
	ColorID     string `json:"color_id,omitempty"`
	DateDue     string `json:"date_due,omitempty"`
}

func (c *Client) CreateTask(p CreateTaskParams) (int, error) {
	var id int
	if err := c.Call("createTask", p, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) RemoveTask(taskID int) error {
	var ok bool
	if err := c.Call("removeTask", map[string]int{"task_id": taskID}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("removeTask")
	}
	return nil
}

// MoveTaskPositionParams holds parameters for moveTaskPosition.
type MoveTaskPositionParams struct {
	ProjectID  int `json:"project_id"`
	TaskID     int `json:"task_id"`
	ColumnID   int `json:"column_id"`
	Position   int `json:"position"`
	SwimlaneID int `json:"swimlane_id"`
}

func (c *Client) MoveTaskPosition(p MoveTaskPositionParams) error {
	var ok bool
	if err := c.Call("moveTaskPosition", p, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("moveTaskPosition")
	}
	return nil
}

func (c *Client) MoveTaskToProject(taskID, projectID int) error {
	var ok bool
	params := []int{taskID, projectID}
	if err := c.Call("moveTaskToProject", params, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("moveTaskToProject")
	}
	return nil
}

func (c *Client) CloseTask(taskID int) error {
	var ok bool
	if err := c.Call("closeTask", map[string]int{"task_id": taskID}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("closeTask")
	}
	return nil
}

func (c *Client) OpenTask(taskID int) error {
	var ok bool
	if err := c.Call("openTask", map[string]int{"task_id": taskID}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("openTask")
	}
	return nil
}

// ---- Comment methods --------------------------------------------------------

func (c *Client) GetAllComments(taskID int) ([]Comment, error) {
	var result []Comment
	if err := c.Call("getAllComments", map[string]int{"task_id": taskID}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateComment(taskID, userID int, content string) (int, error) {
	params := map[string]interface{}{
		"task_id": taskID,
		"user_id": userID,
		"content": content,
	}
	var id int
	if err := c.Call("createComment", params, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) RemoveComment(commentID int) error {
	var ok bool
	if err := c.Call("removeComment", map[string]int{"comment_id": commentID}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("removeComment")
	}
	return nil
}

// ---- Me methods -------------------------------------------------------------

func (c *Client) GetMe() (*Me, error) {
	var result Me
	if err := c.Call("getMe", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---- helpers ----------------------------------------------------------------

func errFailed(method string) error {
	return fmt.Errorf("%s returned false (operation failed)", method)
}
