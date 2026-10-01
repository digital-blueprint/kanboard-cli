package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Project represents a Kanboard project.
type Project struct {
	ID              json.Number `json:"id"`
	Name            string      `json:"name"`
	IsActive        json.Number `json:"is_active"`
	IsPublic        json.Number `json:"is_public,omitempty"`
	Description     string      `json:"description"`
	Identifier      string      `json:"identifier"`
	OwnerID         json.Number `json:"owner_id,omitempty"`
	Email           string      `json:"email,omitempty"`
	StartDate       string      `json:"start_date,omitempty"`
	EndDate         string      `json:"end_date,omitempty"`
	PriorityDefault json.Number `json:"priority_default,omitempty"`
	PriorityStart   json.Number `json:"priority_start,omitempty"`
	PriorityEnd     json.Number `json:"priority_end,omitempty"`
}

// Task represents a Kanboard task.
type Task struct {
	ID                  json.Number       `json:"id"`
	Title               string            `json:"title"`
	Description         string            `json:"description"`
	ProjectID           json.Number       `json:"project_id"`
	ColumnID            json.Number       `json:"column_id"`
	SwimlaneID          json.Number       `json:"swimlane_id"`
	OwnerID             json.Number       `json:"owner_id"`
	CreatorID           json.Number       `json:"creator_id,omitempty"`
	CategoryID          json.Number       `json:"category_id"`
	IsActive            json.Number       `json:"is_active"`
	Position            json.Number       `json:"position"`
	ColorID             string            `json:"color_id"`
	Priority            json.Number       `json:"priority"`
	Score               json.Number       `json:"score"`
	TimeEstimated       json.Number       `json:"time_estimated"`
	TimeSpent           json.Number       `json:"time_spent"`
	DateDue             FlexibleTime      `json:"date_due"`
	DateStarted         FlexibleTime      `json:"date_started"`
	DateCreation        FlexibleTime      `json:"date_creation"`
	DateModification    FlexibleTime      `json:"date_modification"`
	DateCompleted       FlexibleTime      `json:"date_completed"`
	DateMoved           FlexibleTime      `json:"date_moved"`
	Reference           string            `json:"reference"`
	RecurrenceStatus    json.Number       `json:"recurrence_status"`
	RecurrenceTrigger   json.Number       `json:"recurrence_trigger"`
	RecurrenceFactor    json.Number       `json:"recurrence_factor"`
	RecurrenceTimeframe json.Number       `json:"recurrence_timeframe"`
	RecurrenceBasedate  json.Number       `json:"recurrence_basedate"`
	URL                 string            `json:"url,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	Subtasks            []Subtask         `json:"subtasks,omitempty"`
}

// Category represents a project category.
type Category struct {
	ID        json.Number `json:"id"`
	Name      string      `json:"name"`
	ProjectID json.Number `json:"project_id"`
	ColorID   string      `json:"color_id,omitempty"`
}

// User represents a Kanboard user as returned by getUserByName.
type User struct {
	ID       json.Number `json:"id"`
	Username string      `json:"username"`
	Name     string      `json:"name"`
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

// GetProjectByID returns the project with the given ID, or nil if it does not
// exist.
func (c *Client) GetProjectByID(id int) (*Project, error) {
	var result Project
	found, err := c.callOptional("getProjectById", map[string]int{"project_id": id}, &result)
	if err != nil || !found || result.ID.String() == "" {
		return nil, err
	}
	return &result, nil
}

// GetProjectByName returns the project with the given name, or nil if it does
// not exist.
func (c *Client) GetProjectByName(name string) (*Project, error) {
	var result Project
	found, err := c.callOptional("getProjectByName", map[string]string{"name": name}, &result)
	if err != nil || !found || result.ID.String() == "" {
		return nil, err
	}
	return &result, nil
}

// UpdateProjectParams holds parameters for updateProject. Nil fields are not
// sent and therefore left unchanged.
type UpdateProjectParams struct {
	ProjectID       int     `json:"project_id"`
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	OwnerID         *int    `json:"owner_id,omitempty"`
	Identifier      *string `json:"identifier,omitempty"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
	PriorityDefault *int    `json:"priority_default,omitempty"`
	PriorityStart   *int    `json:"priority_start,omitempty"`
	PriorityEnd     *int    `json:"priority_end,omitempty"`
	Email           *string `json:"email,omitempty"`
}

// IsEmpty reports whether no field besides the project ID is set.
func (p UpdateProjectParams) IsEmpty() bool {
	return p.Name == nil && p.Description == nil && p.OwnerID == nil && p.Identifier == nil &&
		p.StartDate == nil && p.EndDate == nil && p.PriorityDefault == nil &&
		p.PriorityStart == nil && p.PriorityEnd == nil && p.Email == nil
}

