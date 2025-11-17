package main

import (
	"fmt"
	"math"
	"myxb/internal/api"
	"myxb/internal/models"
	"myxb/pkg/gpa"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	// MaxDisplaySemesters is the maximum number of semesters to display in the list
	MaxDisplaySemesters = 10

	// ElectiveCourseKeyword is used to identify elective courses
	ElectiveCourseKeyword = "Ele"
)

func calculateGPA(apiClient *api.API) {
	// Get semesters
	printInfo("Fetching semesters...")
	semesters, err := apiClient.GetSemesters()
	if err != nil {
		printError(fmt.Sprintf("Failed to get semesters: %v", err))
		os.Exit(1)
	}

	fmt.Println()

	// Display semester list (reversed, latest at bottom)
	var currentIndex int = -1
	displayCount := len(semesters)
	if displayCount > MaxDisplaySemesters {
		displayCount = MaxDisplaySemesters
	}

	// Display in reverse order
	for i := displayCount - 1; i >= 0; i-- {
		prefix := fmt.Sprintf("[%d]", i)
		suffix := ""
		if semesters[i].IsNow {
			suffix = cyan(" (current)")
			currentIndex = i
		}
		fmt.Printf("%s %d-%d Semester %d%s\n",
			gray(prefix),
			semesters[i].Year,
			semesters[i].Year+1,
			semesters[i].Semester,
			suffix)
	}

	fmt.Println()

	// Prompt user to select semester
	var selectedIndex int
	if currentIndex >= 0 {
		fmt.Printf("Select a semester to calculate GPA for (default %s): ", gray(fmt.Sprintf("[%d]", currentIndex)))
	} else {
		fmt.Print("Select a semester to calculate GPA for: ")
	}

	var input string
	fmt.Scanln(&input)

	if input == "" && currentIndex >= 0 {
		selectedIndex = currentIndex
	} else {
		_, err := fmt.Sscanf(input, "%d", &selectedIndex)
		if err != nil || selectedIndex < 0 || selectedIndex >= len(semesters) {
			printError("Invalid semester selection")
			os.Exit(1)
		}
	}

	selectedSemester := &semesters[selectedIndex]
	fmt.Println()
	printSuccess(fmt.Sprintf("Calculating GPA for %d-%d Semester %d",
		selectedSemester.Year, selectedSemester.Year+1, selectedSemester.Semester))
	fmt.Println()

	// Get subjects
	printInfo("Fetching subjects...")
	subjects, err := apiClient.GetSubjectList(selectedSemester.ID)
	if err != nil {
		printError(fmt.Sprintf("Failed to get subjects: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Found %d subjects", len(subjects)))
	fmt.Println()

	// Get semester dynamic scores to identify which subjects count toward GPA
	printInfo("Fetching semester-wide scores...")
	semesterScores, _ := apiClient.GetSemesterDynamicScore(selectedSemester.ID)

	// Create a map for quick lookup
	semesterScoreMap := make(map[uint64]*models.SubjectDynamicScore)
	for i := range semesterScores {
		semesterScoreMap[semesterScores[i].SubjectID] = &semesterScores[i]
	}

	// Process each subject
	calculatedSubjects := []gpa.Subject{}

	printInfo("Fetching scores for each subject...")
	fmt.Println()

	for _, subject := range subjects {
		// Get task list
		tasks, err := apiClient.GetTaskList(selectedSemester.ID, subject.ID)
		if err != nil || len(tasks) == 0 {
			continue
		}

		// Get task detail
		detail, err := apiClient.GetTaskDetail(tasks[0].ID)
		if err != nil {
			continue
		}

		// Get dynamic score detail
		dynamicScore, err := apiClient.GetDynamicScoreDetail(detail.ClassID, subject.ID, selectedSemester.ID)
		if err != nil {
			continue
		}

		// Get semester dynamic info for this subject
		dynamicInfo := semesterScoreMap[subject.ID]

		// Determine if elective
		isElective := strings.Contains(subject.Name, ElectiveCourseKeyword)

		// Process subject
		calcSubject := gpa.ProcessSubject(detail, dynamicScore, dynamicInfo, isElective)
		calculatedSubjects = append(calculatedSubjects, calcSubject)
	}

	// Print detailed tables for each subject
	for _, subject := range calculatedSubjects {
		printSubjectTable(subject)
	}

	// Calculate final GPA
	printInfo("Calculating final GPA...")
	result := gpa.CalculateGPA(calculatedSubjects)

	// Display results
	fmt.Println()
	fmt.Println("----------------------------------------------")

	if !math.IsNaN(result.WeightedGPA) {
		fmt.Printf("%s %.2f / %.2f %s\n",
			bold("Weighted GPA:"),
			result.WeightedGPA,
			result.MaxGPA,
			gray(fmt.Sprintf("(%.1f%%)", result.WeightedGPA/result.MaxGPA*100)))

		fmt.Printf("%s %.2f / %.2f %s\n",
			bold("Unweighted GPA:"),
			result.UnweightedGPA,
			result.UnweightedMaxGPA,
			gray(fmt.Sprintf("(%.1f%%)", result.UnweightedGPA/result.UnweightedMaxGPA*100)))

		fmt.Println()

		// Get official GPA for comparison
		officialGPA, err := apiClient.GetGPA(selectedSemester.ID)
		if err == nil && officialGPA != nil {
			diff := result.WeightedGPA - *officialGPA
			// Use a small tolerance (0.01) to account for floating-point precision
			if math.Abs(diff) > 0.01 {
				sign := "+"
				color := red
				if diff < 0 {
					sign = ""
					color = green
				}
				diffStr := color(fmt.Sprintf("(%s%.2f)", sign, diff))
				fmt.Printf("%s %.2f %s\n",
					bold("Official GPA:"),
					*officialGPA,
					diffStr)

				// Please Report This to Developers
				fmt.Printf("\n%s%.2f%s\n%s\n%s\n%s\n",
					bold("Hi! We found a discrepancy of "),
					diff,
					bold(" points in the GPA calculation."),
					"This may be caused by special courses that are "+yellow("weighted differently or excluded")+" from official GPA calculation.",
					"Please report this to the developers so we can improve the accuracy.",
					"Thank you!")
			} else {
				fmt.Printf("%s %.2f\n",
					bold("Official GPA:"),
					*officialGPA)
			}
		} else if err == nil {
			fmt.Println(gray("Official GPA not yet published"))
		}

		fmt.Println()
		printSuccess(fmt.Sprintf("Calculated GPA from %d subjects", len(result.Subjects)))
	} else {
		printWarning("Unable to calculate GPA - no valid subjects found")
	}
}

// printSubjectTable prints a detailed table for a subject
func printSubjectTable(subject gpa.Subject) {
	if math.IsNaN(subject.Score) {
		return
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	// Add subject header row
	scoreStr := fmt.Sprintf("%.1f", subject.Score)
	if subject.ExtraCredit > 0.0 {
		scoreStr += fmt.Sprintf(" (%.2f Extra credit)", subject.ExtraCredit)
	}

	typeStr := "Regular"
	if subject.IsWeighted {
		typeStr = "Weighted"
	}
	if subject.IsElective {
		typeStr += " Elective"
	}

	scoreLevel := getScoreLevel(subject)
	t.AppendRow(table.Row{
		colorizeByScoreLevel(subject.Name, scoreLevel),
		bold(scoreStr),
		bold(scoreLevel),
		bold(fmt.Sprintf("%.2f", subject.GPA)),
		bold(typeStr),
	})

	// Add 分割线
	t.AppendSeparator()

	// Add evaluation projects
	for _, evalProject := range subject.EvaluationDetails {
		if evalProject.ScoreIsNull {
			continue
		}

		addEvaluationProjectRows(t, &evalProject, "", true, subject.IsWeighted)
	}

	fmt.Println(t.Render())
	fmt.Println()
}

// addEvaluationProjectRows recursively adds evaluation project rows to the table
func addEvaluationProjectRows(t table.Writer, evalProject *models.EvaluationProject, indent string, showTasks bool, isWeighted bool) {
	// Add the evaluation project row
	name := indent + evalProject.EvaluationProjectEName
	proportionStr := fmt.Sprintf("%.2f%%", evalProject.Proportion)

	t.AppendRow(table.Row{
		colorizeByScoreLevel(name, evalProject.ScoreLevel),
		fmt.Sprintf("%.1f", evalProject.Score),
		evalProject.ScoreLevel,
		fmt.Sprintf("%.2f", evalProject.GPA),
		proportionStr,
	})

	// Add learning tasks if showTasks is true
	if showTasks && len(evalProject.LearningTaskAndExamList) > 0 {
		tasksWithScores := []models.LearningTask{}
		for _, task := range evalProject.LearningTaskAndExamList {
			if task.Score != nil {
				tasksWithScores = append(tasksWithScores, task)
			}
		}

		if len(tasksWithScores) > 0 {
			weight := evalProject.Proportion / float64(len(tasksWithScores))
			for _, task := range tasksWithScores {
				score := *task.Score / task.TotalScore * 100.0
				scoreLevel := gpa.GetScoreLevelFromScore(score, isWeighted)

				t.AppendRow(table.Row{
					colorizeByScoreLevel(indent+"- "+task.Name, scoreLevel),
					fmt.Sprintf("%.0f / %.0f", *task.Score, task.TotalScore),
					fmt.Sprintf("%.2f%%", score),
					"",
					indent + fmt.Sprintf("- %.2f%%", weight),
				})
			}
		}
	}

	// Recursively add nested evaluation projects
	for _, nestedProject := range evalProject.EvaluationProjectList {
		if nestedProject.ScoreIsNull {
			continue
		}
		addEvaluationProjectRows(t, &nestedProject, indent+"- ", showTasks, isWeighted)
	}
}

// getScoreLevel gets the score level from a subject's evaluation projects
func getScoreLevel(subject gpa.Subject) string {
	// Use the subject's score and weighted status to get the accurate level
	return gpa.GetScoreLevelFromScore(subject.Score, subject.IsWeighted)
}
