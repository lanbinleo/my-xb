package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"myxb/internal/api"
	"myxb/internal/models"
	"myxb/pkg/gpa"
	"os"
	"strconv"
	"strings"
)

type semesterReport struct {
	Semester       models.Semester
	Subjects       []gpa.Subject
	TasksBySubject map[uint64][]models.TaskItem
	Result         gpa.CalculatedGPA
	OfficialGPA    *float64
	OfficialGPAErr error
}

type jsonOutput struct {
	Version string               `json:"version"`
	Reports []jsonSemesterReport `json:"reports"`
}

type jsonSemesterReport struct {
	Semester jsonSemesterMeta    `json:"semester"`
	Summary  jsonSummary         `json:"summary"`
	Subjects []jsonSubjectReport `json:"subjects"`
}

type jsonSemesterMeta struct {
	ID       uint64 `json:"id"`
	Year     uint64 `json:"year"`
	YearEnd  uint64 `json:"year_end"`
	Semester uint64 `json:"semester"`
	IsNow    bool   `json:"is_now"`
	Label    string `json:"label"`
}

type jsonSummary struct {
	WeightedGPA      *float64 `json:"weighted_gpa"`
	MaxGPA           *float64 `json:"max_gpa"`
	UnweightedGPA    *float64 `json:"unweighted_gpa"`
	UnweightedMaxGPA *float64 `json:"unweighted_max_gpa"`
	OfficialGPA      *float64 `json:"official_gpa,omitempty"`
	OfficialGPAError string   `json:"official_gpa_error,omitempty"`
	OfficialDiff     *float64 `json:"official_diff,omitempty"`
	SubjectCount     int      `json:"subject_count"`
}

type jsonSubjectReport struct {
	ID                uint64                  `json:"id"`
	Name              string                  `json:"name"`
	ASCIIName         string                  `json:"ascii_name"`
	Score             float64                 `json:"score"`
	OfficialScore     *float64                `json:"official_score,omitempty"`
	ExtraCredit       float64                 `json:"extra_credit,omitempty"`
	GPA               float64                 `json:"gpa"`
	UnweightedGPA     float64                 `json:"unweighted_gpa"`
	MaxGPA            float64                 `json:"max_gpa"`
	UnweightedMaxGPA  float64                 `json:"unweighted_max_gpa"`
	Weight            float64                 `json:"weight"`
	IsWeighted        bool                    `json:"is_weighted"`
	IsElective        bool                    `json:"is_elective"`
	IsInGrade         bool                    `json:"is_in_grade"`
	Type              string                  `json:"type"`
	EvaluationDetails []jsonEvaluationProject `json:"evaluation_details"`
	Tasks             []models.TaskItem       `json:"tasks,omitempty"`
}

type jsonEvaluationProject struct {
	Name       string                  `json:"name"`
	ASCIIName  string                  `json:"ascii_name"`
	Proportion float64                 `json:"proportion"`
	Score      float64                 `json:"score"`
	ScoreLevel string                  `json:"score_level"`
	GPA        float64                 `json:"gpa"`
	Children   []jsonEvaluationProject `json:"children,omitempty"`
}

