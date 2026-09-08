package attendance

import (
	"fmt"
	"myxb/internal/models"
	"testing"
)

func TestGetYearViewUsesCurrentSemesterOwnRange(t *testing.T) {
	provider := &fakeProvider{
		semesters: []models.Semester{
			{ID: 1, Year: 2025, Semester: 1, StartDate: "2025-06-23T00:00:00+08:00", EndDate: "2026-02-20T00:00:00+08:00"},
			// The isNow "semester=1" entry already spans the whole school year;
			// "semester=2" is the following Feb-July segment and must be ignored.
			{ID: 2, Year: 2026, Semester: 1, IsNow: true, StartDate: "2026-06-23T00:00:00+08:00", EndDate: "2027-02-20T00:00:00+08:00"},
			{ID: 3, Year: 2026, Semester: 2, StartDate: "2027-02-21T00:00:00+08:00", EndDate: "2027-07-31T00:00:00+08:00"},
		},
	}

	service := NewService(provider)
	view, err := service.GetYearView()
	if err != nil {
		t.Fatalf("GetYearView returned error: %v", err)
	}

	if view.Year != 2026 {
		t.Fatalf("year = %d, want 2026", view.Year)
	}
	if view.Begin.Format(statisticTimeLayout) != "2026-06-23 00:00:00" {
		t.Fatalf("begin = %s, want 2026-06-23 00:00:00", view.Begin.Format(statisticTimeLayout))
	}
	if view.End.Format(statisticTimeLayout) != "2027-02-20 23:59:59" {
		t.Fatalf("end = %s, want 2027-02-20 23:59:59 (not the semester-2 tail)", view.End.Format(statisticTimeLayout))
	}
	if provider.overallQueryBegin != "2026-06-23 00:00:00" || provider.overallQueryEnd != "2027-02-20 23:59:59" {
		t.Fatalf("query range = %q ~ %q, want the isNow semester range", provider.overallQueryBegin, provider.overallQueryEnd)
	}
}

func TestGetYearViewComputesSubjectRowsAndSortsByAbsence(t *testing.T) {
	provider := &fakeProvider{
		semesters: []models.Semester{
			{ID: 3, Year: 2026, Semester: 2, IsNow: true, StartDate: "2026-06-23T00:00:00+08:00", EndDate: "2027-02-20T00:00:00+08:00"},
		},
		overall: models.AttendanceStatisticData{
			TotalCount:     47,
			StateCountList: []models.AttendanceStateCount{{AttendanceState: 0, Count: 39}, {AttendanceState: 3, Count: 8}},
			AttendanceRate: 82.98,
		},
		subjects: []models.SubjectAttendanceStat{
			{SubjectID: 1, SubjectName: "AP Chemistry", NormalCount: 10, AbsentCount: 2},
			{SubjectID: 2, SubjectName: "AP English Literature and Composition", NormalCount: 10, AbsentCount: 4},
			{SubjectID: 3, SubjectEName: "Chinese Culture & Literature", NormalCount: 8},
		},
	}

	service := NewService(provider)
	view, err := service.GetYearView()
	if err != nil {
		t.Fatalf("GetYearView returned error: %v", err)
	}

	if view.Overall.AttendanceRate != 82.98 {
		t.Fatalf("overall rate = %v, want 82.98", view.Overall.AttendanceRate)
	}
	if len(view.Subjects) != 3 {
		t.Fatalf("subject count = %d, want 3", len(view.Subjects))
	}

	top := view.Subjects[0]
	if top.Name != "AP English Literature and Composition" {
		t.Fatalf("first subject = %q, want the one with most absences", top.Name)
	}
	if top.Total != 14 || top.Normal != 10 || top.Absent != 4 {
		t.Fatalf("top row counts = %+v, want total 14 normal 10 absent 4", top)
	}
	if rate := fmt.Sprintf("%.2f", top.RatePercent); rate != "71.43" {
		t.Fatalf("top rate = %s, want 71.43", rate)
	}

	if view.Subjects[2].Name != "Chinese Culture & Literature" {
		t.Fatalf("EName fallback failed: %+v", view.Subjects[2])
	}
	if view.Subjects[2].Total != 8 || view.Subjects[2].RatePercent != 100 {
		t.Fatalf("full-attendance row = %+v, want total 8 rate 100", view.Subjects[2])
	}
}

func TestBuildSubjectRowGuardsZeroTotal(t *testing.T) {
	row := buildSubjectRow(models.SubjectAttendanceStat{SubjectName: "Empty"})
	if row.Total != 0 || row.RatePercent != 0 {
		t.Fatalf("empty row = %+v, want zero total and zero rate", row)
	}
}

type fakeProvider struct {
	semesters         []models.Semester
	overall           models.AttendanceStatisticData
	subjects          []models.SubjectAttendanceStat
	overallQueryBegin string
	overallQueryEnd   string
}

func (f *fakeProvider) GetSemesters() ([]models.Semester, error) {
	return f.semesters, nil
}

func (f *fakeProvider) GetAttendanceStatistic(beginTime, endTime string) (*models.AttendanceStatisticData, error) {
	f.overallQueryBegin = beginTime
	f.overallQueryEnd = endTime
	overall := f.overall
	return &overall, nil
}

func (f *fakeProvider) GetSubjectAttendanceStatistic(beginTime, endTime string) ([]models.SubjectAttendanceStat, error) {
	return f.subjects, nil
}
