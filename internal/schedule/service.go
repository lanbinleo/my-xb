package schedule

import (
	"encoding/json"
	"fmt"
	"myxb/internal/config"
	"myxb/internal/models"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	defaultCacheTTL    = 30 * time.Minute
	dateLayout         = "2006-01-02"
	scheduleTimeLayout = "2006-01-02T15:04:05"
)

// Provider fetches schedule data from Xiaobao.
type Provider interface {
	ListScheduleByParent(beginTime, endTime string) ([]models.ScheduleItem, error)
}

// Cache stores schedule data between runs.
type Cache interface {
	LoadWeek(accountKey, beginTime, endTime string, ttl time.Duration) ([]models.ScheduleItem, bool, error)
	SaveWeek(accountKey, beginTime, endTime string, items []models.ScheduleItem) error
}

// Entry is a rendered-ready schedule item for a single day.
type Entry struct {
	ID           uint64
	Name         string
	EnglishName  string
	ScheduleType int
	Order        int
	Start        time.Time
	End          time.Time
	Location     string
	Teachers     []string
	Remark       string
	IsFreeBlock  bool
}

// CourseName returns the preferred course name.
func (e Entry) CourseName() string {
	if e.IsFreeBlock {
		return "Free Block"
	}

	if strings.TrimSpace(e.Name) != "" {
		return e.Name
	}

	return e.EnglishName
}

// TeacherSummary joins teachers into a readable string.
func (e Entry) TeacherSummary() string {
	if len(e.Teachers) == 0 {
		return ""
	}

	return strings.Join(e.Teachers, ", ")
}

// TimeRange formats the schedule entry time.
func (e Entry) TimeRange() string {
	return fmt.Sprintf("%s-%s", e.Start.Format("15:04"), e.End.Format("15:04"))
}

// TypeLabel returns a readable schedule type label.
func (e Entry) TypeLabel() string {
	if e.IsFreeBlock {
		return "Free"
	}

	switch e.ScheduleType {
	case 3:
		return "Club"
	case 4:
		return "Class"
	default:
		return "Event"
	}
}

// BlockLabel returns the displayed period/block label.
func (e Entry) BlockLabel(profile string) string {
	if e.Order <= 0 {
		return "-"
	}

	if profile == config.ScheduleProfileHighSchool {
		switch e.Order {
		case 6:
			return "B6"
		case 7:
			return "B7"
		case 8:
			return "B8"
		default:
			return fmt.Sprintf("P%d", e.Order)
		}
	}

	return fmt.Sprintf("#%d", e.Order)
}

// DayView is the schedule for one day plus live pointers for today.
type DayView struct {
	Date    time.Time
	Profile string
	Entries []Entry
	Current *Entry
	Next    *Entry
	IsToday bool
}

// ProfileLabel renders a friendly label for the saved schedule profile.
func ProfileLabel(profile string) string {
	switch config.NormalizeScheduleProfile(profile) {
	case config.ScheduleProfileHighSchool:
		return "High School"
	default:
		return "Standard"
	}
}

// Service handles schedule fetching, caching, and day-based queries.
type Service struct {
	provider Provider
	cache    Cache
	account  string
	now      func() time.Time
	location *time.Location
	cacheTTL time.Duration
}

// NewService creates a schedule service backed by the on-disk cache.
func NewService(provider Provider, accountKey string) *Service {
	location := schoolLocation()
	var cache Cache
	fileCache, err := newFileCache(time.Now)
	if err != nil {
		cache = noopCache{}
	} else {
		cache = fileCache
	}

	return &Service{
		provider: provider,
		cache:    cache,
		account:  normalizeAccountKey(accountKey),
		now:      time.Now,
		location: location,
		cacheTTL: defaultCacheTTL,
	}
}

// NewServiceWithDependencies is primarily used by tests.
func NewServiceWithDependencies(provider Provider, cache Cache, now func() time.Time, accountKey string) *Service {
	if cache == nil {
		cache = noopCache{}
	}
	if now == nil {
		now = time.Now
	}

	return &Service{
		provider: provider,
		cache:    cache,
		account:  normalizeAccountKey(accountKey),
		now:      now,
		location: schoolLocation(),
		cacheTTL: defaultCacheTTL,
	}
}

// ResolveDay parses a date or weekday selector relative to the current school week.
func (s *Service) ResolveDay(input string) (time.Time, error) {
	return ResolveDay(input, s.now(), s.location)
}