func (c *Client) UpdateProject(p UpdateProjectParams) error {
	return c.callBool("updateProject", p)
}

// SetProjectActive enables or disables a project.
func (c *Client) SetProjectActive(projectID int, active bool) error {
	method := "disableProject"
	if active {
		method = "enableProject"
	}
	return c.callBool(method, map[string]int{"project_id": projectID})
}

// SetProjectPublic enables or disables public access to a project.
func (c *Client) SetProjectPublic(projectID int, public bool) error {
	method := "disableProjectPublicAccess"
	if public {
		method = "enableProjectPublicAccess"
	}
	return c.callBool(method, map[string]int{"project_id": projectID})
}

// GetProjectUsers returns all project members as a map of user ID to display
// name.
func (c *Client) GetProjectUsers(projectID int) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.Call("getProjectUsers", map[string]int{"project_id": projectID}, &raw); err != nil {
		return nil, err
	}
	return decodeStringMap(raw, "project users")
}

// GetAssignableUsers returns the users that can be assigned to tasks of a
// project as a map of user ID to display name.
func (c *Client) GetAssignableUsers(projectID int) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.Call("getAssignableUsers", map[string]int{"project_id": projectID}, &raw); err != nil {
		return nil, err
	}
	return decodeStringMap(raw, "assignable users")
}

// ---- Category methods -------------------------------------------------------

func (c *Client) GetAllCategories(projectID int) ([]Category, error) {
	var raw json.RawMessage
	if err := c.Call("getAllCategories", map[string]int{"project_id": projectID}, &raw); err != nil {
		return nil, err
	}
	if isEmptyResult(raw) {
		return []Category{}, nil
	}
	var result []Category
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}
	return result, nil
}

// ---- User methods -----------------------------------------------------------