func calculateGPA(apiClient *api.API, opts gpaCommandOptions) {
	reports, err := collectSemesterReports(apiClient, opts)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	rendered, err := renderSemesterReports(reports, opts)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	fmt.Print(rendered)
	if !strings.HasSuffix(rendered, "\n") {
		fmt.Println()
	}

	if err := maybeExportOutput(rendered, reports, opts); err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

func collectSemesterReports(apiClient *api.API, opts gpaCommandOptions) ([]semesterReport, error) {
	logProgress(opts, "Fetching semesters...")
	semesters, err := apiClient.GetSemesters()
	if err != nil {
		return nil, fmt.Errorf("failed to get semesters: %w", err)
	}

	selectedSemesters, err := resolveSemesterSelection(semesters, opts)
	if err != nil {
		return nil, err
	}

	reports := make([]semesterReport, 0, len(selectedSemesters))
	for _, semester := range selectedSemesters {
		if !opts.suppressProgress() {
			fmt.Println()
			printSuccess("Calculating GPA for " + semesterLabel(semester))
			fmt.Println()
		}

		report, err := collectSingleSemesterReport(apiClient, semester, opts)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func collectSingleSemesterReport(apiClient *api.API, semester models.Semester, opts gpaCommandOptions) (semesterReport, error) {
	logProgress(opts, "Fetching subjects...")
	subjects, err := apiClient.GetSubjectList(semester.ID)
	if err != nil {
		return semesterReport{}, fmt.Errorf("failed to get subjects: %w", err)
	}

	if !opts.suppressProgress() {
		printSuccess(fmt.Sprintf("Found %d subjects", len(subjects)))
		fmt.Println()
	}

	logProgress(opts, "Fetching semester-wide scores...")
	semesterScores, _ := apiClient.GetSemesterDynamicScore(semester.ID)

	semesterScoreMap := make(map[uint64]*models.SubjectDynamicScore)
	for idx := range semesterScores {
		semesterScoreMap[semesterScores[idx].SubjectID] = &semesterScores[idx]
	}

	calculatedSubjects := []gpa.Subject{}
	subjectTasksMap := make(map[uint64][]models.TaskItem)

	logProgress(opts, "Fetching scores for each subject...")
	if !opts.suppressProgress() {
		fmt.Println()
	}

	for _, subject := range subjects {
		tasks, err := apiClient.GetTaskList(semester.ID, subject.ID)
		if err != nil || len(tasks) == 0 {
			continue
		}

		detail, err := apiClient.GetTaskDetail(tasks[0].ID)
		if err != nil {
			continue
		}

		dynamicScore, err := apiClient.GetDynamicScoreDetail(detail.ClassID, subject.ID, semester.ID)
		if err != nil {
			continue
		}

		dynamicInfo := semesterScoreMap[subject.ID]
		isElective := strings.Contains(subject.Name, ElectiveCourseKeyword)
		calculatedSubjects = append(calculatedSubjects, gpa.ProcessSubject(detail, dynamicScore, dynamicInfo, isElective))
		subjectTasksMap[subject.ID] = tasks
	}

	result := gpa.CalculateGPA(calculatedSubjects)
	officialGPA, officialGPAErr := apiClient.GetGPA(semester.ID)

	return semesterReport{
		Semester:       semester,
		Subjects:       calculatedSubjects,
		TasksBySubject: subjectTasksMap,
		Result:         result,
		OfficialGPA:    officialGPA,
		OfficialGPAErr: officialGPAErr,
	}, nil
}

func resolveSemesterSelection(semesters []models.Semester, opts gpaCommandOptions) ([]models.Semester, error) {
	if len(semesters) == 0 {
		return nil, fmt.Errorf("no semesters found")
	}

	selector := strings.TrimSpace(opts.SemesterSelector)
	if selector == "" {
		if !opts.Clean {
			selected, err := promptForSemesterSelection(semesters)
			if err != nil {
				return nil, err
			}
			return []models.Semester{*selected}, nil
		}

		if current := currentSemester(semesters); current != nil {
			return []models.Semester{*current}, nil
		}

		return []models.Semester{semesters[len(semesters)-1]}, nil
	}

	selected := make([]models.Semester, 0)
	seen := make(map[uint64]bool)

	for _, token := range strings.Split(selector, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		matches, err := matchSemesterSelector(semesters, token)
		if err != nil {
			return nil, err
		}

		for _, semester := range matches {
			if !seen[semester.ID] {
				selected = append(selected, semester)
				seen[semester.ID] = true
			}
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no semesters matched %q", selector)
	}

	return selected, nil
}

func promptForSemesterSelection(semesters []models.Semester) (*models.Semester, error) {
	fmt.Println()

	currentIdx := -1
	displayCount := len(semesters)
	if displayCount > MaxDisplaySemesters {
		displayCount = MaxDisplaySemesters
	}

	for idx := displayCount - 1; idx >= 0; idx-- {
		suffix := ""
		if semesters[idx].IsNow {
			suffix = cyan(" (current)")
			currentIdx = idx
		}
		fmt.Printf("%s %s%s\n", gray(fmt.Sprintf("[%d]", idx)), semesterLabel(semesters[idx]), suffix)
	}

	fmt.Println()

	if currentIdx >= 0 {
		fmt.Printf("Select a semester to calculate GPA for (default %s): ", gray(fmt.Sprintf("[%d]", currentIdx)))
	} else {
		fmt.Print("Select a semester to calculate GPA for: ")
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read semester selection: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" && currentIdx >= 0 {
		return &semesters[currentIdx], nil
	}

	selectedIndex, err := strconv.Atoi(input)
	if err != nil || selectedIndex < 0 || selectedIndex >= len(semesters) {
		return nil, fmt.Errorf("invalid semester selection")
	}

	return &semesters[selectedIndex], nil
}

func matchSemesterSelector(semesters []models.Semester, token string) ([]models.Semester, error) {
	lowerToken := strings.ToLower(strings.TrimSpace(token))

	switch lowerToken {
	case "current":
		current := currentSemester(semesters)
		if current == nil {
			return nil, fmt.Errorf("no current semester found")
		}
		return []models.Semester{*current}, nil
	case "all":
		return semesters, nil
	}

	if index, err := strconv.Atoi(token); err == nil {
		if index < 0 || index >= len(semesters) {
			return nil, fmt.Errorf("semester index %d out of range", index)
		}
		return []models.Semester{semesters[index]}, nil
	}

	parts := strings.Split(token, "-")
	switch len(parts) {
	case 2:
		first, errFirst := strconv.ParseUint(parts[0], 10, 64)
		second, errSecond := strconv.ParseUint(parts[1], 10, 64)
		if errFirst != nil || errSecond != nil {
			break
		}

		if second <= 3 {
			return filterSemesters(semesters, func(semester models.Semester) bool {
				return semester.Year == first && semester.Semester == second
			}, token)
		}

		if second == first+1 {
			return filterSemesters(semesters, func(semester models.Semester) bool {
				return semester.Year == first
			}, token)
		}
	case 3:
		first, errFirst := strconv.ParseUint(parts[0], 10, 64)
		second, errSecond := strconv.ParseUint(parts[1], 10, 64)
		third, errThird := strconv.ParseUint(parts[2], 10, 64)
		if errFirst == nil && errSecond == nil && errThird == nil && second == first+1 {
			return filterSemesters(semesters, func(semester models.Semester) bool {
				return semester.Year == first && semester.Semester == third
			}, token)
		}
	}

	return nil, fmt.Errorf("unsupported semester selector %q", token)
}

func filterSemesters(semesters []models.Semester, match func(models.Semester) bool, token string) ([]models.Semester, error) {
	filtered := make([]models.Semester, 0)
	for _, semester := range semesters {
		if match(semester) {
			filtered = append(filtered, semester)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no semesters matched %q", token)
	}

	return filtered, nil
}

func currentSemester(semesters []models.Semester) *models.Semester {
	for idx := range semesters {
		if semesters[idx].IsNow {
			return &semesters[idx]
		}
	}
	return nil
}

func renderSemesterReports(reports []semesterReport, opts gpaCommandOptions) (string, error) {
	switch opts.Format {
	case formatTable:
		return renderTableReports(reports, opts), nil
	case formatPlain:
		return renderPlainReports(reports, opts), nil
	case formatMarkdown:
		return renderMarkdownReports(reports, opts), nil
	case formatJSON:
		return renderJSONReports(reports, opts)
	default:
		return renderHumanReports(reports, opts), nil
	}
}

func renderHumanReports(reports []semesterReport, opts gpaCommandOptions) string {
	var out strings.Builder
	for idx, report := range reports {
		if len(reports) > 1 {
			out.WriteString(bold(semesterLabel(report.Semester)))
			out.WriteString("\n\n")
		}

		for _, subject := range report.Subjects {
			out.WriteString(renderSubjectTable(subject, opts.ShowTasks, report.TasksBySubject[subject.ID]))
			out.WriteString("\n\n")
		}

		out.WriteString(renderHumanSummary(report, opts))

		if idx < len(reports)-1 {
			out.WriteString("\n\n")
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderHumanSummary(report semesterReport, opts gpaCommandOptions) string {
	var out strings.Builder
	if !opts.suppressProgress() {
		out.WriteString(blue("i"))
		out.WriteString(" Calculating final GPA...\n")
	}

	out.WriteString("\n")
	out.WriteString(strings.Repeat("-", 45))
	out.WriteString("\n")

	if math.IsNaN(report.Result.WeightedGPA) {
		out.WriteString(yellow("!"))
		out.WriteString(" Unable to calculate GPA - no valid subjects found")
		return out.String()
	}

	out.WriteString(fmt.Sprintf("%s %.2f / %.2f %s\n",
		bold("Weighted GPA:"),
		report.Result.WeightedGPA,
		report.Result.MaxGPA,
		gray(fmt.Sprintf("(%.1f%%)", report.Result.WeightedGPA/report.Result.MaxGPA*100))))

	out.WriteString(fmt.Sprintf("%s %.2f / %.2f %s\n",
		bold("Unweighted GPA:"),
		report.Result.UnweightedGPA,
		report.Result.UnweightedMaxGPA,
		gray(fmt.Sprintf("(%.1f%%)", report.Result.UnweightedGPA/report.Result.UnweightedMaxGPA*100))))

	out.WriteString("\n")

	if report.OfficialGPA != nil {
		diff := report.Result.WeightedGPA - *report.OfficialGPA
		if math.Abs(diff) > 0.01 {
			sign := "+"
			color := red
			if diff < 0 {
				sign = ""
				color = green
			}
			diffStr := color(fmt.Sprintf("(%s%.2f)", sign, diff))
			out.WriteString(fmt.Sprintf("%s %.2f %s\n", bold("Official GPA:"), *report.OfficialGPA, diffStr))

			out.WriteString(strings.Repeat("-", 45))
			out.WriteString(fmt.Sprintf("\n\n%s%.2f%s\n%s\n%s\n%s\n",
				bold("Hi! We found a discrepancy of "),
				diff,
				bold(" points in the GPA calculation."),
				"This may be caused by special courses that are "+yellow("weighted differently or excluded")+" from official GPA calculation.",
				"Please report this to the developers so we can improve the accuracy.",
				"Thank you!"))
		} else {
			out.WriteString(fmt.Sprintf("%s %.2f\n", bold("Official GPA:"), *report.OfficialGPA))
			out.WriteString(strings.Repeat("-", 45))
		}
	} else if report.OfficialGPAErr != nil {
		out.WriteString(yellow("Official GPA unavailable: " + report.OfficialGPAErr.Error()))
		out.WriteString("\n")
		out.WriteString(strings.Repeat("-", 45))
	} else {
		out.WriteString(gray("Official GPA not yet published"))
		out.WriteString("\n")
		out.WriteString(strings.Repeat("-", 45))
	}

	out.WriteString("\n")
	if !opts.suppressProgress() {
		out.WriteString(green("✓"))
		out.WriteString(" ")
		out.WriteString(fmt.Sprintf("Calculated GPA from %d subjects", len(report.Result.Subjects)))
	}
	return out.String()
}

func renderTableReports(reports []semesterReport, opts gpaCommandOptions) string {
	var out strings.Builder

	for reportIdx, report := range reports {
		if math.IsNaN(report.Result.WeightedGPA) {
			out.WriteString("Semester: " + semesterLabel(report.Semester) + "\nNo GPA data available.\n")
			continue
		}

		out.WriteString("Semester: " + semesterLabel(report.Semester) + "\n")
		out.WriteString(fmt.Sprintf(
			"GPA: weighted %.2f/%.2f | unweighted %.2f/%.2f",
			report.Result.WeightedGPA,
			report.Result.MaxGPA,
			report.Result.UnweightedGPA,
			report.Result.UnweightedMaxGPA,
		))
		if report.OfficialGPA != nil {
			out.WriteString(fmt.Sprintf(" | official %.2f", *report.OfficialGPA))
		} else if report.OfficialGPAErr != nil {
			out.WriteString(" | official unavailable")
		}
		out.WriteString("\n")

		for subjectIdx, subject := range report.Subjects {
			if subjectIdx == 0 {
				out.WriteString("\n")
			}
			out.WriteString("Subject: " + asciiDisplayText(subject.Name) + "\n")
			out.WriteString(paddedColumns(
				[]int{7, 6, 5, 8},
				[]string{"Score", "Level", "GPA", "Type"},
			))
			out.WriteString("\n")
			out.WriteString(paddedColumns(
				[]int{7, 6, 5, 8},
				[]string{
					fmt.Sprintf("%.1f", subject.Score),
					getScoreLevel(subject),
					fmt.Sprintf("%.2f", subject.GPA),
					subjectTypeLabel(subject),
				},
			))
			out.WriteString("\n")

			if opts.ShowTasks {
				tasks := report.TasksBySubject[subject.ID]
				if len(tasks) > 0 {
					out.WriteString("\n")
					out.WriteString(paddedColumns(
						[]int{34, 8, 10, 6},
						[]string{"Task", "Status", "Score", "Pct"},
					))
					out.WriteString("\n")
					for _, task := range tasks {
						out.WriteString(paddedColumns(
							[]int{34, 8, 10, 6},
							[]string{
								asciiDisplayText(task.Name),
								taskStatusCode(task),
								taskScoreDisplaySpaced(task),
								taskPctDisplayForTable(task),
							},
						))
						out.WriteString("\n")
					}
				}
			}
		}

		if reportIdx < len(reports)-1 {
			out.WriteString("\n")
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderPlainReports(reports []semesterReport, opts gpaCommandOptions) string {
	var out strings.Builder
	for reportIdx, report := range reports {
		out.WriteString("Semester: " + semesterLabel(report.Semester) + "\n")
		if math.IsNaN(report.Result.WeightedGPA) {
			out.WriteString("No GPA data available.\n")
		} else {
			out.WriteString(fmt.Sprintf("Weighted GPA: %.2f / %.2f\n", report.Result.WeightedGPA, report.Result.MaxGPA))
			out.WriteString(fmt.Sprintf("Unweighted GPA: %.2f / %.2f\n", report.Result.UnweightedGPA, report.Result.UnweightedMaxGPA))
			if report.OfficialGPA != nil {
				out.WriteString(fmt.Sprintf("Official GPA: %.2f\n", *report.OfficialGPA))
			} else if report.OfficialGPAErr != nil {
				out.WriteString(fmt.Sprintf("Official GPA unavailable: %v\n", report.OfficialGPAErr))
			}
			out.WriteString(fmt.Sprintf("Subjects: %d\n", len(report.Result.Subjects)))
		}
		for _, subject := range report.Subjects {
			out.WriteString("\n[" + asciiDisplayText(subject.Name) + "]\n")
			out.WriteString(fmt.Sprintf("Score: %.1f\n", subject.Score))
			out.WriteString(fmt.Sprintf("Level: %s\n", getScoreLevel(subject)))
			out.WriteString(fmt.Sprintf("GPA: %.2f\n", subject.GPA))
			out.WriteString(fmt.Sprintf("Type: %s\n", subjectTypeLabel(subject)))
			if opts.ShowTasks {
				out.WriteString("\nTasks:\n")
				for _, task := range report.TasksBySubject[subject.ID] {
					out.WriteString(fmt.Sprintf("- %s | %s | %s", asciiDisplayText(task.Name), taskStatusCode(task), taskScoreDisplaySpaced(task)))
					if pct := taskPctDisplay(task); pct != "-" {
						out.WriteString(" | " + pct + "%")
					}
					out.WriteString("\n")
				}
			}
		}
		if reportIdx < len(reports)-1 {
			out.WriteString("\n")
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderMarkdownReports(reports []semesterReport, opts gpaCommandOptions) string {
	var out strings.Builder
	for reportIdx, report := range reports {
		out.WriteString("## " + semesterLabel(report.Semester) + "\n\n")
		if math.IsNaN(report.Result.WeightedGPA) {
			out.WriteString("- No GPA data available\n")
		} else {
			out.WriteString(fmt.Sprintf("- Weighted GPA: %.2f / %.2f\n", report.Result.WeightedGPA, report.Result.MaxGPA))
			out.WriteString(fmt.Sprintf("- Unweighted GPA: %.2f / %.2f\n", report.Result.UnweightedGPA, report.Result.UnweightedMaxGPA))
			if report.OfficialGPA != nil {
				out.WriteString(fmt.Sprintf("- Official GPA: %.2f\n", *report.OfficialGPA))
			} else if report.OfficialGPAErr != nil {
				out.WriteString(fmt.Sprintf("- Official GPA unavailable: %v\n", report.OfficialGPAErr))
			}
			out.WriteString(fmt.Sprintf("- Subjects: %d\n", len(report.Result.Subjects)))
		}
		for _, subject := range report.Subjects {
			out.WriteString("\n### " + asciiDisplayText(subject.Name) + "\n")
			out.WriteString(fmt.Sprintf("- Score: %.1f\n", subject.Score))
			out.WriteString(fmt.Sprintf("- Level: %s\n", getScoreLevel(subject)))
			out.WriteString(fmt.Sprintf("- GPA: %.2f\n", subject.GPA))
			out.WriteString(fmt.Sprintf("- Type: %s\n", subjectTypeLabel(subject)))
			if opts.ShowTasks {
				out.WriteString("\nTasks:\n")
				for _, task := range report.TasksBySubject[subject.ID] {
					out.WriteString(fmt.Sprintf("- %s: %s, %s", asciiDisplayText(task.Name), taskStatusCode(task), taskScoreDisplaySpaced(task)))
					if pct := taskPctDisplay(task); pct != "-" {
						out.WriteString(", " + pct + "%")
					}
					out.WriteString("\n")
				}
			}
		}
		if reportIdx < len(reports)-1 {
			out.WriteString("\n")
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func renderJSONReports(reports []semesterReport, opts gpaCommandOptions) (string, error) {
	payload := jsonOutput{
		Version: version,
		Reports: make([]jsonSemesterReport, 0, len(reports)),
	}

	for _, report := range reports {
		summary := jsonSummary{
			WeightedGPA:      nullableJSONFloat(report.Result.WeightedGPA),
			MaxGPA:           nullableJSONFloat(report.Result.MaxGPA),
			UnweightedGPA:    nullableJSONFloat(report.Result.UnweightedGPA),
			UnweightedMaxGPA: nullableJSONFloat(report.Result.UnweightedMaxGPA),
			OfficialGPA:      report.OfficialGPA,
			OfficialGPAError: errorString(report.OfficialGPAErr),
			SubjectCount:     len(report.Result.Subjects),
		}
		if report.OfficialGPA != nil && !math.IsNaN(report.Result.WeightedGPA) {
			diff := report.Result.WeightedGPA - *report.OfficialGPA
			summary.OfficialDiff = &diff
		}

		jsonSubjects := make([]jsonSubjectReport, 0, len(report.Subjects))
		for _, subject := range report.Subjects {
			jsonSubject := jsonSubjectReport{
				ID:                subject.ID,
				Name:              subject.Name,
				ASCIIName:         asciiDisplayText(subject.Name),
				Score:             subject.Score,
				OfficialScore:     subject.OfficialScore,
				ExtraCredit:       subject.ExtraCredit,
				GPA:               subject.GPA,
				UnweightedGPA:     subject.UnweightedGPA,
				MaxGPA:            subject.MaxGPA,
				UnweightedMaxGPA:  subject.UnweightedMaxGPA,
				Weight:            subject.Weight,
				IsWeighted:        subject.IsWeighted,
				IsElective:        subject.IsElective,
				IsInGrade:         subject.IsInGrade,
				Type:              subjectTypeCode(subject),
				EvaluationDetails: convertEvaluationProjects(subject.EvaluationDetails),
			}
			if opts.ShowTasks {
				jsonSubject.Tasks = report.TasksBySubject[subject.ID]
			}
			jsonSubjects = append(jsonSubjects, jsonSubject)
		}

		payload.Reports = append(payload.Reports, jsonSemesterReport{
			Semester: jsonSemesterMeta{
				ID:       report.Semester.ID,
				Year:     report.Semester.Year,
				YearEnd:  report.Semester.Year + 1,
				Semester: report.Semester.Semester,
				IsNow:    report.Semester.IsNow,
				Label:    semesterLabel(report.Semester),
			},
			Summary:  summary,
			Subjects: jsonSubjects,
		})
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode JSON output: %w", err)
	}

	return string(encoded), nil
}

func convertEvaluationProjects(projects []models.EvaluationProject) []jsonEvaluationProject {
	out := make([]jsonEvaluationProject, 0, len(projects))
	for _, project := range projects {
		if project.ScoreIsNull {
			continue
		}
		out = append(out, jsonEvaluationProject{
			Name:       project.EvaluationProjectEName,
			ASCIIName:  asciiDisplayText(project.EvaluationProjectEName),
			Proportion: project.Proportion,
			Score:      project.Score,
			ScoreLevel: project.ScoreLevel,
			GPA:        project.GPA,
			Children:   convertEvaluationProjects(project.EvaluationProjectList),
		})
	}
	return out
}

func semesterLabel(semester models.Semester) string {
	return fmt.Sprintf("%d-%d Semester %d", semester.Year, semester.Year+1, semester.Semester)
}

func subjectTypeCode(subject gpa.Subject) string {
	switch {
	case subject.IsWeighted && subject.IsElective:
		return "weighted_elective"
	case subject.IsWeighted:
		return "weighted"
	case subject.IsElective:
		return "elective"
	default:
		return "regular"
	}
}

func taskScoreDisplay(task models.TaskItem) string {
	if task.FinishState != 0 && task.Score != nil {
		return fmt.Sprintf("%.0f/%.0f", *task.Score, task.TotalScore)
	}
	return fmt.Sprintf("-/%.0f", task.TotalScore)
}

func taskScoreDisplaySpaced(task models.TaskItem) string {
	if task.FinishState != 0 && task.Score != nil {
		return fmt.Sprintf("%.0f / %.0f", *task.Score, task.TotalScore)
	}
	return fmt.Sprintf("- / %.0f", task.TotalScore)
}

func taskPctDisplay(task models.TaskItem) string {
	if task.FinishState != 0 && task.Score != nil && task.TotalScore > 0 {
		return fmt.Sprintf("%.1f", *task.Score/task.TotalScore*100.0)
	}
	return "-"
}

func taskPctDisplayForTable(task models.TaskItem) string {
	if pct := taskPctDisplay(task); pct != "-" {
		return pct + "%"
	}
	return "-"
}

func taskStatusCode(task models.TaskItem) string {
	if task.FinishState != 0 && task.Score != nil {
		return "graded"
	}
	return "pending"
}

func subjectTypeLabel(subject gpa.Subject) string {
	switch subjectTypeCode(subject) {
	case "weighted_elective":
		return "Weighted Elective"
	case "weighted":
		return "Weighted"
	case "elective":
		return "Elective"
	default:
		return "Regular"
	}
}

func paddedColumns(widths []int, values []string) string {
	parts := make([]string, 0, len(values))
	for idx, value := range values {
		text := value
		if idx < len(widths) {
			if len(text) > widths[idx] {
				text = text[:widths[idx]]
			}
			text = text + strings.Repeat(" ", widths[idx]-len(text))
		}
		parts = append(parts, text)
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}

func logProgress(opts gpaCommandOptions, message string) {
	if !opts.suppressProgress() {
		printInfo(message)
	}
}

func nullableJSONFloat(value float64) *float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return &value
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
