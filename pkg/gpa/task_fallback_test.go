package gpa

import (
	"math"
	"myxb/internal/models"
	"testing"
)

func floatPtr(value float64) *float64 { return &value }

// scoredTask builds a task list entry the way GetList returns it for a graded,
// mobile-synced task.
func scoredTask(id uint64, score, totalScore float64) models.TaskItem {
	return models.TaskItem{
		ID:                 id,
		Score:              floatPtr(score),
		TotalScore:         totalScore,
		FinishState:        1,
		IsSynchroToMobiled: 1,
	}
}

// detailWithPath builds a task detail whose evaProjects path is leaf-first,
// matching the real GetDetail response order.
func detailWithPath(isCounted bool, path ...models.TaskEvaluationProject) *models.SubjectDetail {
	return &models.SubjectDetail{
		IsInSubjectScore: isCounted,
		EvaProjects:      path,
	}
}

func apChemistryFixture() ([]models.TaskItem, map[uint64]*models.SubjectDetail) {
	comprehensive := models.TaskEvaluationProject{ID: 30468, Name: "综合性评估", EName: "Comprehensive Assessments", Proportion: 40, ProPath: "30468,", ParentProID: 0}
	continuous := models.TaskEvaluationProject{ID: 30469, Name: "连续性评估", EName: "Continuous Assessments", Proportion: 40, ProPath: "30469,", ParentProID: 0}
	homework := models.TaskEvaluationProject{ID: 2409, Name: "Homework", EName: "Homework", Proportion: 33, ProPath: "30469,2409,", ParentProID: 30469}
	classroom := models.TaskEvaluationProject{ID: 47616, Name: "课堂评估", EName: "Classroom-Based Assessments", Proportion: 67, ProPath: "30469,47616,", ParentProID: 30469}

	tasks := []models.TaskItem{
		scoredTask(1, 90, 100), // homework
		scoredTask(2, 80, 100), // homework
		scoredTask(3, 100, 100),
		{ID: 4, Score: floatPtr(0), TotalScore: 100, FinishState: 1, IsSynchroToMobiled: 0},  // graded-looking but not synced
		{ID: 5, Score: floatPtr(70), TotalScore: 100, FinishState: 1, IsSynchroToMobiled: 1}, // not counted in subject score
	}
	tasks[3].Name = "Unit 1 Test"

	details := map[uint64]*models.SubjectDetail{
		1: detailWithPath(true, homework, continuous),
		2: detailWithPath(true, homework, continuous),
		3: detailWithPath(true, classroom, continuous),
		4: detailWithPath(true, comprehensive),
		5: detailWithPath(false, comprehensive),
	}
	return tasks, details
}

func findProject(t *testing.T, projects []models.EvaluationProject, id uint64) *models.EvaluationProject {
	t.Helper()
	for idx := range projects {
		if projects[idx].EvaluationProjectID == id {
			return &projects[idx]
		}
		nested := findProject(t, projects[idx].EvaluationProjectList, id)
		if nested != nil {
			return nested
		}
	}
	return nil
}

func TestBuildEvaluationFromTasksNestedCategories(t *testing.T) {
	tasks, details := apChemistryFixture()

	projects := BuildEvaluationFromTasks(tasks, details, true)

	if len(projects) != 2 {
		t.Fatalf("expected 2 top-level categories, got %d", len(projects))
	}

	homework := findProject(t, projects, 2409)
	if homework == nil {
		t.Fatal("homework category missing")
	}
	if homework.ScoreIsNull || homework.Score != 85 {
		t.Errorf("homework score = %.2f (null=%v), want 85", homework.Score, homework.ScoreIsNull)
	}
	if homework.ScoreLevel != "B" || homework.GPA != 3.5 {
		t.Errorf("homework level/GPA = %s/%.2f, want B/3.50", homework.ScoreLevel, homework.GPA)
	}

	classroom := findProject(t, projects, 47616)
	if classroom == nil || classroom.ScoreIsNull || classroom.Score != 100 {
		t.Errorf("classroom score = %.2f, want 100", classroom.Score)
	}

	continuous := findProject(t, projects, 30469)
	if continuous == nil {
		t.Fatal("continuous category missing")
	}
	wantContinuous := (85.0*33 + 100.0*67) / 100.0
	if continuous.ScoreIsNull || math.Abs(continuous.Score-wantContinuous) > 1e-9 {
		t.Errorf("continuous score = %.4f, want %.4f", continuous.Score, wantContinuous)
	}

	comprehensive := findProject(t, projects, 30468)
	if comprehensive == nil || !comprehensive.ScoreIsNull {
		t.Errorf("comprehensive should stay score-null (unsynced task only), got %+v", comprehensive)
	}

	// The whole-subject pipeline should ignore the null category and score the
	// subject from Continuous Assessments alone.
	AdjustProportions(projects)
	subjectScore := CalculateSubjectScore(projects)
	if math.IsNaN(subjectScore) || math.Abs(subjectScore-wantContinuous) > 1e-9 {
		t.Errorf("subject score = %.4f, want %.4f", subjectScore, wantContinuous)
	}
}

