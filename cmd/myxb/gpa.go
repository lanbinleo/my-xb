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

		addEvaluationProjectRows(t, &evalProject, "", false, subject.IsWeighted)
	}

	// Add all tasks section if showTasks is true
	if showTasks && len(tasks) > 0 {
		// t.AppendSeparator()
		// 加一个空行
		t.AppendRow(table.Row{
			"",
			"",
			"",
			"",
			"",
		})

		// Add "All Tasks" header
		t.AppendRow(table.Row{
			bold("All Tasks"),
			"Score",
			"Pct",
			"Status",
			"Weight",
		})

		// Display all tasks in one loop
		for _, task := range tasks {
			if task.FinishState != 0 && task.Score != nil {
				// Task with score
				score := *task.Score / task.TotalScore * 100.0
				scoreLevel := gpa.GetScoreLevelFromScore(score, subject.IsWeighted)
				status := green("出分")

				t.AppendRow(table.Row{
					colorizeByScoreLevel("  - "+asciiDisplayText(task.Name), scoreLevel),
					fmt.Sprintf("%.0f / %.0f", *task.Score, task.TotalScore),
					fmt.Sprintf("%.1f%%", score),
					status,
					"",
				})
			} else {
				// Task without score
				t.AppendRow(table.Row{
					gray("  - " + asciiDisplayText(task.Name)),
					gray(fmt.Sprintf("- / %.0f", task.TotalScore)),
					gray("-"),
					yellow("未出分"),
					gray("-"),
				})
			}
		}
	}

	return t.Render()
}

// addEvaluationProjectRows recursively adds evaluation project rows to the table
func addEvaluationProjectRows(t table.Writer, evalProject *models.EvaluationProject, indent string, showTasks bool, isWeighted bool) {
	// Add the evaluation project row
	name := indent + asciiDisplayText(evalProject.EvaluationProjectEName)
	proportionStr := fmt.Sprintf("%.2f%%", evalProject.Proportion)

	t.AppendRow(table.Row{
		colorizeByScoreLevel(name, evalProject.ScoreLevel),
		fmt.Sprintf("%.1f", evalProject.Score),
		evalProject.ScoreLevel,
		fmt.Sprintf("%.2f", evalProject.GPA),
		proportionStr,
	})

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
