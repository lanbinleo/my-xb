package schedule

import (
	"myxb/internal/config"
	"myxb/internal/models"
	"testing"
	"time"
)

func TestResolveDaySupportsDateAndWeekdayAliases(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}

	now := time.Date(2026, 4, 1, 9, 0, 0, 0, location) // Wednesday

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "2026-04-01"},
		{input: "2026-04-02", want: "2026-04-02"},
		{input: "friday", want: "2026-04-03"},
		{input: "周五", want: "2026-04-03"},
		{input: "today", want: "2026-04-01"},
		{input: "明天", want: "2026-04-02"},
	}

	for _, tt := range tests {
		got, err := ResolveDay(tt.input, now, location)
		if err != nil {
			t.Fatalf("ResolveDay(%q) returned error: %v", tt.input, err)
		}
		if got.Format(dateLayout) != tt.want {
			t.Fatalf("ResolveDay(%q) = %s, want %s", tt.input, got.Format(dateLayout), tt.want)
		}
	}
}

func TestGetDayViewAppliesHighSchoolBellScheduleAndLivePointers(t *testing.T) {
	now := mustSchoolTime(t, "2026-04-02T08:50:00")
	provider := &fakeProvider{
		items: []models.ScheduleItem{
			{ID: 2, Name: "AP English", BeginTime: "2026-04-02T09:15:00", EndTime: "2026-04-02T09:55:00", FormalCourseOrder: 2, ScheduleType: 4},
			{ID: 3, Name: "AP Physics", BeginTime: "2026-04-02T10:10:00", EndTime: "2026-04-02T10:50:00", FormalCourseOrder: 3, ScheduleType: 4},
		},
	}

	service := NewServiceWithDependencies(provider, &memoryCache{}, func() time.Time { return now }, "alice")
	view, err := service.GetDayView(now, config.ScheduleProfileHighSchool, false)
	if err != nil {
		t.Fatalf("GetDayView returned error: %v", err)
	}

	if len(view.Entries) != 8 {
		t.Fatalf("GetDayView entry count = %d, want 8 core blocks", len(view.Entries))
	}
	if view.Entries[0].Start.Format("15:04") != "08:00" || view.Entries[0].End.Format("15:04") != "08:40" {
		t.Fatalf("high-school override for period 1 = %s-%s, want 08:00-08:40", view.Entries[0].Start.Format("15:04"), view.Entries[0].End.Format("15:04"))
	}
	if !view.Entries[0].IsFreeBlock {
		t.Fatalf("period 1 should be a free block, got %+v", view.Entries[0])
	}
	if view.Entries[1].Start.Format("15:04") != "08:40" || view.Entries[1].End.Format("15:04") != "09:20" {
		t.Fatalf("high-school override for period 2 = %s-%s, want 08:40-09:20", view.Entries[1].Start.Format("15:04"), view.Entries[1].End.Format("15:04"))
	}
	if view.Current == nil || view.Current.ID != 2 {
		t.Fatalf("current entry = %+v, want period 2", view.Current)
	}
	if view.Next == nil || view.Next.ID != 3 {
		t.Fatalf("next entry = %+v, want period 3", view.Next)
	}
}

func TestGetDayViewUsesCachedWeekBeforeCallingProviderAgain(t *testing.T) {
	now := mustSchoolTime(t, "2026-04-02T08:10:00")
	provider := &fakeProvider{
		items: []models.ScheduleItem{
			{ID: 1, Name: "AP Statistics", BeginTime: "2026-04-02T08:25:00", EndTime: "2026-04-02T09:05:00", FormalCourseOrder: 1, ScheduleType: 4},
		},
	}
	cache := &memoryCache{}

	service := NewServiceWithDependencies(provider, cache, func() time.Time { return now }, "alice")
	if _, err := service.GetDayView(now, config.ScheduleProfileStandard, false); err != nil {
		t.Fatalf("first GetDayView returned error: %v", err)
	}
	if _, err := service.GetDayView(now, config.ScheduleProfileStandard, false); err != nil {
		t.Fatalf("second GetDayView returned error: %v", err)
	}

	if provider.calls != 1 {
		t.Fatalf("provider call count = %d, want 1", provider.calls)
	}
}

