# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool for calculating weighted and unweighted GPA from Tsinglan School's Xiaobao system (https://tsinglanstudent.schoolis.cn/). This is a refactor of the original tls-xb project.

## Build and Run Commands

```bash
# Build the project
go build -o myxb ./cmd/myxb

# On Windows
go build -o myxb.exe ./cmd/myxb

# Run the program
./myxb              # Calculate GPA using saved credentials
./myxb login        # Login and save credentials
./myxb help         # Show help message
```

## Architecture

### Directory Structure

- `cmd/myxb/` - Main application entry point and CLI commands
  - `main.go` - Command routing (login, help, default GPA calculation)
  - `login.go` - Login flow and credential input handlers
  - `gpa.go` - GPA calculation command implementation
  - `colors.go` - Terminal color output utilities

- `internal/` - Internal packages (not importable by external projects)
  - `api/` - API client wrapper for all Xiaobao API endpoints
  - `auth/` - Password hashing (double MD5: MD5(password), then MD5(hash+timestamp))
  - `client/` - HTTP client with automatic cookie management
  - `config/` - Credential storage (~/.myxb/config.json)
  - `models/` - Data structures for API requests/responses

- `pkg/gpa/` - Public GPA calculation logic
  - `calculator.go` - Core GPA calculation algorithms
  - `score_mapping.go` - Score to GPA conversion, course classification
  - `score_mapping.json` - Weighted (max 4.8) and non-weighted (max 4.3) GPA mappings (embedded)
  - `course_classification.json` - Lists of weighted/unweighted courses (embedded)

### Key Data Flow

1. **Authentication**: Double MD5 password hashing → Login API → Cookie storage
2. **GPA Calculation Pipeline**:
   - GetSemesters → identify current semester
   - GetSubjectList → get all subjects
   - For each subject:
     - GetTaskList → GetTaskDetail → extract classID, subjectID
     - GetDynamicScoreDetail → get evaluation projects and scores
     - AdjustProportions → normalize proportions to 100% (excluding null scores)
     - CalculateSubjectScore → compute 0-100 score
     - ScoreToGPA → convert to GPA using weighted/non-weighted mapping
   - GetSemesterDynamicScore → get official scores and IsInGrade flags
   - CalculateGPA → weighted average across all subjects

### GPA Calculation Details

**Weighted Courses** (max GPA 4.8):
- AP courses (name contains "AP")
- A Level courses (name contains "A Level")
- AS courses (name contains "AS ")
- Specific advanced courses: Linear Algebra, Modern Physics and Optics, Multivariable Calculus
- Explicit list in `course_classification.json` ("weighted" array)

**Course Weights**:
- Regular courses: weight = 1.0
- Elective courses: weight = 0.5 (name contains "Ele" OR name is "C-Humanities")

**Score Adjustment**:
- Evaluation projects with null scores are excluded
- Remaining proportions are normalized to sum to 100%
- Handles nested evaluation projects recursively
- Official scores from GetSemesterDynamicScore API override calculated scores

**Final GPA Formula**:
```
Weighted GPA = Σ(subject_GPA × course_weight) / Σ(course_weight)
```

Only subjects with `IsInGrade = true` and non-NaN GPA are included.

## Important Implementation Notes

1. **Embedded JSON Files**: `score_mapping.json` and `course_classification.json` are embedded using `go:embed` directive. Changes require rebuild.

2. **Double MD5 Hashing**:
   - First hash (stored locally): `MD5(password)`
   - Second hash (sent to API): `MD5(first_hash + unix_timestamp)`

3. **Cookie Management**: The HTTP client automatically handles cookies. After successful login, all subsequent requests include session cookies.

4. **Proportion Adjustment**: Critical step - API returns raw proportions that don't account for missing assignments. Must recalculate to sum to 100% across non-null projects.

5. **Course Classification Priority**:
   - Explicit unweighted list (highest priority)
   - Explicit weighted list
   - Keyword matching (AP, A Level, AS)

6. **Score Precision**: All scores are rounded to 1 decimal place before GPA conversion.

## API Dependencies

Base URL: `https://tsinglanstudent.schoolis.cn`

Critical endpoints (in order of typical usage):
1. `/api/MemberShip/GetStudentCaptchaForLogin` - Get captcha image
2. `/api/MemberShip/Login` - Authenticate with hashed password
3. `/api/School/GetSchoolSemesters` - Get semester list
4. `/api/LearningTask/GetStuSubjectListForSelect` - Get subject IDs (requires deduplication)
5. `/api/LearningTask/GetList` - Get task ID for subject
6. `/api/LearningTask/GetDetail` - Get classID and subjectID
7. `/api/DynamicScore/GetDynamicScoreDetail` - Get evaluation projects and scores
8. `/api/DynamicScore/GetStuSemesterDynamicScore` - Get official scores and IsInGrade flags
9. `/api/DynamicScore/GetGpa` - Get official GPA for comparison

See `API_DOCUMENTATION.md` and `GPA_CALCULATION.md` for detailed specifications.

## Testing Workflow

When making changes:
1. Build the project: `go build -o myxb.exe ./cmd/myxb`
2. Test login flow: `./myxb login`
3. Test GPA calculation: `./myxb`
4. Verify output matches expected format with color-coded grades

## Common Pitfalls

- Don't forget to normalize proportions before calculating subject scores
- Check both weighted and unweighted course classification lists before defaulting to keyword matching
- Handle null scores in evaluation projects - they should be excluded from calculations
- Respect `IsInGrade` flag from API - some subjects may not count toward GPA
- Remember that elective courses have 0.5 weight, not 1.0
