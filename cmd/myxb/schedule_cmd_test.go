package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"myxb/internal/config"
	"myxb/internal/models"
	"myxb/internal/schedule"
)

func TestLazyScheduleProviderConnectsOnceAndOnlyWhenNeeded(t *testing.T) {
	connects := 0
	inner := &stubScheduleProvider{
		items: []models.ScheduleItem{
			{ID: 1, Name: "AP Statistics", BeginTime: "2026-04-02T08:25:00", EndTime: "2026-04-02T09:05:00", FormalCourseOrder: 1, ScheduleType: 4},
		},
	}
	provider := &lazyScheduleProvider{
		connect: func() (schedule.Provider, error) {
			connects++
			return inner, nil
		},
	}

	if _, err := provider.ListScheduleByParent("2026-03-30", "2026-04-05"); err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if _, err := provider.ListScheduleByParent("2026-03-30", "2026-04-05"); err != nil {
		t.Fatalf("second call returned error: %v", err)
	}

	if connects != 1 {
		t.Fatalf("connect count = %d, want 1", connects)
	}
	if inner.calls != 2 {
		t.Fatalf("provider call count = %d, want 2", inner.calls)
	}
}

func TestLazyScheduleProviderPropagatesConnectError(t *testing.T) {
	provider := &lazyScheduleProvider{
		connect: func() (schedule.Provider, error) {
			return nil, errors.New("boom")
		},
	}

	if _, err := provider.ListScheduleByParent("2026-03-30", "2026-04-05"); err == nil {
		t.Fatal("ListScheduleByParent should propagate the connect error")
	}
}

func TestResolveScheduleProfileFromConfigPrefersOverrideThenSavedThenDefault(t *testing.T) {
	cfg := &config.Config{ScheduleProfile: config.ScheduleProfileHighSchool}

	if profile, isDefault, err := resolveScheduleProfileFromConfig(cfg, "standard"); err != nil || profile != config.ScheduleProfileStandard || isDefault {
		t.Fatalf("override resolution = %q (default=%v, err=%v), want standard", profile, isDefault, err)
	}
	if profile, isDefault, err := resolveScheduleProfileFromConfig(cfg, ""); err != nil || profile != config.ScheduleProfileHighSchool || isDefault {
		t.Fatalf("saved resolution = %q (default=%v, err=%v), want highschool", profile, isDefault, err)
	}
	if _, _, err := resolveScheduleProfileFromConfig(cfg, "nonsense"); err == nil {
		t.Fatal("invalid override should return an error")
	}

	empty := &config.Config{}
	if profile, isDefault, err := resolveScheduleProfileFromConfig(empty, ""); err != nil || profile != config.ScheduleProfileStandard || !isDefault {
		t.Fatalf("default resolution = %q (default=%v, err=%v), want standard default", profile, isDefault, err)
	}
}

func TestRenderScheduleFocusNextMode(t *testing.T) {
	location := mustTestLocation(t)

	current := schedule.Entry{
		ID:           1,
		Name:         "AP English",
		ScheduleType: 4,
		Order:        2,
		Start:        time.Date(2026, 4, 2, 8, 40, 0, 0, location),
		End:          time.Date(2026, 4, 2, 9, 20, 0, 0, location),
	}
	view := schedule.DayView{
		Date:    time.Date(2026, 4, 2, 0, 0, 0, 0, location),
		Profile: config.ScheduleProfileHighSchool,
		Entries: []schedule.Entry{current},
		Current: &current,
		IsToday: true,
	}

	rendered := renderScheduleFocus(view, focusNext)
	if !strings.Contains(rendered, "No later classes today.") {
		t.Fatalf("focusNext output = %q, want no-later-classes message", rendered)
	}
	if !strings.Contains(rendered, "NOW") {
		t.Fatalf("focusNext output = %q, want fallback NOW card", rendered)
	}
}

func TestRenderScheduleWeekGridAndOtherItems(t *testing.T) {
	location := mustTestLocation(t)

	monday := time.Date(2026, 4, 6, 0, 0, 0, 0, location)
	current := schedule.Entry{
		ID:           7,
		Name:         "高等数学与线性代数",
		ScheduleType: 4,
		Order:        2,
		Start:        time.Date(2026, 4, 6, 8, 40, 0, 0, location),
		End:          time.Date(2026, 4, 6, 9, 20, 0, 0, location),
		Location:     "D411",
	}
	club := schedule.Entry{
		ID:           8,
		Name:         "Robotics Club",
		ScheduleType: 3,
		Order:        7,
		Start:        time.Date(2026, 4, 8, 14, 15, 0, 0, location),
		End:          time.Date(2026, 4, 8, 14, 55, 0, 0, location),
	}
	activity := schedule.Entry{
		ID:           9,
		Name:         "Assembly",
		ScheduleType: 5,
		Order:        0,
		Start:        time.Date(2026, 4, 9, 16, 0, 0, 0, location),
		End:          time.Date(2026, 4, 9, 17, 0, 0, 0, location),
	}

	days := make([]schedule.DayView, 7)
	for idx := range days {
		days[idx] = schedule.DayView{
			Date:    monday.AddDate(0, 0, idx),
			Profile: config.ScheduleProfileHighSchool,
		}
	}
	days[0].Entries = []schedule.Entry{current}
	days[0].Current = &current
	days[0].IsToday = true
	days[2].Entries = []schedule.Entry{club}
	days[3].Entries = []schedule.Entry{activity}
	view := schedule.WeekView{
		Begin:   monday,
		Profile: config.ScheduleProfileHighSchool,
		Days:    days,
		Today:   &days[0],
	}

	rendered := renderScheduleWeek(view)
	for _, want := range []string{"2026-04-06 ~ 2026-04-12", "周一", "周日", "Free", "Robotics Club", "Assembly", "Other items:"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("renderScheduleWeek output missing %q:\n%s", want, rendered)
		}
	}
	if !strings.Contains(rendered, "高等数学与线性...") {
		t.Fatalf("renderScheduleWeek output should truncate wide CJK names:\n%s", rendered)
	}
	if strings.Contains(rendered, "…") {
		t.Fatalf("renderScheduleWeek must not use the width-ambiguous ellipsis:\n%s", rendered)
	}
}

func TestTruncateForWeekCell(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "AP Physics", want: "AP Physics"},
		{input: "AP Physics C Mechanics", want: "AP Physics C..."},
		{input: "AP English Literature and Composition", want: "AP English..."},
		{input: "Chinese Culture & Literature", want: "Chinese Culture..."},
		{input: "高等数学与线性代数", want: "高等数学与线性..."},
		{input: "化学", want: "化学"},
	}

	for _, tt := range tests {
		if got := truncateForWeekCell(tt.input); got != tt.want {
			t.Fatalf("truncateForWeekCell(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

type stubScheduleProvider struct {
	items []models.ScheduleItem
	calls int
}

func (s *stubScheduleProvider) ListScheduleByParent(beginTime, endTime string) ([]models.ScheduleItem, error) {
	s.calls++
	items := make([]models.ScheduleItem, len(s.items))
	copy(items, s.items)
	return items, nil
}

func mustTestLocation(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}
	return location
}
