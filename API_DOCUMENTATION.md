# 学校API文档

Base URL: `https://tsinglanstudent.schoolis.cn`

## 认证机制

### 密码加密

登录密码需要经过两次MD5加密：

```go
// 第一次：MD5(原始密码) -> 转大写十六进制
hash1 := strings.ToUpper(fmt.Sprintf("%X", md5.Sum([]byte(password))))

// 第二次：MD5(hash1 + timestamp) -> 转大写十六进制
combined := hash1 + strconv.FormatUint(timestamp, 10)
hashedPassword := strings.ToUpper(fmt.Sprintf("%X", md5.Sum([]byte(combined))))
```

### Cookie管理

登录成功后，服务器会设置Cookie，后续所有请求都需要携带这些Cookie。建议使用支持自动Cookie管理的HTTP客户端。

---

## API端点

### 1. 获取登录验证码

**端点**: `GET /api/MemberShip/GetStudentCaptchaForLogin`

**说明**: 获取Base64编码的验证码图片，某些情况下可能返回空字符串（无需验证码）

**请求参数**: 无

**响应结构**:
```go
type CaptchaResponse struct {
    State int    `json:"state"` // 0表示成功
    Msg   string `json:"msg"`
    Data  string `json:"data"`  // Base64编码的图片，格式: "data:image/png;base64,..."
}
```

**说明**:
- `Data`字段是Base64编码的PNG图片
- 如果`Data`为空字符串，说明不需要验证码，登录时`captcha`参数传空字符串即可

---

### 2. 用户登录

**端点**: `POST /api/MemberShip/Login?captcha={captcha}`

**Query参数**:
- `captcha`: 验证码（如果不需要验证码则为空字符串）

**请求体**:
```go
type LoginRequest struct {
    Name      string `json:"name"`      // 用户名
    Password  string `json:"password"`  // 加密后的密码
    Timestamp uint64 `json:"timestamp"` // Unix时间戳（秒）
}
```

**响应结构**:
```go
type LoginResponse struct {
    State int    `json:"state"` // 状态码
    Msg   string `json:"msg"`   // 消息
}
```

**状态码说明**:
- `0`: 登录成功
- `1180038`: 验证码错误
- `13` 或 `1010076`: 用户名或密码错误
- 其他: 未知错误

---

### 3. 获取学期列表

**端点**: `GET /api/School/GetSchoolSemesters`

**请求参数**: 无（需要Cookie认证）

**响应结构**:
```go
type SemestersResponse struct {
    State int        `json:"state"`
    Msg   string     `json:"msg"`
    Data  []Semester `json:"data"`
}

type Semester struct {
    ID        uint64 `json:"id"`
    Year      uint64 `json:"year"`      // 学年开始年份
    Semester  uint64 `json:"semester"`  // 学期编号（1或2）
    IsNow     bool   `json:"isNow"`     // 是否为当前学期
    StartDate string `json:"startDate"` // 格式: "2024-09-01T00:00:00+08:00"
    EndDate   string `json:"endDate"`   // 格式: "2025-01-15T00:00:00+08:00"
}
```

---

### 4. 获取科目ID列表

**端点**: `GET /api/LearningTask/GetStuSubjectListForSelect?semesterId={semesterId}`

**Query参数**:
- `semesterId`: 学期ID

**响应结构**:
```go
type SubjectListResponse struct {
    State int              `json:"state"`
    Msg   string           `json:"msg"`
    Data  []SubjectSimple  `json:"data"`
}

type SubjectSimple struct {
    ID   uint64 `json:"id"`   // 科目ID
    Name string `json:"name"` // 科目名称
}
```

**说明**:
- 返回的科目ID可能有重复，需要去重
- 使用这些ID来获取详细的科目信息和成绩

---

### 5. 获取科目的学习任务列表

**端点**: `GET /api/LearningTask/GetList?semesterId={semesterId}&subjectId={subjectId}&pageIndex=1&pageSize=1`

**Query参数**:
- `semesterId`: 学期ID
- `subjectId`: 科目ID
- `pageIndex`: 页码（通常为1）
- `pageSize`: 每页数量（通常为1即可）

**响应结构**:
```go
type TaskListResponse struct {
    State int      `json:"state"`
    Msg   string   `json:"msg"`
    Data  TaskData `json:"data"`
}

type TaskData struct {
    List []TaskItem `json:"list"`
}

type TaskItem struct {
    ID uint64 `json:"id"` // 学习任务ID，用于获取详情
}
```

**说明**:
- 这个API主要用于获取`taskId`，用于下一步获取科目详细信息
- 通常只需要获取第一个任务即可

---

### 6. 获取学习任务详情

**端点**: `GET /api/LearningTask/GetDetail?learningTaskId={taskId}`

**Query参数**:
- `learningTaskId`: 学习任务ID（从上一个API获取）