// GetUserByName returns the user with the given username, or nil if it does
// not exist. Requires administrator rights when using the user API.
func (c *Client) GetUserByName(username string) (*User, error) {
	var result User
	found, err := c.callOptional("getUserByName", map[string]string{"username": username}, &result)
	if err != nil || !found || result.ID.String() == "" {
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

// GetTaskTags returns the tags of a task as a map of tag ID to tag name.
func (c *Client) GetTaskTags(taskID int) (map[string]string, error) {
	var raw json.RawMessage
	if err := c.Call("getTaskTags", map[string]int{"task_id": taskID}, &raw); err != nil {
		return nil, err
	}
	return decodeStringMap(raw, "task tags")
}

// SetTaskTags replaces all tags of a task. Tags that do not exist yet are
// created by Kanboard. An empty list removes all tags.
func (c *Client) SetTaskTags(projectID, taskID int, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	var ok bool
	params := map[string]interface{}{
		"project_id": projectID,
		"task_id":    taskID,
		"tags":       tags,
	}
	if err := c.Call("setTaskTags", params, &ok); err != nil {
		return err
	}
	if !ok {
		return errFailed("setTaskTags")
	}
	return nil
}

// CreateTaskParams holds parameters for creating a task. Zero values are not
// sent, so Kanboard applies its defaults.
type CreateTaskParams struct {
	Title               string   `json:"title"`
	ProjectID           int      `json:"project_id"`
	ColumnID            int      `json:"column_id,omitempty"`
	SwimlaneID          int      `json:"swimlane_id,omitempty"`
	OwnerID             int      `json:"owner_id,omitempty"`
	CategoryID          int      `json:"category_id,omitempty"`
	Description         string   `json:"description,omitempty"`
	ColorID             string   `json:"color_id,omitempty"`
	DateDue             string   `json:"date_due,omitempty"`
	DateStarted         string   `json:"date_started,omitempty"`
	Score               int      `json:"score,omitempty"`
	Priority            *int     `json:"priority,omitempty"`
	Reference           string   `json:"reference,omitempty"`
	TimeEstimated       *float64 `json:"time_estimated,omitempty"`
	TimeSpent           *float64 `json:"time_spent,omitempty"`
	RecurrenceStatus    int      `json:"recurrence_status,omitempty"`
	RecurrenceTrigger   int      `json:"recurrence_trigger,omitempty"`
	RecurrenceFactor    int      `json:"recurrence_factor,omitempty"`
	RecurrenceTimeframe int      `json:"recurrence_timeframe,omitempty"`
	RecurrenceBasedate  int      `json:"recurrence_basedate,omitempty"`
	Tags                []string `json:"tags,omitempty"`
}

func (c *Client) CreateTask(p CreateTaskParams) (int, error) {
	return c.callForID("createTask", p)
}

// UpdateTaskParams holds parameters for updateTask. Nil fields are not sent
// and therefore left unchanged. Dates use the format "YYYY-MM-DD HH:MM"; an
// empty date string clears the date.
type UpdateTaskParams struct {
	ID                  int       `json:"id"`
	Title               *string   `json:"title,omitempty"`
	ColorID             *string   `json:"color_id,omitempty"`
	OwnerID             *int      `json:"owner_id,omitempty"`
	DateDue             *string   `json:"date_due,omitempty"`
	Description         *string   `json:"description,omitempty"`
	CategoryID          *int      `json:"category_id,omitempty"`
	Score               *int      `json:"score,omitempty"`
	Priority            *int      `json:"priority,omitempty"`
	RecurrenceStatus    *int      `json:"recurrence_status,omitempty"`
	RecurrenceTrigger   *int      `json:"recurrence_trigger,omitempty"`
	RecurrenceFactor    *int      `json:"recurrence_factor,omitempty"`
	RecurrenceTimeframe *int      `json:"recurrence_timeframe,omitempty"`
	RecurrenceBasedate  *int      `json:"recurrence_basedate,omitempty"`
	Reference           *string   `json:"reference,omitempty"`
	Tags                *[]string `json:"tags,omitempty"`
	DateStarted         *string   `json:"date_started,omitempty"`
	TimeSpent           *float64  `json:"time_spent,omitempty"`
	TimeEstimated       *float64  `json:"time_estimated,omitempty"`
}

// IsEmpty reports whether no field besides the task ID is set.
func (p UpdateTaskParams) IsEmpty() bool {
	data, err := json.Marshal(p)
	return err == nil && string(data) == fmt.Sprintf(`{"id":%d}`, p.ID)
}

// UpdateTask updates the given fields of a task.
func (c *Client) UpdateTask(p UpdateTaskParams) error {
	if p.Tags != nil && *p.Tags == nil {
		empty := []string{}
		p.Tags = &empty
	}
	return c.callBool("updateTask", p)
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
	return c.UpdateTask(UpdateTaskParams{ID: taskID, OwnerID: &userID})
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

// GetComment returns the comment with the given ID, or nil if it does not
// exist.
func (c *Client) GetComment(commentID int) (*Comment, error) {
	var result Comment
	found, err := c.callOptional("getComment", map[string]int{"comment_id": commentID}, &result)
	if err != nil || !found || result.ID.String() == "" {
		return nil, err
	}
	return &result, nil
}

// UpdateComment replaces the content of a comment.
func (c *Client) UpdateComment(commentID int, content string) error {
	params := map[string]interface{}{"id": commentID, "content": content}
	return c.callBool("updateComment", params)
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
	return c.UpdateTask(UpdateTaskParams{ID: taskID, Description: &description})
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

// callBool calls a procedure that returns true on success and false on
// failure.
func (c *Client) callBool(method string, params interface{}) error {
	var raw json.RawMessage
	if err := c.Call(method, params, &raw); err != nil {
		return err
	}
	var ok bool
	if err := json.Unmarshal(raw, &ok); err != nil || !ok {
		return errFailed(method)
	}
	return nil
}

// callOptional calls a procedure that returns an object on success and
// false/null when the object does not exist. It reports whether an object was
// returned.
func (c *Client) callOptional(method string, params, out interface{}) (bool, error) {
	var raw json.RawMessage
	if err := c.Call(method, params, &raw); err != nil {
		return false, err
	}
	if isEmptyResult(raw) {
		return false, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, fmt.Errorf("decode %s result: %w", method, err)
	}
	return true, nil
}

// isEmptyResult reports whether raw is an empty/failed JSON-RPC result.
func isEmptyResult(raw json.RawMessage) bool {
	switch strings.TrimSpace(string(raw)) {
	case "", "null", "false", "[]", "{}":
		return true
	}
	return false
}

// decodeStringMap decodes a PHP associative array of strings. PHP encodes an
// empty array as [] instead of {}, and failures as false/null.
func decodeStringMap(raw json.RawMessage, what string) (map[string]string, error) {
	result := map[string]string{}
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "" || trimmed == "null" || trimmed == "false":
		return result, nil
	case strings.HasPrefix(trimmed, "["):
		var list []string
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("decode %s: %w", what, err)
		}
		for i, name := range list {
			result[strconv.Itoa(i)] = name
		}
		return result, nil
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode %s: %w", what, err)
	}
	return result, nil
}

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
