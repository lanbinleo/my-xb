package main

import (
	"math"
	"testing"

	"myxb/internal/models"
)

func TestAttachTaskDetailMetadataAddsCategoryAndEstimatedWeight(t *testing.T) {
	scoreA := 90.0
	scoreB := 80.0
	inSubjectScore := true
	tasks := []models.TaskItem{
		{ID: 1, Name: "Quiz 1", Score: &scoreA, TotalScore: 100, FinishState: 1},
		{ID: 2, Name: "Quiz 2", Score: &scoreB, TotalScore: 100, FinishState: 1},
	}
	details := map[uint64]*models.SubjectDetail{
		1: {IsInSubjectScore: true, EvaProjects: []models.TaskEvaluationProject{{ID: 10, EName: "Continuous Assessments"}}},
		2: {IsInSubjectScore: true, EvaProjects: []models.TaskEvaluationProject{{ID: 10, EName: "Continuous Assessments"}}},
	}
	projects := []models.EvaluationProject{
		{EvaluationProjectID: 10, EvaluationProjectEName: "Continuous Assessments", Proportion: 40, Score: 85},
	}

	got := attachTaskDetailMetadata(tasks, details, projects)

	if got[0].CategoryID != 10 || got[0].CategoryEName != "Continuous Assessments" {
		t.Fatalf("task category = %d/%q, want 10/Continuous Assessments", got[0].CategoryID, got[0].CategoryEName)
	}
	if got[0].IsInSubjectScore == nil || *got[0].IsInSubjectScore != inSubjectScore {
		t.Fatalf("IsInSubjectScore = %v, want true", got[0].IsInSubjectScore)
	}
	if got[0].EstimatedSubjectWeight == nil {
		t.Fatalf("EstimatedSubjectWeight is nil, want inferred weight")
	}
	if math.Abs(*got[0].EstimatedSubjectWeight-20) > 1e-9 {
		t.Fatalf("EstimatedSubjectWeight = %.4f, want 20", *got[0].EstimatedSubjectWeight)
	}
}

func TestAttachTaskDetailMetadataSkipsWeightWhenCategoryScoreDoesNotMatchTasks(t *testing.T) {
	score := 50.0
	tasks := []models.TaskItem{
		{ID: 1, Name: "Task", Score: &score, TotalScore: 100, FinishState: 1},
	}
	details := map[uint64]*models.SubjectDetail{
		1: {IsInSubjectScore: true, EvaProjects: []models.TaskEvaluationProject{{ID: 10, EName: "Project"}}},
	}
	projects := []models.EvaluationProject{
		{EvaluationProjectID: 10, EvaluationProjectEName: "Project", Proportion: 100, Score: 90},
	}

	got := attachTaskDetailMetadata(tasks, details, projects)

	if got[0].CategoryID != 10 {
		t.Fatalf("CategoryID = %d, want 10", got[0].CategoryID)
	}
	if got[0].EstimatedSubjectWeight != nil {
		t.Fatalf("EstimatedSubjectWeight = %.4f, want nil when score inference is unsafe", *got[0].EstimatedSubjectWeight)
	}
}