// Today returns the current school date at midnight.
func (s *Service) Today() time.Time {
	return startOfDay(s.now().In(s.location))
}

// GetDayView returns the schedule for the provided day.
func (s *Service) GetDayView(target time.Time, profile string, refresh bool) (DayView, error) {
	items, err := s.fetchWeek(target, refresh)
	if err != nil {
		return DayView{}, err
	}

	target = startOfDay(target.In(s.location))
	entries, err := s.entriesForDay(items, target, profile)
	if err != nil {
		return DayView{}, err
	}

	now := s.now().In(s.location)
	view := DayView{
		Date:    target,
		Profile: normalizedProfile(profile),
		Entries: entries,
		IsToday: sameDay(target, now),
	}

	if view.IsToday {
		view.Current, view.Next = currentAndNext(entries, now)
	}

	return view, nil
}

// ResolveDay parses dates like 2026-04-02, friday, 周五, today, or 明天.
func ResolveDay(input string, now time.Time, location *time.Location) (time.Time, error) {
	if location == nil {
		location = schoolLocation()
	}

	now = now.In(location)
	value := strings.TrimSpace(strings.ToLower(input))
	if value == "" || value == "today" || value == "今天" {
		return startOfDay(now), nil
	}
	if value == "tomorrow" || value == "明天" {
		return startOfDay(now.AddDate(0, 0, 1)), nil
	}

	if parsed, err := time.ParseInLocation(dateLayout, input, location); err == nil {
		return startOfDay(parsed), nil
	}

	if weekday, ok := weekdayAliases()[value]; ok {
		weekStart := startOfWeek(now)
		offset := int(weekday - time.Monday)
		if weekday == time.Sunday {
			offset = 6
		}
		return weekStart.AddDate(0, 0, offset), nil
	}

	return time.Time{}, fmt.Errorf("invalid day selector %q: use YYYY-MM-DD, today, tomorrow, monday, or 周一", input)
}

func weekdayAliases() map[string]time.Weekday {
	return map[string]time.Weekday{
		"monday":    time.Monday,
		"mon":       time.Monday,
		"周一":        time.Monday,
		"星期一":       time.Monday,
		"礼拜一":       time.Monday,
		"tuesday":   time.Tuesday,
		"tue":       time.Tuesday,
		"tues":      time.Tuesday,
		"周二":        time.Tuesday,
		"星期二":       time.Tuesday,
		"礼拜二":       time.Tuesday,
		"wednesday": time.Wednesday,
		"wed":       time.Wednesday,
		"周三":        time.Wednesday,
		"星期三":       time.Wednesday,
		"礼拜三":       time.Wednesday,
		"thursday":  time.Thursday,
		"thu":       time.Thursday,
		"thur":      time.Thursday,
		"thurs":     time.Thursday,
		"周四":        time.Thursday,
		"星期四":       time.Thursday,
		"礼拜四":       time.Thursday,
		"friday":    time.Friday,
		"fri":       time.Friday,
		"周五":        time.Friday,
		"星期五":       time.Friday,
		"礼拜五":       time.Friday,
		"saturday":  time.Saturday,
		"sat":       time.Saturday,
		"周六":        time.Saturday,
		"星期六":       time.Saturday,
		"礼拜六":       time.Saturday,
		"sunday":    time.Sunday,
		"sun":       time.Sunday,
		"周日":        time.Sunday,
		"周天":        time.Sunday,
		"星期日":       time.Sunday,
		"星期天":       time.Sunday,
		"礼拜天":       time.Sunday,
	}
}

func (s *Service) fetchWeek(target time.Time, refresh bool) ([]models.ScheduleItem, error) {
	begin := startOfWeek(target.In(s.location))
	end := begin.AddDate(0, 0, 6)
	beginText := begin.Format(dateLayout)
	endText := end.Format(dateLayout)

	if !refresh {
		items, hit, err := s.cache.LoadWeek(s.account, beginText, endText, s.cacheTTL)
		if err == nil && hit {
			return items, nil
		}
	}

	items, err := s.provider.ListScheduleByParent(beginText, endText)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SaveWeek(s.account, beginText, endText, items)
	return items, nil
}

