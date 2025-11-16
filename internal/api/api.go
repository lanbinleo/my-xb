package api

import (
	"fmt"
	"myxb/internal/auth"
	"myxb/internal/client"
	"myxb/internal/models"
	"strconv"
	"time"
)

// API wraps the HTTP client and provides API methods
type API struct {
	client *client.Client
}

// New creates a new API instance
func New(c *client.Client) *API {
	return &API{client: c}
}

// GetCaptcha retrieves the login captcha
func (a *API) GetCaptcha() (*models.CaptchaResponse, error) {
	var resp models.CaptchaResponse
	err := a.client.GetJSON("/api/MemberShip/GetStudentCaptchaForLogin", nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return &resp, nil
}

// Login performs user login
func (a *API) Login(username, password, captcha string) error {
	timestamp := uint64(time.Now().Unix())
	hashedPassword := auth.HashPassword(password, timestamp)

	loginReq := models.LoginRequest{
		Name:      username,
		Password:  hashedPassword,
		Timestamp: timestamp,
	}

	queryParams := map[string]string{
		"captcha": captcha,
	}

	var resp models.LoginResponse
	err := a.client.PostJSON("/api/MemberShip/Login", queryParams, loginReq, &resp)
	if err != nil {
		return err
	}

	if resp.State != 0 {
		switch resp.State {
		case 1180038:
			return fmt.Errorf("incorrect captcha")
		case 13, 1010076:
			return fmt.Errorf("incorrect username or password")
		default:
			return fmt.Errorf("login failed: %s", resp.Msg)
		}
	}

	return nil
}

// LoginWithPasswordHash performs login with an already-hashed password (first MD5 only)
func (a *API) LoginWithPasswordHash(username, passwordHash, captcha string) error {
	timestamp := uint64(time.Now().Unix())

	// Apply second MD5 hash with timestamp
	finalHash := auth.SecondHash(passwordHash, timestamp)

	loginReq := models.LoginRequest{
		Name:      username,
		Password:  finalHash,
		Timestamp: timestamp,
	}

	queryParams := map[string]string{
		"captcha": captcha,
	}

	var resp models.LoginResponse
	err := a.client.PostJSON("/api/MemberShip/Login", queryParams, loginReq, &resp)
	if err != nil {
		return err
	}

	if resp.State != 0 {
		switch resp.State {
		case 1180038:
			return fmt.Errorf("incorrect captcha")
		case 13, 1010076:
			return fmt.Errorf("incorrect username or password")
		default:
			return fmt.Errorf("login failed: %s", resp.Msg)
		}
	}

	return nil
}

// GetSemesters retrieves the list of semesters
func (a *API) GetSemesters() ([]models.Semester, error) {
	var resp models.SemestersResponse
	err := a.client.GetJSON("/api/School/GetSchoolSemesters", nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return resp.Data, nil
}

// GetSubjectList retrieves the subject list for a semester
func (a *API) GetSubjectList(semesterID uint64) ([]models.SubjectSimple, error) {
	queryParams := map[string]string{
		"semesterId": strconv.FormatUint(semesterID, 10),
	}

	var resp models.SubjectListResponse
	err := a.client.GetJSON("/api/LearningTask/GetStuSubjectListForSelect", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	// Deduplicate subjects by ID
	seen := make(map[uint64]bool)
	unique := []models.SubjectSimple{}
	for _, subject := range resp.Data {
		if !seen[subject.ID] {
			seen[subject.ID] = true
			unique = append(unique, subject)
		}
	}

	return unique, nil
}

// GetTaskList retrieves the task list for a subject
func (a *API) GetTaskList(semesterID, subjectID uint64) ([]models.TaskItem, error) {
	queryParams := map[string]string{
		"semesterId": strconv.FormatUint(semesterID, 10),
		"subjectId":  strconv.FormatUint(subjectID, 10),
		"pageIndex":  "1",
		"pageSize":   "1",
	}

	var resp models.TaskListResponse
	err := a.client.GetJSON("/api/LearningTask/GetList", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return resp.Data.List, nil
}

// GetTaskDetail retrieves detailed information about a learning task
func (a *API) GetTaskDetail(taskID uint64) (*models.SubjectDetail, error) {
	queryParams := map[string]string{
		"learningTaskId": strconv.FormatUint(taskID, 10),
	}

	var resp models.TaskDetailResponse
	err := a.client.GetJSON("/api/LearningTask/GetDetail", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// GetDynamicScoreDetail retrieves the dynamic score details for a subject
func (a *API) GetDynamicScoreDetail(classID, subjectID, semesterID uint64) (*models.DynamicScoreData, error) {
	queryParams := map[string]string{
		"classId":    strconv.FormatUint(classID, 10),
		"subjectId":  strconv.FormatUint(subjectID, 10),
		"semesterId": strconv.FormatUint(semesterID, 10),
	}

	var resp models.DynamicScoreResponse
	err := a.client.GetJSON("/api/DynamicScore/GetDynamicScoreDetail", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// GetSemesterDynamicScore retrieves the semester-wide dynamic scores
func (a *API) GetSemesterDynamicScore(semesterID uint64) ([]models.SubjectDynamicScore, error) {
	queryParams := map[string]string{
		"semesterId": strconv.FormatUint(semesterID, 10),
	}

	var resp models.SemesterDynamicScoreResponse
	err := a.client.GetJSON("/api/DynamicScore/GetStuSemesterDynamicScore", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	return resp.Data.StudentSemesterDynamicScoreBasicDtos, nil
}

// GetGPA retrieves the official GPA for a semester
func (a *API) GetGPA(semesterID uint64) (*float64, error) {
	queryParams := map[string]string{
		"semesterId": strconv.FormatUint(semesterID, 10),
	}

	var resp models.GpaResponse
	err := a.client.GetJSON("/api/DynamicScore/GetGpa", queryParams, &resp)
	if err != nil {
		return nil, err
	}

	if resp.State != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	// Data can be null if GPA not published
	if resp.Data == 0 {
		return nil, nil
	}

	return &resp.Data, nil
}
