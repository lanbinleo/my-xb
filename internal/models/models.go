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
	ID                     uint64   `json:"id"`
	Name                   string   `json:"name"`
	SubjectName            string   `json:"subjectName"`
	Score                  *float64 `json:"score"`
	TotalScore             float64  `json:"totalScore"`
	Status                 string   `json:"status"`
	EvaluationProjectEName string   `json:"evaluationProjectEName"`
	EvaluationProjectID    uint64   `json:"evaluationProjectId"`
	FinishState            uint8    `json:"finishState"`
	BeginTime              string   `json:"beginTime"`
	EndTime                string   `json:"endTime"`
	SyncTime               string   `json:"syncTime"`
	LearningTaskState      uint8    `json:"learningTaskState"`
	TypeName               string   `json:"typeName"`
	TypeEName              string   `json:"typeEName"`
	ScoreType              uint8    `json:"scoreType"`
	LevelString            string   `json:"levelString"`
	IsExempt               bool     `json:"isExempt"`
	CategoryID             uint64   `json:"category_id,omitempty"`
	CategoryName           string   `json:"category_name,omitempty"`
	CategoryEName          string   `json:"category_ename,omitempty"`
	CategoryProportion     float64  `json:"category_proportion,omitempty"`
	EstimatedSubjectWeight *float64 `json:"estimated_subject_weight,omitempty"`
	IsInSubjectScore       *bool    `json:"is_in_subject_score,omitempty"`
}

// TaskDetailResponse represents the task detail API response
type TaskDetailResponse struct {
	State int           `json:"state"`
	Msg   string        `json:"msg"`
	Data  SubjectDetail `json:"data"`
}

// SubjectDetail contains detailed information about a subject
type SubjectDetail struct {
	SubjectName      string                  `json:"subjectName"`
	ClassID          uint64                  `json:"classId"`
	SubjectID        uint64                  `json:"subjectId"`
	SchoolSemesterID uint64                  `json:"schoolSemesterId"`
	LearningTaskName string                  `json:"learningTaskName"`
	IsInSubjectScore bool                    `json:"isInSubjectScore"`
	EvaProjects      []TaskEvaluationProject `json:"evaProjects"`
}

// TaskEvaluationProject is the category metadata attached to one learning task detail.
type TaskEvaluationProject struct {
	ID                  uint64  `json:"id"`
	Name                string  `json:"name"`
	EName               string  `json:"eName"`
	ParentProID         uint64  `json:"parentProId"`
	ProPath             string  `json:"proPath"`
	Proportion          float64 `json:"proportion"`
	IsDisplayProportion bool    `json:"isDisplayProportion"`
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
	EvaluationProjectName   string              `json:"evaluationProjectName"`
	EvaluationProjectID     uint64              `json:"evaluationProjectId"`
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
	Score      *float64 `json:"score"` // Nullable
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
	SubjectScore      *float64 `json:"subjectScore"` // Nullable
	ScoreMappingID    uint64   `json:"scoreMappingId"`
	SubjectTotalScore float64  `json:"subjectTotalScore"`
}

// GpaResponse represents the GPA API response
type GpaResponse struct {
	State int     `json:"state"`
	Msg   string  `json:"msg"`
	Data  float64 `json:"data"` // Can be null if not published
}

// ScheduleRequest represents the schedule request payload.
type ScheduleRequest struct {
	BeginTime string `json:"beginTime"`
	EndTime   string `json:"endTime"`
}

// ScheduleListResponse represents the schedule API response.
type ScheduleListResponse struct {
	State int            `json:"state"`
	Msg   string         `json:"msg"`
	MsgCN string         `json:"msgCN"`
	MsgEN string         `json:"msgEN"`
	Data  []ScheduleItem `json:"data"`
}

// ScheduleItem represents a single schedule entry.
type ScheduleItem struct {
	ScheduleType      int               `json:"scheduleType"`
	RearrangementType int               `json:"rearrangementType"`
	Color             string            `json:"color"`
	IsAllDay          bool              `json:"isAllDay"`
	BeginTime         string            `json:"beginTime"`
	EndTime           string            `json:"endTime"`
	FormalCourseOrder int               `json:"formalCourseOrder"`
	TeacherList       []ScheduleTeacher `json:"teacherList"`
	PlaygroundName    string            `json:"playgroundName"`
	PlaygroundEName   string            `json:"playgroundEName"`
	ClassInfo         ScheduleClassInfo `json:"classInfo"`
	Remark            string            `json:"remark"`
	EName             string            `json:"eName"`
	ID                uint64            `json:"id"`
	Name              string            `json:"name"`
}

// ScheduleTeacher represents a teacher attached to a schedule item.
type ScheduleTeacher struct {
	EName string `json:"eName"`
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
}

// ScheduleClassInfo contains class metadata for a schedule item.
type ScheduleClassInfo struct {
	ID         uint64 `json:"id"`
	ClassName  string `json:"className"`
	ClassEName string `json:"classEName"`
	Grade      int    `json:"grade"`
	ClassCode  string `json:"classCode"`
}