func (s *Service) entriesForDay(items []models.ScheduleItem, target time.Time, profile string) ([]Entry, error) {
	profile = normalizedProfile(profile)
	entries := make([]Entry, 0)

	for _, item := range items {
		begin, err := time.ParseInLocation(scheduleTimeLayout, item.BeginTime, s.location)
		if err != nil {
			return nil, fmt.Errorf("parse schedule begin time %q: %w", item.BeginTime, err)
		}
		end, err := time.ParseInLocation(scheduleTimeLayout, item.EndTime, s.location)
		if err != nil {
			return nil, fmt.Errorf("parse schedule end time %q: %w", item.EndTime, err)
		}
		if !sameDay(begin, target) {
			continue
		}

		begin, end = overrideTimeRange(target, item.FormalCourseOrder, begin, end, profile)
		entry := Entry{
			ID:           item.ID,
			Name:         item.Name,
			EnglishName:  item.EName,
			ScheduleType: item.ScheduleType,
			Order:        item.FormalCourseOrder,
			Start:        begin,
			End:          end,
			Location:     firstNonEmpty(item.PlaygroundName, item.PlaygroundEName),
			Teachers:     teacherNames(item.TeacherList),
			Remark:       item.Remark,
		}
		entries = append(entries, entry)
	}

	entries = fillFreeBlocks(entries, target, profile)

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Start.Equal(entries[j].Start) {
			if entries[i].Order == entries[j].Order {
				return entries[i].ID < entries[j].ID
			}
			return entries[i].Order < entries[j].Order
		}
		return entries[i].Start.Before(entries[j].Start)
	})

	return entries, nil
}

func teacherNames(teachers []models.ScheduleTeacher) []string {
	names := make([]string, 0, len(teachers))
	for _, teacher := range teachers {
		name := strings.TrimSpace(teacher.Name)
		if name == "" {
			name = strings.TrimSpace(teacher.EName)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func currentAndNext(entries []Entry, now time.Time) (*Entry, *Entry) {
	for idx := range entries {
		entry := entries[idx]
		if entry.IsFreeBlock {
			continue
		}
		if !now.Before(entry.Start) && now.Before(entry.End) {
			current := entry
			if next := nextRealEntry(entries[idx+1:]); next != nil {
				return &current, next
			}
			return &current, nil
		}
		if now.Before(entry.Start) {
			next := entry
			return nil, &next
		}
	}

	return nil, nil
}

func nextRealEntry(entries []Entry) *Entry {
	for idx := range entries {
		if entries[idx].IsFreeBlock {
			continue
		}

		next := entries[idx]
		return &next
	}

	return nil
}

func overrideTimeRange(date time.Time, order int, begin, end time.Time, profile string) (time.Time, time.Time) {
	if profile != config.ScheduleProfileHighSchool {
		return begin, end
	}

	slot, ok := highSchoolSlots[order]
	if !ok {
		return begin, end
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), slot.startHour, slot.startMinute, 0, 0, begin.Location())
	finish := time.Date(date.Year(), date.Month(), date.Day(), slot.endHour, slot.endMinute, 0, 0, begin.Location())
	return start, finish
}

type slotRange struct {
	startHour   int
	startMinute int
	endHour     int
	endMinute   int
}

var highSchoolSlots = map[int]slotRange{
	1: {startHour: 8, startMinute: 0, endHour: 8, endMinute: 40},
	2: {startHour: 8, startMinute: 40, endHour: 9, endMinute: 20},
	3: {startHour: 10, startMinute: 10, endHour: 10, endMinute: 50},
	4: {startHour: 11, startMinute: 0, endHour: 11, endMinute: 40},
	5: {startHour: 11, startMinute: 50, endHour: 12, endMinute: 30},
	6: {startHour: 13, startMinute: 25, endHour: 14, endMinute: 5},
	7: {startHour: 14, startMinute: 15, endHour: 14, endMinute: 55},
	8: {startHour: 15, startMinute: 5, endHour: 15, endMinute: 45},
}

var standardSlots = map[int]slotRange{
	1: {startHour: 8, startMinute: 25, endHour: 9, endMinute: 5},
	2: {startHour: 9, startMinute: 15, endHour: 9, endMinute: 55},
	3: {startHour: 10, startMinute: 10, endHour: 10, endMinute: 50},
	4: {startHour: 11, startMinute: 0, endHour: 11, endMinute: 40},
	5: {startHour: 12, startMinute: 35, endHour: 13, endMinute: 15},
	6: {startHour: 13, startMinute: 25, endHour: 14, endMinute: 5},
	7: {startHour: 14, startMinute: 15, endHour: 14, endMinute: 55},
	8: {startHour: 15, startMinute: 5, endHour: 15, endMinute: 45},
}

func fillFreeBlocks(entries []Entry, date time.Time, profile string) []Entry {
	occupied := make(map[int]bool, 8)
	for _, entry := range entries {
		if entry.Order < 1 || entry.Order > 8 {
			continue
		}
		if entry.IsFreeBlock {
			continue
		}

		occupied[entry.Order] = true
	}

	filled := make([]Entry, 0, len(entries)+8)
	filled = append(filled, entries...)

	for order := 1; order <= 8; order++ {
		if occupied[order] {
			continue
		}

		start, end, ok := blockTimeRange(date, order, profile)
		if !ok {
			continue
		}

		filled = append(filled, Entry{
			Order:       order,
			Start:       start,
			End:         end,
			IsFreeBlock: true,
		})
	}

	return filled
}

func blockTimeRange(date time.Time, order int, profile string) (time.Time, time.Time, bool) {
	slots := standardSlots
	if normalizedProfile(profile) == config.ScheduleProfileHighSchool {
		slots = highSchoolSlots
	}

	slot, ok := slots[order]
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), slot.startHour, slot.startMinute, 0, 0, date.Location())
	end := time.Date(date.Year(), date.Month(), date.Day(), slot.endHour, slot.endMinute, 0, 0, date.Location())
	return start, end, true
}

