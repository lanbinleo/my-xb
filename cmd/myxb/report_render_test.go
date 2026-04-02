package main

import (
	"errors"
	"math"
	"strings"
	"testing"

	"myxb/internal/models"
	"myxb/pkg/gpa"
)

func TestMatchSemesterSelector(t *testing.T) {
	semesters := []models.Semester{
		{ID: 1, Year: 2024, Semester: 1},
		{ID: 2, Year: 2024, Semester: 2},
		{ID: 3, Year: 2025, Semester: 1, IsNow: true},
		{ID: 4, Year: 2025, Semester: 2},
	}

	tests := []struct {
		selector string
		wantIDs  []uint64
	}{
		{selector: "current", wantIDs: []uint64{3}},
		{selector: "1", wantIDs: []uint64{2}},
		{selector: "2025-1", wantIDs: []uint64{3}},
		{selector: "2025-2026", wantIDs: []uint64{3, 4}},
		{selector: "2025-2026-2", wantIDs: []uint64{4}},
	}

	for _, tt := range tests {
		got, err := matchSemesterSelector(semesters, tt.selector)
		if err != nil {
			t.Fatalf("matchSemesterSelector(%q) returned error: %v", tt.selector, err)
		}
		if len(got) != len(tt.wantIDs) {
			t.Fatalf("matchSemesterSelector(%q) len = %d, want %d", tt.selector, len(got), len(tt.wantIDs))
		}
		for idx := range tt.wantIDs {
			if got[idx].ID != tt.wantIDs[idx] {
				t.Fatalf("matchSemesterSelector(%q)[%d] = %d, want %d", tt.selector, idx, got[idx].ID, tt.wantIDs[idx])
			}
		}
	}
}

func TestResolveSemesterSelectionCleanModeDefaultsToCurrent(t *testing.T) {
	semesters := []models.Semester{
		{ID: 1, Year: 2024, Semester: 1},
		{ID: 2, Year: 2024, Semester: 2, IsNow: true},
		{ID: 3, Year: 2025, Semester: 1},
	}

	selected, err := resolveSemesterSelection(semesters, gpaCommandOptions{Clean: true})
	if err != nil {
		t.Fatalf("resolveSemesterSelection returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].ID != 2 {
		t.Fatalf("resolveSemesterSelection selected %+v, want semester ID 2", selected)
	}
}

func TestRenderJSONReportsAllowsNaNSummaryValues(t *testing.T) {
	reports := []semesterReport{
		{
			Semester: models.Semester{ID: 1, Year: 2025, Semester: 1},
			Result: gpa.CalculatedGPA{
				WeightedGPA:      math.NaN(),
				MaxGPA:           math.NaN(),
				UnweightedGPA:    math.NaN(),
				UnweightedMaxGPA: math.NaN(),
			},
		},
	}

	rendered, err := renderJSONReports(reports, gpaCommandOptions{Format: formatJSON})
	if err != nil {
		t.Fatalf("renderJSONReports returned error: %v", err)
	}
	if !strings.Contains(rendered, `"weighted_gpa": null`) {
		t.Fatalf("renderJSONReports output = %s, want weighted_gpa to be null", rendered)
	}
	if !strings.Contains(rendered, `"unweighted_max_gpa": null`) {
		t.Fatalf("renderJSONReports output = %s, want unweighted_max_gpa to be null", rendered)
	}
}

func TestRenderJSONReportsIncludesOfficialGPAError(t *testing.T) {
	reports := []semesterReport{
		{
			Semester:       models.Semester{ID: 1, Year: 2025, Semester: 1},
			OfficialGPAErr: errors.New("timeout"),
		},
	}

	rendered, err := renderJSONReports(reports, gpaCommandOptions{Format: formatJSON})
	if err != nil {
		t.Fatalf("renderJSONReports returned error: %v", err)
	}
	if !strings.Contains(rendered, `"official_gpa_error": "timeout"`) {
		t.Fatalf("renderJSONReports output = %s, want official_gpa_error to be included", rendered)
	}
}

func TestRenderTableReportsDoesNotInsertBlankLineBetweenSubjects(t *testing.T) {
	reports := []semesterReport{
		{
			Semester: models.Semester{Year: 2025, Semester: 1},
			Subjects: []gpa.Subject{
				{Name: "Math", Score: 95, GPA: 4.3},
				{Name: "English", Score: 93, GPA: 4.0},
			},
			Result: gpa.CalculatedGPA{
				WeightedGPA:      4.15,
				MaxGPA:           4.3,
				UnweightedGPA:    4.15,
				UnweightedMaxGPA: 4.3,
			},
		},
	}

	rendered := renderTableReports(reports, gpaCommandOptions{Format: formatTable})
	if strings.Contains(rendered, "Regular\n\nSubject:") {
		t.Fatalf("renderTableReports output = %s, want no blank line between subjects", rendered)
	}
}
