package models

// CaptchaResponse represents the captcha API response
type CaptchaResponse struct {
	State int    `json:"state"`
	Msg   string `json:"msg"`
	Data  string `json:"data"` // Base64 encoded image
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Name      string `json:"name"`
	Password  string `json:"password"`
	Timestamp uint64 `json:"timestamp"`
}

// LoginResponse represents the login API response
type LoginResponse struct {
	State int    `json:"state"`
	Msg   string `json:"msg"`
}

// SemestersResponse represents the semesters API response
type SemestersResponse struct {
	State int        `json:"state"`
	Msg   string     `json:"msg"`
	Data  []Semester `json:"data"`
}

// Semester represents a school semester
type Semester struct {
	ID        uint64 `json:"id"`
	Year      uint64 `json:"year"`
	Semester  uint64 `json:"semester"`
	IsNow     bool   `json:"isNow"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// SubjectListResponse represents the subject list API response
type SubjectListResponse struct {
	State int             `json:"state"`
	Msg   string          `json:"msg"`
	Data  []SubjectSimple `json:"data"`
}

// SubjectSimple represents a simple subject with ID and name
type SubjectSimple struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// TaskListResponse represents the learning task list API response
type TaskListResponse struct {
	State int      `json:"state"`
	Msg   string   `json:"msg"`
	Data  TaskData `json:"data"`
}

// TaskData contains the list of tasks
type TaskData struct {
	List []TaskItem `json:"list"`
}

// TaskItem represents a learning task
type TaskItem struct {
	ID uint64 `json:"id"`
}

// TaskDetailResponse represents the task detail API response
type TaskDetailResponse struct {
	State int           `json:"state"`
	Msg   string        `json:"msg"`
	Data  SubjectDetail `json:"data"`
}

// SubjectDetail contains detailed information about a subject
type SubjectDetail struct {
	SubjectName      string `json:"subjectName"`
	ClassID          uint64 `json:"classId"`
	SubjectID        uint64 `json:"subjectId"`
	SchoolSemesterID uint64 `json:"schoolSemesterId"`
}

// DynamicScoreResponse represents the dynamic score API response
type DynamicScoreResponse struct {
	State int              `json:"state"`
	Msg   string           `json:"msg"`
	Data  DynamicScoreData `json:"data"`
}

// DynamicScoreData contains evaluation projects
type DynamicScoreData struct {
	EvaluationProjectList []EvaluationProject `json:"evaluationProjectList"`
}

// EvaluationProject represents an evaluation project (can be nested)
type EvaluationProject struct {
	EvaluationProjectEName  string              `json:"evaluationProjectEName"`
	Proportion              float64             `json:"proportion"`
	Score                   float64             `json:"score"`
	ScoreLevel              string              `json:"scoreLevel"`
	GPA                     float64             `json:"gpa"`
	ScoreIsNull             bool                `json:"scoreIsNull"`
	LearningTaskAndExamList []LearningTask      `json:"learningTaskAndExamList"`
	EvaluationProjectList   []EvaluationProject `json:"evaluationProjectList"` // Nested structure
}

// LearningTask represents a learning task or exam
type LearningTask struct {
	Name       string   `json:"name"`
	Score      *float64 `json:"score"`      // Nullable
	TotalScore float64  `json:"totalScore"`
}

// SemesterDynamicScoreResponse represents the semester dynamic score API response
type SemesterDynamicScoreResponse struct {
	State int                 `json:"state"`
	Msg   string              `json:"msg"`
	Data  SemesterDynamicData `json:"data"`
}

// SemesterDynamicData contains semester score information
type SemesterDynamicData struct {
	StudentSemesterDynamicScoreBasicDtos []SubjectDynamicScore `json:"studentSemesterDynamicScoreBasicDtos"`
}

// SubjectDynamicScore represents a subject's dynamic score
type SubjectDynamicScore struct {
	ClassID           uint64   `json:"classId"`
	ClassName         string   `json:"className"`
	SubjectID         uint64   `json:"subjectId"`
	SubjectName       string   `json:"subjectName"`
	IsInGrade         bool     `json:"isInGrade"`
	SubjectScore      *float64 `json:"subjectScore"`      // Nullable
	ScoreMappingID    uint64   `json:"scoreMappingId"`
	SubjectTotalScore float64  `json:"subjectTotalScore"`
}

// GpaResponse represents the GPA API response
type GpaResponse struct {
	State int     `json:"state"`
	Msg   string  `json:"msg"`
	Data  float64 `json:"data"` // Can be null if not published
}