func TestBuildEvaluationFromTasksSingleRootCategory(t *testing.T) {
	category := models.TaskEvaluationProject{ID: 30468, EName: "Comprehensive Assessments", Proportion: 100, ProPath: "30468,"}
	tasks := []models.TaskItem{scoredTask(1, 80, 100), scoredTask(2, 90, 100)}
	details := map[uint64]*models.SubjectDetail{
		1: detailWithPath(true, category),
		2: detailWithPath(true, category),
	}

	projects := BuildEvaluationFromTasks(tasks, details, false)

	if len(projects) != 1 {
		t.Fatalf("expected 1 category, got %d", len(projects))
	}
	if projects[0].ScoreIsNull || projects[0].Score != 85 {
		t.Errorf("category score = %.2f, want 85", projects[0].Score)
	}
	if projects[0].ScoreLevel != "B" {
		t.Errorf("non-weighted level = %s, want B", projects[0].ScoreLevel)
	}
}

func TestBuildEvaluationFromTasksAllUnsynced(t *testing.T) {
	category := models.TaskEvaluationProject{ID: 30468, EName: "Comprehensive Assessments", Proportion: 40, ProPath: "30468,"}
	tasks := []models.TaskItem{{ID: 1, Score: floatPtr(0), TotalScore: 100, FinishState: 1, IsSynchroToMobiled: 0}}
	details := map[uint64]*models.SubjectDetail{1: detailWithPath(true, category)}

	projects := BuildEvaluationFromTasks(tasks, details, true)

	if len(projects) != 1 || !projects[0].ScoreIsNull {
		t.Fatalf("expected one score-null category, got %+v", projects)
	}
	if !math.IsNaN(CalculateSubjectScore(projects)) {
		t.Error("expected NaN subject score when no task is synced")
	}
}

func TestBuildEvaluationFromTasksIgnoresMissingDetails(t *testing.T) {
	tasks, _ := apChemistryFixture()
	projects := BuildEvaluationFromTasks(tasks, nil, true)

	// Tasks without details neither contribute scores nor register categories.
	if len(projects) != 0 {
		t.Fatalf("expected no categories without details, got %d", len(projects))
	}
}

func TestTaskIsScored(t *testing.T) {
	cases := []struct {
		name string
		task models.TaskItem
		want bool
	}{
		{"synced graded", scoredTask(1, 90, 100), true},
		{"not synced", models.TaskItem{Score: floatPtr(0), TotalScore: 100, FinishState: 1, IsSynchroToMobiled: 0}, false},
		{"not finished", models.TaskItem{Score: floatPtr(90), TotalScore: 100, FinishState: 0, IsSynchroToMobiled: 1}, false},
		{"null score", models.TaskItem{TotalScore: 100, FinishState: 1, IsSynchroToMobiled: 1}, false},
		{"zero total", models.TaskItem{Score: floatPtr(90), TotalScore: 0, FinishState: 1, IsSynchroToMobiled: 1}, false},
	}
	for _, tc := range cases {
		if got := taskIsScored(tc.task); got != tc.want {
			t.Errorf("%s: taskIsScored = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestProPathDepth(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"", 0},
		{"30468,", 1},
		{"30469,2409,", 2},
		{"30469, 2409, 7,", 3},
	}
	for _, tc := range cases {
		if got := proPathDepth(tc.path); got != tc.want {
			t.Errorf("proPathDepth(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}
