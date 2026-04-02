package gpa

import (
	"math"
	"testing"

	"myxb/internal/models"
)

func TestResolveSubjectWeightSupportsFractionalCredits(t *testing.T) {
	tests := []struct {
		name         string
		subjectName  string
		electiveHint bool
		want         float64
	}{
		{name: "regular course", subjectName: "Physical Education", want: 1.0},
		{name: "elective hint", subjectName: "Creative Writing", electiveHint: true, want: 0.5},
		{name: "configured half weighted humanities", subjectName: "C-Humanities", want: 0.5},
		{name: "configured one third", subjectName: "Chinese History", want: 1.0 / 3.0},
		{name: "configured two thirds", subjectName: "Chinese II", want: 2.0 / 3.0},
		{name: "configured half weighted spanish", subjectName: "Spanish II", want: 0.5},
		{name: "weighted language keeps full credit", subjectName: "AP Spanish Language and Culture", want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSubjectWeight(tt.subjectName, tt.electiveHint)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("ResolveSubjectWeight(%q, %t) = %.12f, want %.12f", tt.subjectName, tt.electiveHint, got, tt.want)
			}
		})
	}
}

func TestCalculateGPAMatchesOfficialWithHalfCreditSpanish(t *testing.T) {
	subjects := []Subject{
		{Name: "AP English Language and Composition", GPA: 4.2, UnweightedGPA: 3.7, MaxGPA: 4.8, Weight: 1.0, IsWeighted: true, IsInGrade: true},
		{Name: "AP Statistics", GPA: 1.5, UnweightedGPA: 1.0, MaxGPA: 4.8, Weight: 1.0, IsWeighted: true, IsInGrade: true},
		{Name: "Spanish II", GPA: 4.0, UnweightedGPA: 4.0, MaxGPA: 4.3, Weight: ResolveSubjectWeight("Spanish II", false), IsInGrade: true},
		{Name: "Physical Education", GPA: 4.3, UnweightedGPA: 4.3, MaxGPA: 4.3, Weight: 1.0, IsInGrade: true},
		{Name: "AP Physics C", GPA: 2.8, UnweightedGPA: 2.3, MaxGPA: 4.8, Weight: 1.0, IsWeighted: true, IsInGrade: true},
		{Name: "AP Psychology", GPA: 4.8, UnweightedGPA: 4.3, MaxGPA: 4.8, Weight: 1.0, IsWeighted: true, IsInGrade: true},
	}

	got := CalculateGPA(subjects)

	if math.Abs(got.WeightedGPA-3.5636363636) > 1e-9 {
		t.Fatalf("CalculateGPA weighted = %.10f, want %.10f", got.WeightedGPA, 3.5636363636)
	}
	if rounded := math.Round(got.WeightedGPA*100) / 100; rounded != 3.56 {
		t.Fatalf("CalculateGPA rounded weighted = %.2f, want 3.56", rounded)
	}
}

func TestProcessSubjectKeepsElectiveFlagSeparateFromCreditWeight(t *testing.T) {
	tests := []struct {
		name         string
		subjectName  string
		electiveHint bool
		wantWeight   float64
		wantElective bool
	}{
		{name: "configured half credit course is not forced to elective", subjectName: "Spanish II", wantWeight: 0.5, wantElective: false},
		{name: "hinted elective stays elective", subjectName: "Creative Writing", electiveHint: true, wantWeight: 0.5, wantElective: true},
		{name: "fractional credit regular course stays non elective", subjectName: "Chinese History", wantWeight: 1.0 / 3.0, wantElective: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := ProcessSubject(
				&models.SubjectDetail{
					SubjectID:   1,
					SubjectName: tt.subjectName,
					ClassID:     2,
				},
				&models.DynamicScoreData{
					EvaluationProjectList: []models.EvaluationProject{
						{EvaluationProjectEName: "Total", Proportion: 100, Score: 90},
					},
				},
				nil,
				tt.electiveHint,
			)

			if math.Abs(subject.Weight-tt.wantWeight) > 1e-9 {
				t.Fatalf("ProcessSubject(%q).Weight = %.12f, want %.12f", tt.subjectName, subject.Weight, tt.wantWeight)
			}
			if subject.IsElective != tt.wantElective {
				t.Fatalf("ProcessSubject(%q).IsElective = %t, want %t", tt.subjectName, subject.IsElective, tt.wantElective)
			}
		})
	}
}
