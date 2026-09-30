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
	Subtasks     []Subtask         `json:"subtasks,omitempty"`
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

// Subtask statuses as defined by Kanboard.
const (
	SubtaskStatusTodo       = 0
	SubtaskStatusInProgress = 1
	SubtaskStatusDone       = 2
)

// Subtask represents a Kanboard subtask. Username, Name and StatusName are
// only populated by getAllSubtasks.
type Subtask struct {
	ID            json.Number `json:"id"`
	Title         string      `json:"title"`
	Status        json.Number `json:"status"`
	TimeEstimated json.Number `json:"time_estimated"`
	TimeSpent     json.Number `json:"time_spent"`
	TaskID        json.Number `json:"task_id"`
	UserID        json.Number `json:"user_id"`
	Position      json.Number `json:"position"`
	Username      string      `json:"username,omitempty"`
	Name          string      `json:"name,omitempty"`
	StatusName    string      `json:"status_name,omitempty"`
}

// Column represents a board column.
type Column struct {
	ID       json.Number `json:"id"`
	Title    string      `json:"title"`
	Position json.Number `json:"position"`
}

// Swimlane represents a project swimlane.
type Swimlane struct {
	ID        json.Number `json:"id"`
	Name      string      `json:"name"`
	Position  json.Number `json:"position"`
	IsActive  json.Number `json:"is_active"`
	ProjectID json.Number `json:"project_id"`
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

func (c *Client) GetProjectByName(name string) (*Project, error) {
	var result Project
	if err := c.Call("getProjectByName", map[string]string{"name": name}, &result); err != nil {
		return nil, err
	}
	if result.ID.String() == "" {
		return nil, nil
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

// ---- Swimlane methods -------------------------------------------------------

func (c *Client) GetActiveSwimlanes(projectID int) ([]Swimlane, error) {
	var result []Swimlane
	if err := c.Call("getActiveSwimlanes", []int{projectID}, &result); err != nil {
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

func (c *Client) AssignTask(taskID, userID int) error {
	var ok bool
	params := map[string]int{"id": taskID, "owner_id": userID}
	if err := c.Call("updateTask", params, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("updateTask")
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

// ---- Subtask methods --------------------------------------------------------

func (c *Client) GetAllSubtasks(taskID int) ([]Subtask, error) {
	var result []Subtask
	if err := c.Call("getAllSubtasks", map[string]int{"task_id": taskID}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSubtask returns the subtask with the given ID, or nil if it does not exist.
func (c *Client) GetSubtask(subtaskID int) (*Subtask, error) {
	var result *Subtask
	if err := c.Call("getSubtask", map[string]int{"subtask_id": subtaskID}, &result); err != nil {
		return nil, err
	}
	if result == nil || result.ID.String() == "" {
		return nil, nil
	}
	return result, nil
}

// CreateSubtaskParams holds parameters for createSubtask.
type CreateSubtaskParams struct {
	TaskID        int     `json:"task_id"`
	Title         string  `json:"title"`
	UserID        int     `json:"user_id,omitempty"`
	TimeEstimated float64 `json:"time_estimated,omitempty"`
	TimeSpent     float64 `json:"time_spent,omitempty"`
	Status        int     `json:"status,omitempty"`
}

func (c *Client) CreateSubtask(p CreateSubtaskParams) (int, error) {
	return c.callForID("createSubtask", p)
}

// UpdateSubtaskParams holds parameters for updateSubtask. Nil fields are not
// sent and therefore left unchanged. Kanboard requires both ID and TaskID.
type UpdateSubtaskParams struct {
	ID            int      `json:"id"`
	TaskID        int      `json:"task_id"`
	Title         *string  `json:"title,omitempty"`
	UserID        *int     `json:"user_id,omitempty"`
	TimeEstimated *float64 `json:"time_estimated,omitempty"`
	TimeSpent     *float64 `json:"time_spent,omitempty"`
	Status        *int     `json:"status,omitempty"`
}

func (c *Client) UpdateSubtask(p UpdateSubtaskParams) error {
	var ok bool
	if err := c.Call("updateSubtask", p, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("updateSubtask")
	}
	return nil
}

func (c *Client) RemoveSubtask(subtaskID int) error {
	var ok bool
	if err := c.Call("removeSubtask", map[string]int{"subtask_id": subtaskID}, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("removeSubtask")
	}
	return nil
}

// UpdateTaskDescription replaces the description of a task.
func (c *Client) UpdateTaskDescription(taskID int, description string) error {
	var ok bool
	params := map[string]interface{}{"id": taskID, "description": description}
	if err := c.Call("updateTask", params, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("updateTask")
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

// callForID calls a procedure that returns a new record ID on success and
// false on failure.
func (c *Client) callForID(method string, params interface{}) (int, error) {
	var raw json.RawMessage
	if err := c.Call(method, params, &raw); err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil || id == 0 {
		return 0, errFailed(method)
	}
	return id, nil
}