func normalizedProfile(profile string) string {
	if normalized := config.NormalizeScheduleProfile(profile); normalized != "" {
		return normalized
	}

	return config.ScheduleProfileStandard
}

func sameDay(a time.Time, b time.Time) bool {
	a = a.In(b.Location())
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	day := startOfDay(t)
	offset := int(day.Weekday() - time.Monday)
	if day.Weekday() == time.Sunday {
		offset = 6
	}
	return day.AddDate(0, 0, -offset)
}

func schoolLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}

	return location
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

type noopCache struct{}

func (noopCache) LoadWeek(string, string, string, time.Duration) ([]models.ScheduleItem, bool, error) {
	return nil, false, nil
}

func (noopCache) SaveWeek(string, string, string, []models.ScheduleItem) error {
	return nil
}

type fileCache struct {
	path string
	now  func() time.Time
}

type cacheFileData struct {
	Weeks map[string]cacheWeek `json:"weeks"`
}

type cacheWeek struct {
	FetchedAt string                `json:"fetched_at"`
	Items     []models.ScheduleItem `json:"items"`
}

func newFileCache(now func() time.Time) (*fileCache, error) {
	path, err := config.GetScheduleCachePath()
	if err != nil {
		return nil, err
	}

	return &fileCache{path: path, now: now}, nil
}

func (c *fileCache) LoadWeek(accountKey, beginTime, endTime string, ttl time.Duration) ([]models.ScheduleItem, bool, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var cacheData cacheFileData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil, false, err
	}

	week, ok := cacheData.Weeks[weekKey(accountKey, beginTime, endTime)]
	if !ok {
		return nil, false, nil
	}

	fetchedAt, err := time.Parse(time.RFC3339, week.FetchedAt)
	if err != nil {
		return nil, false, err
	}
	if c.now().After(fetchedAt.Add(ttl)) {
		return nil, false, nil
	}

	return week.Items, true, nil
}

func (c *fileCache) SaveWeek(accountKey, beginTime, endTime string, items []models.ScheduleItem) error {
	cacheData := cacheFileData{Weeks: map[string]cacheWeek{}}
	data, err := os.ReadFile(c.path)
	if err == nil {
		if err := json.Unmarshal(data, &cacheData); err != nil {
			cacheData = cacheFileData{Weeks: map[string]cacheWeek{}}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if cacheData.Weeks == nil {
		cacheData.Weeks = map[string]cacheWeek{}
	}

	cacheData.Weeks[weekKey(accountKey, beginTime, endTime)] = cacheWeek{
		FetchedAt: c.now().Format(time.RFC3339),
		Items:     items,
	}

	encoded, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, encoded, 0600)
}

func weekKey(accountKey, beginTime, endTime string) string {
	return normalizeAccountKey(accountKey) + "|" + beginTime + "_" + endTime
}

func normalizeAccountKey(accountKey string) string {
	accountKey = strings.ToLower(strings.TrimSpace(accountKey))
	if accountKey == "" {
		return "default"
	}

	return accountKey
}
