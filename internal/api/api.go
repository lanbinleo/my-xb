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

// Error code constants
const (
	ErrCodeInvalidCaptcha     = 1180038
	ErrCodeInvalidCredentials = 13
	ErrCodeAuthFailed         = 1010076
)

// checkAPIResponse checks the generic API response state and returns an error if state != 0
func checkAPIResponse(state int, msg string) error {
	if state != 0 {
		return fmt.Errorf("API error: %s", msg)
	}
	return nil
}

// checkLoginResponse checks the login response state and returns a specific error message
func checkLoginResponse(state int, msg string) error {
	if state != 0 {
		switch state {
		case ErrCodeInvalidCaptcha:
			return fmt.Errorf("incorrect captcha")
		case ErrCodeInvalidCredentials, ErrCodeAuthFailed:
			return fmt.Errorf("incorrect username or password")
		default:
			return fmt.Errorf("login failed: %s", msg)
		}
	}
	return nil
}

// GetCaptcha retrieves the login captcha
func (a *API) GetCaptcha() (*models.CaptchaResponse, error) {
	var resp models.CaptchaResponse
	err := a.client.GetJSON("/api/MemberShip/GetStudentCaptchaForLogin", nil, &resp)
	if err != nil {
		return nil, err
	}

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	return checkLoginResponse(resp.State, resp.Msg)
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

	return checkLoginResponse(resp.State, resp.Msg)
}

// GetSemesters retrieves the list of semesters
func (a *API) GetSemesters() ([]models.Semester, error) {
	var resp models.SemestersResponse
	err := a.client.GetJSON("/api/School/GetSchoolSemesters", nil, &resp)
	if err != nil {
		return nil, err
	}

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
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

	if err := checkAPIResponse(resp.State, resp.Msg); err != nil {
		return nil, err
	}

	// Data can be null if GPA not published
	if resp.Data == 0 {
		return nil, nil
	}

	return &resp.Data, nil
}
