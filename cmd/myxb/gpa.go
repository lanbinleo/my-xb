package main

import (
	"fmt"
	"math"
	"myxb/internal/models"
	"myxb/pkg/gpa"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	// MaxDisplaySemesters is the maximum number of semesters to display in the list
	MaxDisplaySemesters = 10

	// ElectiveCourseKeyword is used to identify elective courses
	ElectiveCourseKeyword = "Ele"
)

// renderSubjectTable renders a detailed table for a subject.
func renderSubjectTable(subject gpa.Subject, showTasks bool, tasks []models.TaskItem) string {
	if math.IsNaN(subject.Score) {
		return ""
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
		colorizeByScoreLevel(asciiDisplayText(subject.Name), scoreLevel),
		emphasizeByScoreLevel(scoreStr, scoreLevel, true),
		emphasizeByScoreLevel(scoreLevel, scoreLevel, true),
		emphasizeByScoreLevel(fmt.Sprintf("%.2f", subject.GPA), scoreLevel, true),
		bold(typeStr),
	})

	// Add 分割线
	t.AppendSeparator()

	tasksByCategory := map[uint64][]models.TaskItem{}
	knownCategoryIDs := map[uint64]bool{}
	if showTasks {
		tasksByCategory = groupTasksByCategory(tasks)
		knownCategoryIDs = evaluationProjectIDs(subject.EvaluationDetails)
	}

	// Add evaluation projects
	for _, evalProject := range subject.EvaluationDetails {
		if !evaluationProjectVisible(evalProject, tasksByCategory) {
			continue
		}

		addEvaluationProjectRows(t, &evalProject, "", showTasks, subject.IsWeighted, tasksByCategory)
	}

	if showTasks {
		addUncategorizedTaskRows(t, tasksWithoutKnownCategory(tasks, knownCategoryIDs), subject.IsWeighted)
	}

	return t.Render()
}

// addEvaluationProjectRows recursively adds evaluation project rows to the table
func addEvaluationProjectRows(t table.Writer, evalProject *models.EvaluationProject, indent string, showTasks bool, isWeighted bool, tasksByCategory map[uint64][]models.TaskItem) {
	// Add the evaluation project row
	name := indent + asciiDisplayText(evalProject.EvaluationProjectEName)
	proportionStr := fmt.Sprintf("%.2f%%", evalProject.Proportion)
	score := fmt.Sprintf("%.1f", evalProject.Score)
	level := evalProject.ScoreLevel
	projectGPA := fmt.Sprintf("%.2f", evalProject.GPA)
	if evalProject.ScoreIsNull {
		name = gray(name)
		score = gray("--")
		level = gray("-")
		projectGPA = gray("-")
		proportionStr = gray(proportionStr)
	}

	t.AppendRow(table.Row{
		colorizeByScoreLevel(name, evalProject.ScoreLevel),
		emphasizeByScoreLevel(score, evalProject.ScoreLevel, false),
		emphasizeByScoreLevel(level, evalProject.ScoreLevel, false),
		emphasizeByScoreLevel(projectGPA, evalProject.ScoreLevel, false),
		proportionStr,
	})

	// Recursively add nested evaluation projects
	for _, nestedProject := range evalProject.EvaluationProjectList {
		if !evaluationProjectVisible(nestedProject, tasksByCategory) {
			continue
		}
		addEvaluationProjectRows(t, &nestedProject, indent+"- ", showTasks, isWeighted, tasksByCategory)
	}

	if showTasks {
		addTaskRows(t, tasksByCategory[evalProject.EvaluationProjectID], indent+"  ", isWeighted)
	}
}

func evaluationProjectVisible(project models.EvaluationProject, tasksByCategory map[uint64][]models.TaskItem) bool {
	if !project.ScoreIsNull || len(tasksByCategory[project.EvaluationProjectID]) > 0 {
		return true
	}
	for _, child := range project.EvaluationProjectList {
		if evaluationProjectVisible(child, tasksByCategory) {
			return true
		}
	}
	return false
}

func addTaskRows(t table.Writer, tasks []models.TaskItem, indent string, isWeighted bool) {
	for _, task := range tasks {
		if taskHasScore(task) && task.TotalScore > 0 {
			score := *task.Score / task.TotalScore * 100.0
			scoreLevel := gpa.GetScoreLevelFromScore(score, isWeighted)
			t.AppendRow(table.Row{
				colorizeByScoreLevel(indent+"- "+asciiDisplayText(task.Name), scoreLevel),
				emphasizeByScoreLevel(fmt.Sprintf("%.0f / %.0f", *task.Score, task.TotalScore), scoreLevel, false),
				emphasizeByScoreLevel(fmt.Sprintf("%.1f%%", score), scoreLevel, false),
				green("出分"),
				taskEstimatedWeightDisplay(task),
			})
			continue
		}

		t.AppendRow(table.Row{
			gray(indent + "- " + asciiDisplayText(task.Name)),
			gray(fmt.Sprintf("- / %.0f", task.TotalScore)),
			gray("-"),
			yellow("未出分"),
			gray(taskEstimatedWeightDisplay(task)),
		})
	}
}

func addUncategorizedTaskRows(t table.Writer, tasks []models.TaskItem, isWeighted bool) {
	if len(tasks) == 0 {
		return
	}

	t.AppendRow(table.Row{
		bold("Uncategorized Tasks"),
		"Score",
		"Pct",
		"Status",
		"Weight",
	})
	addTaskRows(t, tasks, "  ", isWeighted)
}

// getScoreLevel gets the score level from a subject's evaluation projects
func getScoreLevel(subject gpa.Subject) string {
	// Use the subject's score and weighted status to get the accurate level
	return gpa.GetScoreLevelFromScore(subject.Score, subject.IsWeighted)
}

func emphasizeByScoreLevel(text, scoreLevel string, alwaysBold bool) string {
	if alwaysBold || scoreLevel == "A+" {
		return bold(text)
	}
	return text
}

func groupTasksByCategory(tasks []models.TaskItem) map[uint64][]models.TaskItem {
	grouped := make(map[uint64][]models.TaskItem)
	for _, task := range tasks {
		if task.CategoryID == 0 {
			continue
		}
		grouped[task.CategoryID] = append(grouped[task.CategoryID], task)
	}
	return grouped
}

func evaluationProjectIDs(projects []models.EvaluationProject) map[uint64]bool {
	ids := make(map[uint64]bool)
	var walk func([]models.EvaluationProject)
	walk = func(items []models.EvaluationProject) {
		for _, project := range items {
			if project.EvaluationProjectID != 0 {
				ids[project.EvaluationProjectID] = true
			}
			walk(project.EvaluationProjectList)
		}
	}
	walk(projects)
	return ids
}

func tasksWithoutKnownCategory(tasks []models.TaskItem, knownCategoryIDs map[uint64]bool) []models.TaskItem {
	ungrouped := make([]models.TaskItem, 0)
	for _, task := range tasks {
		if task.CategoryID == 0 || !knownCategoryIDs[task.CategoryID] {
			ungrouped = append(ungrouped, task)
		}
	}
	return ungrouped
}