func TestGetDayViewFillsStandardFreeBlocks(t *testing.T) {
	now := mustSchoolTime(t, "2026-04-02T12:00:00")
	provider := &fakeProvider{
		items: []models.ScheduleItem{
			{ID: 5, Name: "AP English", BeginTime: "2026-04-02T12:35:00", EndTime: "2026-04-02T13:15:00", FormalCourseOrder: 5, ScheduleType: 4},
		},
	}

	service := NewServiceWithDependencies(provider, &memoryCache{}, func() time.Time { return now }, "alice")
	view, err := service.GetDayView(now, config.ScheduleProfileStandard, false)
	if err != nil {
		t.Fatalf("GetDayView returned error: %v", err)
	}

	if len(view.Entries) != 8 {
		t.Fatalf("GetDayView entry count = %d, want 8", len(view.Entries))
	}
	if !view.Entries[0].IsFreeBlock {
		t.Fatalf("period 1 should be a free block, got %+v", view.Entries[0])
	}
	if view.Entries[0].Start.Format("15:04") != "08:25" || view.Entries[0].End.Format("15:04") != "09:05" {
		t.Fatalf("standard free block time = %s-%s, want 08:25-09:05", view.Entries[0].Start.Format("15:04"), view.Entries[0].End.Format("15:04"))
	}
	if view.Entries[4].ID != 5 || view.Entries[4].IsFreeBlock {
		t.Fatalf("period 5 should keep the real class, got %+v", view.Entries[4])
	}
}

func TestGetDayViewSeparatesCacheByAccount(t *testing.T) {
	now := mustSchoolTime(t, "2026-04-02T08:10:00")
	provider := &fakeProvider{
		items: []models.ScheduleItem{
			{ID: 1, Name: "AP Statistics", BeginTime: "2026-04-02T08:25:00", EndTime: "2026-04-02T09:05:00", FormalCourseOrder: 1, ScheduleType: 4},
		},
	}
	cache := &memoryCache{}

	serviceAlice := NewServiceWithDependencies(provider, cache, func() time.Time { return now }, "alice")
	if _, err := serviceAlice.GetDayView(now, config.ScheduleProfileStandard, false); err != nil {
		t.Fatalf("alice GetDayView returned error: %v", err)
	}

	serviceBob := NewServiceWithDependencies(provider, cache, func() time.Time { return now }, "bob")
	if _, err := serviceBob.GetDayView(now, config.ScheduleProfileStandard, false); err != nil {
		t.Fatalf("bob GetDayView returned error: %v", err)
	}

	if provider.calls != 2 {
		t.Fatalf("provider call count = %d, want 2 for separate account caches", provider.calls)
	}
}

type fakeProvider struct {
	items []models.ScheduleItem
	calls int
}

func (f *fakeProvider) ListScheduleByParent(beginTime, endTime string) ([]models.ScheduleItem, error) {
	f.calls++
	items := make([]models.ScheduleItem, len(f.items))
	copy(items, f.items)
	return items, nil
}

type memoryCache struct {
	items map[string][]models.ScheduleItem
}

func (m *memoryCache) LoadWeek(accountKey, beginTime, endTime string, ttl time.Duration) ([]models.ScheduleItem, bool, error) {
	if m.items == nil {
		return nil, false, nil
	}

	key := weekKey(accountKey, beginTime, endTime)
	items, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}

	copied := make([]models.ScheduleItem, len(items))
	copy(copied, items)
	return copied, true, nil
}

func (m *memoryCache) SaveWeek(accountKey, beginTime, endTime string, items []models.ScheduleItem) error {
	if m.items == nil {
		m.items = map[string][]models.ScheduleItem{}
	}

	copied := make([]models.ScheduleItem, len(items))
	copy(copied, items)
	m.items[weekKey(accountKey, beginTime, endTime)] = copied
	return nil
}

func mustSchoolTime(t *testing.T, value string) time.Time {
	t.Helper()

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}

	parsed, err := time.ParseInLocation(scheduleTimeLayout, value, location)
	if err != nil {
		t.Fatalf("ParseInLocation(%q) returned error: %v", value, err)
	}

	return parsed
}
