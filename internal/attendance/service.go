// Package attendance builds attendance-rate views from Xiaobao.
package attendance

import (
	"fmt"
	"myxb/internal/models"
	"sort"
	"strings"
	"time"
)

const statisticTimeLayout = "2006-01-02 15:04:05"

// Provider fetches semester metadata and attendance statistics from Xiaobao.
type Provider interface {
	GetSemesters() ([]models.Semester, error)
	GetAttendanceStatistic(beginTime, endTime string) (*models.AttendanceStatisticData, error)
	GetSubjectAttendanceStatistic(beginTime, endTime string) ([]models.SubjectAttendanceStat, error)
}

// SubjectRow is a rendered-ready per-subject attendance summary.
type SubjectRow struct {
	Name        string
	Normal      int
	Late        int
	EarlyLeave  int
	Absent      int
	Total       int
	RatePercent float64
}

// YearView is the attendance summary for one school year.
type YearView struct {
	Begin    time.Time
	End      time.Time
	Year     uint64
	Overall  models.AttendanceStatisticData
	Subjects []SubjectRow
}

// Service resolves the current school year and queries attendance stats.
type Service struct {
	provider Provider
	location *time.Location
}

// NewService creates an attendance service.
func NewService(provider Provider) *Service {
	return NewServiceWithDependencies(provider, schoolLocation())
}

// NewServiceWithDependencies is primarily used by tests.
func NewServiceWithDependencies(provider Provider, location *time.Location) *Service {
	if location == nil {
		location = schoolLocation()
	}

	return &Service{provider: provider, location: location}
}

// GetYearView resolves the current school year and fetches its attendance
// statistics. Attendance data is always fetched fresh; there is no local cache.
func (s *Service) GetYearView() (*YearView, error) {
	begin, end, year, err := s.currentTermRange()
	if err != nil {
		return nil, err
	}

	beginText := begin.Format(statisticTimeLayout)
	endText := end.Format(statisticTimeLayout)

	overall, err := s.provider.GetAttendanceStatistic(beginText, endText)
	if err != nil {
		return nil, err
	}
	stats, err := s.provider.GetSubjectAttendanceStatistic(beginText, endText)
	if err != nil {
		return nil, err
	}

	rows := make([]SubjectRow, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, buildSubjectRow(stat))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Absent != rows[j].Absent {
			return rows[i].Absent > rows[j].Absent
		}
		if rows[i].RatePercent != rows[j].RatePercent {
			return rows[i].RatePercent < rows[j].RatePercent
		}
		return rows[i].Name < rows[j].Name
	})

	return &YearView{
		Begin:    begin,
		End:      end,
		Year:     year,
		Overall:  *overall,
		Subjects: rows,
	}, nil
}

// currentTermRange uses the current (isNow) semester's own start/end dates.
// At this school a "semester=1" entry already spans the whole school year
// (e.g. 2026-06-23 ~ 2027-02-20), which is exactly the range the web client
// sends; "semester=2" is the following Feb-July segment.
func (s *Service) currentTermRange() (time.Time, time.Time, uint64, error) {
	semesters, err := s.provider.GetSemesters()
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	for idx := range semesters {
		semester := semesters[idx]
		if !semester.IsNow {
			continue
		}
		begin, err := parseSemesterDate(semester.StartDate, s.location)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("semester %d start date: %w", semester.ID, err)
		}
		end, err := parseSemesterDate(semester.EndDate, s.location)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("semester %d end date: %w", semester.ID, err)
		}
		return startOfDay(begin), endOfDay(end), semester.Year, nil
	}

	return time.Time{}, time.Time{}, 0, fmt.Errorf("no current semester found")
}

// buildSubjectRow computes totals and the per-subject rate. The rate divides
// normal attendance by total sessions, matching the overall API rate observed
// in captures (39/47 = 82.98%).
func buildSubjectRow(stat models.SubjectAttendanceStat) SubjectRow {
	name := strings.TrimSpace(stat.SubjectName)
	if name == "" {
		name = strings.TrimSpace(stat.SubjectEName)
	}

	row := SubjectRow{
		Name:       name,
		Normal:     stat.NormalCount,
		Late:       stat.LateCount,
		EarlyLeave: stat.EarlyLeaveCount,
		Absent:     stat.AbsentCount,
		Total:      stat.NormalCount + stat.LateCount + stat.EarlyLeaveCount + stat.AbsentCount,
	}
	if row.Total > 0 {
		row.RatePercent = float64(row.Normal) / float64(row.Total) * 100
	}

	return row
}

func parseSemesterDate(value string, location *time.Location) (time.Time, error) {
	// Semester dates are RFC3339 ("2024-09-01T00:00:00+08:00"); accept a few
	// fallback layouts defensively.
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
	value = strings.TrimSpace(value)
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid semester date %q", value)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

func schoolLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}

	return location
}