**响应结构**:
```go
type TaskDetailResponse struct {
    State int          `json:"state"`
    Msg   string       `json:"msg"`
    Data  SubjectDetail `json:"data"`
}

type SubjectDetail struct {
    SubjectName      string `json:"subjectName"`      // 科目名称
    ClassID          uint64 `json:"classId"`          // 班级ID
    SubjectID        uint64 `json:"subjectId"`        // 科目ID
    SchoolSemesterID uint64 `json:"schoolSemesterId"` // 学期ID
}
```

**说明**:
- 获取到的`ClassID`和`SubjectID`用于获取成绩详情

---

### 7. 获取科目成绩详情

**端点**: `GET /api/DynamicScore/GetDynamicScoreDetail?classId={classId}&subjectId={subjectId}&semesterId={semesterId}`

**Query参数**:
- `classId`: 班级ID
- `subjectId`: 科目ID
- `semesterId`: 学期ID

**响应结构**:
```go
type DynamicScoreResponse struct {
    State int               `json:"state"`
    Msg   string            `json:"msg"`
    Data  DynamicScoreData  `json:"data"`
}

type DynamicScoreData struct {
    EvaluationProjectList []EvaluationProject `json:"evaluationProjectList"`
}

type EvaluationProject struct {
    EvaluationProjectEName string              `json:"evaluationProjectEName"` // 评分项目名称（英文）
    Proportion             float64             `json:"proportion"`             // 比例（原始百分比）
    Score                  float64             `json:"score"`                  // 分数
    ScoreLevel             string              `json:"scoreLevel"`             // 等级（A+, A, B+等）
    GPA                    float64             `json:"gpa"`                    // 该项目的GPA
    ScoreIsNull            bool                `json:"scoreIsNull"`            // 是否无成绩
    LearningTaskAndExamList []LearningTask     `json:"learningTaskAndExamList"`// 具体任务列表
    EvaluationProjectList  []EvaluationProject `json:"evaluationProjectList"`  // 子评分项目（嵌套结构）
}

type LearningTask struct {
    Name       string   `json:"name"`       // 任务名称
    Score      *float64 `json:"score"`      // 得分（可能为null）
    TotalScore float64  `json:"totalScore"` // 满分
}
```

**说明**:
- `EvaluationProject`是递归结构，可能包含子评分项目
- 需要根据`ScoreIsNull`过滤掉没有成绩的项目
- 比例需要重新计算（见GPA计算文档）

---

### 8. 获取学期整体动态分数

**端点**: `GET /api/DynamicScore/GetStuSemesterDynamicScore?semesterId={semesterId}`

**Query参数**:
- `semesterId`: 学期ID

**响应结构**:
```go
type SemesterDynamicScoreResponse struct {
    State int                    `json:"state"`
    Msg   string                 `json:"msg"`
    Data  SemesterDynamicData    `json:"data"`
}

type SemesterDynamicData struct {
    StudentSemesterDynamicScoreBasicDtos []SubjectDynamicScore `json:"studentSemesterDynamicScoreBasicDtos"`
}

type SubjectDynamicScore struct {
    ClassID           uint64   `json:"classId"`
    ClassName         string   `json:"className"`
    SubjectID         uint64   `json:"subjectId"`
    SubjectName       string   `json:"subjectName"`
    IsInGrade         bool     `json:"isInGrade"`         // 是否计入GPA
    SubjectScore      *float64 `json:"subjectScore"`      // 科目总分（可能为null）
    ScoreMappingID    uint64   `json:"scoreMappingId"`    // 分数映射ID
    SubjectTotalScore float64  `json:"subjectTotalScore"` // 科目满分
}
```

**说明**:
- `IsInGrade`标识该科目是否计入GPA（比自己判断更可靠）
- `SubjectScore`可能包含额外加分，需要与计算出的分数对比

---

### 9. 获取官方GPA

**端点**: `GET /api/DynamicScore/GetGpa?semesterId={semesterId}`

**Query参数**:
- `semesterId`: 学期ID

**响应结构**:
```go
type GpaResponse struct {
    State int     `json:"state"`
    Msg   string  `json:"msg"`
    Data  float64 `json:"data"` // GPA值，可能为null（未发布）
}
```

**说明**:
- 这是学校官方计算的GPA
- 如果GPA未发布，`Data`可能为`null`
- 可以与自己计算的GPA进行对比

---

## 典型API调用流程

```
1. GetStudentCaptchaForLogin (获取验证码)
   ↓
2. Login (登录，获取Cookie)
   ↓
3. GetSchoolSemesters (获取学期列表，选择学期)
   ↓
4. GetStuSubjectListForSelect (获取科目ID列表)
   ↓
5. 对每个科目ID:
   ├─ GetList (获取taskId)
   ├─ GetDetail (获取classId等信息)
   └─ GetDynamicScoreDetail (获取成绩详情)
   ↓
6. GetStuSemesterDynamicScore (获取整体信息，确认是否计入GPA)
   ↓
7. GetGpa (获取官方GPA，用于对比)
```

---

## 错误处理

所有API响应都包含`state`字段：
- `state == 0`: 成功
- `state != 0`: 失败，查看`msg`字段获取错误信息

常见错误：
- 未登录/Cookie过期：需要重新登录
- 参数错误：检查必需参数是否提供
- 权限不足：确认用户有访问权限
