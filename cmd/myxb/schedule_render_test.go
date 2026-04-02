package main

import (
	"strings"
	"testing"
	"time"

	"myxb/internal/config"
	"myxb/internal/schedule"
)

func TestRenderScheduleDayIncludesCurrentAndNextMarkers(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}

	current := schedule.Entry{
		ID:           1,
		Name:         "AP English",
		ScheduleType: 4,
		Order:        6,
		Start:        time.Date(2026, 4, 2, 13, 25, 0, 0, location),
		End:          time.Date(2026, 4, 2, 14, 5, 0, 0, location),
		Location:     "D411",
		Teachers:     []string{"Jacqueline Browne"},
	}
	free := schedule.Entry{
		Order:       5,
		Start:       time.Date(2026, 4, 2, 11, 50, 0, 0, location),
		End:         time.Date(2026, 4, 2, 12, 30, 0, 0, location),
		IsFreeBlock: true,
	}
	next := schedule.Entry{
		ID:           2,
		Name:         "AP Psychology",
		ScheduleType: 4,
		Order:        7,
		Start:        time.Date(2026, 4, 2, 14, 15, 0, 0, location),
		End:          time.Date(2026, 4, 2, 14, 55, 0, 0, location),
		Location:     "A418",
		Teachers:     []string{"Irene Shi"},
	}
	view := schedule.DayView{
		Date:    time.Date(2026, 4, 2, 0, 0, 0, 0, location),
		Profile: config.ScheduleProfileHighSchool,
		Entries: []schedule.Entry{free, current, next},
		Current: &current,
		Next:    &next,
		IsToday: true,
	}

	rendered := renderScheduleDay(view)
	if !strings.Contains(rendered, "High School") {
		t.Fatalf("renderScheduleDay output = %q, want profile label", rendered)
	}
	if !strings.Contains(rendered, "NOW") {
		t.Fatalf("renderScheduleDay output = %q, want NOW marker", rendered)
	}
	if !strings.Contains(rendered, "NEXT") {
		t.Fatalf("renderScheduleDay output = %q, want NEXT marker", rendered)
	}
	if !strings.Contains(rendered, "Free Block") {
		t.Fatalf("renderScheduleDay output = %q, want Free Block row", rendered)
	}
}

func TestRenderScheduleFocusBreaksWhenNoCurrentClass(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}

	next := schedule.Entry{
		ID:           2,
		Name:         "AP Psychology",
		ScheduleType: 4,
		Order:        7,
		Start:        time.Date(2026, 4, 2, 14, 15, 0, 0, location),
		End:          time.Date(2026, 4, 2, 14, 55, 0, 0, location),
	}
	view := schedule.DayView{
		Date:    time.Date(2026, 4, 2, 0, 0, 0, 0, location),
		Profile: config.ScheduleProfileStandard,
		Entries: []schedule.Entry{next},
		Next:    &next,
		IsToday: true,
	}

	rendered := renderScheduleFocus(view, "now")
	if !strings.Contains(rendered, "No class is in session right now.") {
		t.Fatalf("renderScheduleFocus output = %q, want break message", rendered)
	}
	if !strings.Contains(rendered, "NEXT") {
		t.Fatalf("renderScheduleFocus output = %q, want next card", rendered)
	}
}
