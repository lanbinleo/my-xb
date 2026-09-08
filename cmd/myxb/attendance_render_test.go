package main

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"myxb/internal/attendance"
	"myxb/internal/models"
)

func TestRenderAttendanceViewShowsRateCountsAndSubjectTable(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}

	view := &attendance.YearView{
		Begin: time.Date(2026, 6, 23, 0, 0, 0, 0, location),
		End:   time.Date(2027, 2, 20, 23, 59, 59, 0, location),
		Year:  2026,
		Overall: models.AttendanceStatisticData{
			TotalCount:     47,
			StateCountList: []models.AttendanceStateCount{{AttendanceState: 0, Count: 39}, {AttendanceState: 3, Count: 8}},
			AttendanceRate: 82.98,
		},
		Subjects: []attendance.SubjectRow{
			{Name: "AP English Literature and Composition", Normal: 10, Absent: 4, Total: 14, RatePercent: 71.43},
			{Name: "AP Chemistry", Normal: 10, Absent: 2, Total: 12, RatePercent: 83.33},
			{Name: "Chinese Culture & Literature", Normal: 8, Total: 8, RatePercent: 100},
		},
	}

	rendered := stripANSICodes(renderAttendanceView(view))
	for _, want := range []string{
		"2026-06-23 ~ 2027-02-20",
		"School Year 2026-2027",
		"Attendance Rate: 82.98%",
		"Total: 47",
		"Normal: 39",
		"Absent: 8",
		"AP English Literature and Composition",
		"71.43%",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("renderAttendanceView output missing %q:\n%s", want, rendered)
		}
	}

	apEnglish := strings.Index(rendered, "AP English Literature")
	apChemistry := strings.Index(rendered, "AP Chemistry")
	if apEnglish < 0 || apChemistry < 0 || apEnglish > apChemistry {
		t.Fatalf("subjects should be sorted by absent count descending:\n%s", rendered)
	}
}

func stripANSICodes(text string) string {
	return ansiPattern.ReplaceAllString(text, "")
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestRenderAttendanceViewHandlesEmptySubjects(t *testing.T) {
	view := &attendance.YearView{
		Year:    2026,
		Overall: models.AttendanceStatisticData{},
	}

	rendered := renderAttendanceView(view)
	if !strings.Contains(rendered, "No subject attendance records found") {
		t.Fatalf("renderAttendanceView output = %q, want empty-records message", rendered)
	}
	if !strings.Contains(rendered, "0.00%") {
		t.Fatalf("renderAttendanceView output = %q, want zero rate line", rendered)
	}
}

func TestColorizeAttendanceRateThresholds(t *testing.T) {
	tests := []struct {
		rate float64
		want string
	}{
		{rate: 100, want: green("x")},
		{rate: 95, want: green("x")},
		{rate: 94.99, want: cyan("x")},
		{rate: 90, want: cyan("x")},
		{rate: 89.9, want: yellow("x")},
		{rate: 80, want: yellow("x")},
		{rate: 79.9, want: red("x")},
		{rate: 0, want: red("x")},
	}

	for _, tt := range tests {
		if got := colorizeAttendanceRate(tt.rate, "x"); got != tt.want {
			t.Fatalf("colorizeAttendanceRate(%v) = %q, want %q", tt.rate, got, tt.want)
		}
	}
}
