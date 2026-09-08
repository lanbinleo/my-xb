package main

import (
	"context"
	"fmt"
	"myxb/internal/config"
	"myxb/internal/models"
	"myxb/internal/schedule"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/urfave/cli/v3"
)

// focusMode selects which live entry the focus view highlights.
type focusMode int

const (
	focusNow focusMode = iota
	focusNext
)

// weekCellMaxWidth caps week-grid cell content. The ASCII ellipsis keeps the
// cut marker an unambiguous terminal width: "…" (U+2026) renders fullwidth in
// many CJK fonts and misaligns the table.
const weekCellMaxWidth = 18

const weekCellEllipsis = "..."

func newScheduleCommand() *cli.Command {
	return &cli.Command{
		Name:    "schedule",
		Aliases: []string{"s", "cal", "calendar"},
		Usage:   "View the weekly timetable, current class, and daily schedules",
		Flags:   scheduleDayFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runScheduleWeekCommand(c, scheduleSelectorArg(c))
		},
		Commands: []*cli.Command{
			{
				Name:    "now",
				Aliases: []string{"n"},
				Usage:   "Show the class happening right now",
				Flags:   scheduleCommonFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleFocusCommand(c, focusNow)
				},
			},
			{
				Name:    "next",
				Aliases: []string{"ne"},
				Usage:   "Show the next class for today",
				Flags:   scheduleCommonFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleFocusCommand(c, focusNext)
				},
			},
			{
				Name:      "day",
				Aliases:   []string{"d"},
				Usage:     "Show the timetable for a date or weekday",
				ArgsUsage: "[date-or-weekday]",
				Flags:     scheduleDayFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleDayCommand(c, scheduleSelectorArg(c))
				},
			},
			{
				Name:      "week",
				Aliases:   []string{"w"},
				Usage:     "Show the timetable for a whole week",
				ArgsUsage: "[date-or-weekday]",
				Flags:     scheduleDayFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleWeekCommand(c, scheduleSelectorArg(c))
				},
			},
			{
				Name:      "profile",
				Aliases:   []string{"p"},
				Usage:     "Show or set your saved schedule profile",
				ArgsUsage: "[standard|highschool]",
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleProfileCommand(c)
				},
			},
		},
	}
}

// newScheduleShortcutCommands exposes the most frequent timetable queries as
// top-level commands so `myxb now` or `myxb week` work without the group name.
func newScheduleShortcutCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "now",
			Usage: "Show the class happening right now (shortcut for 'schedule now')",
			Flags: scheduleCommonFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleFocusCommand(c, focusNow)
			},
		},
		{
			Name:  "next",
			Usage: "Show the next class for today (shortcut for 'schedule next')",
			Flags: scheduleCommonFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleFocusCommand(c, focusNext)
			},
		},
		{
			Name:      "day",
			Usage:     "Show the timetable for a date or weekday (shortcut for 'schedule day')",
			ArgsUsage: "[date-or-weekday]",
			Flags:     scheduleDayFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleDayCommand(c, scheduleSelectorArg(c))
			},
		},
		{
			Name:      "week",
			Usage:     "Show the timetable for a whole week (shortcut for 'schedule week')",
			ArgsUsage: "[date-or-weekday]",
			Flags:     scheduleDayFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleWeekCommand(c, scheduleSelectorArg(c))
			},
		},
	}
}

func scheduleCommonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "refresh",
			Aliases: []string{"r"},
			Usage:   "Bypass the local schedule cache and fetch fresh data",
		},
		&cli.StringFlag{
			Name:  "profile",
			Usage: "Temporarily use a schedule profile: standard or highschool",
		},
	}
}

func scheduleDayFlags() []cli.Flag {
	flags := scheduleCommonFlags()
	flags = append(flags, &cli.StringFlag{
		Name:    "date",
		Aliases: []string{"d"},
		Usage:   "Date or weekday to view: YYYY-MM-DD, monday, 周一, today, tomorrow, or 下周五",
	})
	return flags
}

func scheduleSelectorArg(c *cli.Command) string {
	selector := strings.TrimSpace(c.Args().First())
	if selector == "" {
		selector = strings.TrimSpace(c.String("date"))
	}
	return selector
}

func runScheduleFocusCommand(c *cli.Command, mode focusMode) error {
	service, profile, err := prepareScheduleRun(c)
	if err != nil {
		return err
	}

	view, err := service.GetDayView(service.Today(), profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	fmt.Print(renderScheduleFocus(view, mode))
	return nil
}

func runScheduleDayCommand(c *cli.Command, selector string) error {
	service, profile, err := prepareScheduleRun(c)
	if err != nil {
		return err
	}

	target, err := service.ResolveDay(selector)
	if err != nil {
		return err
	}

	view, err := service.GetDayView(target, profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	fmt.Print(renderScheduleDay(view))
	return nil
}

func runScheduleWeekCommand(c *cli.Command, selector string) error {
	service, profile, err := prepareScheduleRun(c)
	if err != nil {
		return err
	}

	target, err := service.ResolveDay(selector)
	if err != nil {
		return err
	}

	view, err := service.GetWeekView(target, profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	fmt.Print(renderScheduleWeek(view))
	return nil
}

func runScheduleProfileCommand(c *cli.Command) error {
	rawValue := strings.TrimSpace(c.Args().First())
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if rawValue == "" {
		if profile, ok := cfg.ConfiguredScheduleProfile(); ok {
			fmt.Printf("%s %s\n", bold("Saved schedule profile:"), cyan(schedule.ProfileLabel(profile)))
			fmt.Printf("%s %s\n", gray("Use this when reading timetable blocks and current/next class timing."), gray("(Set with: myxb schedule profile highschool)"))
			return nil
		}

		fmt.Printf("%s %s\n", bold("Saved schedule profile:"), yellow("Not set (defaults to Standard)"))
		fmt.Printf("%s\n", gray("Set one to persist your bell schedule:"))
		fmt.Printf("%s\n", gray("  myxb schedule profile standard"))
		fmt.Printf("%s\n", gray("  myxb schedule profile highschool"))
		return nil
	}

	normalized := config.NormalizeScheduleProfile(rawValue)
	if normalized == "" {
		return fmt.Errorf("invalid schedule profile %q: use standard or highschool", rawValue)
	}

	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.ScheduleProfile = normalized

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save schedule profile: %w", err)
	}

	printSuccess(fmt.Sprintf("Saved schedule profile: %s", schedule.ProfileLabel(normalized)))
	return nil
}

// prepareScheduleRun loads the config once, checks for saved credentials, and
// builds a service whose provider only logs in when the cache misses.
func prepareScheduleRun(c *cli.Command) (*schedule.Service, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %w", err)
	}
	if !cfg.HasCredentials() {
		return nil, "", fmt.Errorf("not logged in: run 'myxb login' first")
	}

	profile, isDefault, err := resolveScheduleProfileFromConfig(cfg, c.String("profile"))
	if err != nil {
		return nil, "", err
	}
	if isDefault {
		printInfo(gray("Schedule profile defaults to Standard; set it with: myxb schedule profile highschool"))
	}

	service := schedule.NewService(newLazyScheduleProvider(), cfg.Username)
	return service, profile, nil
}

// resolveScheduleProfileFromConfig picks the profile from an explicit override,
// the saved config, or the Standard default.
func resolveScheduleProfileFromConfig(cfg *config.Config, override string) (profile string, isDefault bool, err error) {
	if strings.TrimSpace(override) != "" {
		normalized := config.NormalizeScheduleProfile(override)
		if normalized == "" {
			return "", false, fmt.Errorf("invalid schedule profile %q: use standard or highschool", override)
		}
		return normalized, false, nil
	}

	if configured, ok := cfg.ConfiguredScheduleProfile(); ok {
		return configured, false, nil
	}

	return config.ScheduleProfileStandard, true, nil
}

// lazyScheduleProvider connects to Xiaobao on first real use so cache hits
// skip the login round trip entirely.
type lazyScheduleProvider struct {
	connect func() (schedule.Provider, error)
	client  schedule.Provider
}

func newLazyScheduleProvider() *lazyScheduleProvider {
	return &lazyScheduleProvider{
		connect: func() (schedule.Provider, error) {
			apiClient, err := ensureLogin(true)
			if err != nil {
				return nil, fmt.Errorf("login failed: %w", err)
			}
			printSuccess("Authentication successful!")
			fmt.Println()
			return apiClient, nil
		},
	}
}

func (p *lazyScheduleProvider) ListScheduleByParent(beginTime, endTime string) ([]models.ScheduleItem, error) {
	if p.client == nil {
		client, err := p.connect()
		if err != nil {
			return nil, err
		}
		p.client = client
	}
	return p.client.ListScheduleByParent(beginTime, endTime)
}

func renderScheduleHeader(view schedule.DayView) string {
	var builder strings.Builder

	builder.WriteString(bold(cyan(view.Date.Format("2006-01-02"))))
	builder.WriteString(" ")
	builder.WriteString(gray(weekdayLabel(view.Date.Weekday())))
	builder.WriteString("\n")
	builder.WriteString(gray("Profile: "))
	builder.WriteString(cyan(schedule.ProfileLabel(view.Profile)))
	builder.WriteString("\n")

	return builder.String()
}

func renderScheduleDay(view schedule.DayView) string {
	var builder strings.Builder

	builder.WriteString(renderScheduleHeader(view))

	if view.IsToday {
		builder.WriteString("\n")
		builder.WriteString(renderScheduleSummary(view))
	}

	if len(view.Entries) == 0 {
		builder.WriteString("\n")
		builder.WriteString(yellow("No schedule items found for this day."))
		builder.WriteString("\n")
		return builder.String()
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Tag", "Time", "Block", "Type", "Course", "Room", "Teacher"})

	for _, entry := range view.Entries {
		tag := ""
		rowColor := func(text string) string { return text }
		if entry.IsFreeBlock {
			rowColor = gray
		}

		switch {
		case isSameEntry(view.Current, entry):
			tag = bold(green("NOW"))
			rowColor = func(text string) string { return bold(green(text)) }
		case isSameEntry(view.Next, entry):
			tag = bold(cyan("NEXT"))
			rowColor = func(text string) string { return bold(cyan(text)) }
		}

		typeLabel := entry.TypeLabel()
		switch {
		case entry.IsFreeBlock:
			typeLabel = gray(typeLabel)
		case entry.ScheduleType == 3:
			typeLabel = magenta(typeLabel)
		case entry.ScheduleType == 4:
			typeLabel = blue(typeLabel)
		}

		room := asciiDisplayText(entry.Location)
		if room == "" {
			room = gray("-")
		}
		teacher := asciiDisplayText(entry.TeacherSummary())
		if teacher == "" {
			teacher = gray("-")
		}

		t.AppendRow(table.Row{
			tag,
			rowColor(entry.TimeRange()),
			rowColor(entry.BlockLabel(view.Profile)),
			typeLabel,
			rowColor(asciiDisplayText(entry.CourseName())),
			room,
			teacher,
		})
	}

	builder.WriteString("\n")
	builder.WriteString(t.Render())
	builder.WriteString("\n")
	return builder.String()
}

func renderScheduleFocus(view schedule.DayView, mode focusMode) string {
	var builder strings.Builder

	builder.WriteString(renderScheduleHeader(view))
	builder.WriteString("\n")

	switch mode {
	case focusNext:
		if view.Next == nil {
			if view.Current != nil {
				builder.WriteString(yellow("No later classes today."))
				builder.WriteString("\n\n")
				builder.WriteString(renderScheduleCard("NOW", *view.Current, view.Profile, green))
				return builder.String()
			}

			builder.WriteString(yellow("No upcoming classes today."))
			builder.WriteString("\n")
			return builder.String()
		}

		builder.WriteString(renderScheduleCard("NEXT", *view.Next, view.Profile, cyan))
	default:
		if view.Current != nil {
			builder.WriteString(renderScheduleCard("NOW", *view.Current, view.Profile, green))
			if view.Next != nil {
				builder.WriteString("\n")
				builder.WriteString(gray("Up next:\n"))
				builder.WriteString(renderScheduleCard("NEXT", *view.Next, view.Profile, cyan))
			}
			return builder.String()
		}

		builder.WriteString(yellow("No class is in session right now."))
		builder.WriteString("\n")
		if view.Next != nil {
			builder.WriteString("\n")
			builder.WriteString(renderScheduleCard("NEXT", *view.Next, view.Profile, cyan))
		}
	}

	return builder.String()
}

func renderScheduleSummary(view schedule.DayView) string {
	var builder strings.Builder

	if view.Current != nil {
		builder.WriteString(renderScheduleCard("NOW", *view.Current, view.Profile, green))
	} else {
		builder.WriteString(yellow("BREAK"))
		builder.WriteString(" ")
		builder.WriteString("No class is in session right now.")
		builder.WriteString("\n")
	}

	if view.Next != nil {
		if view.Current != nil {
			builder.WriteString("\n")
		}
		builder.WriteString(renderScheduleCard("NEXT", *view.Next, view.Profile, cyan))
	} else if view.Current == nil {
		builder.WriteString(gray("No more classes scheduled for today."))
		builder.WriteString("\n")
	}

	return builder.String()
}

func renderScheduleCard(label string, entry schedule.Entry, profile string, colorize func(string) string) string {
	var builder strings.Builder

	builder.WriteString(colorize(bold(label)))
	builder.WriteString(" ")
	builder.WriteString(colorize(asciiDisplayText(entry.CourseName())))
	builder.WriteString("\n")
	builder.WriteString(gray("Time: "))
	builder.WriteString(entry.TimeRange())
	builder.WriteString("  ")
	builder.WriteString(gray("Block: "))
	builder.WriteString(entry.BlockLabel(profile))
	builder.WriteString("  ")
	builder.WriteString(gray("Type: "))
	builder.WriteString(entry.TypeLabel())
	builder.WriteString("\n")

	if entry.Location != "" {
		builder.WriteString(gray("Room: "))
		builder.WriteString(asciiDisplayText(entry.Location))
		builder.WriteString("\n")
	}
	if teachers := entry.TeacherSummary(); teachers != "" {
		builder.WriteString(gray("Teacher: "))
		builder.WriteString(asciiDisplayText(teachers))
		builder.WriteString("\n")
	}
	if remark := strings.TrimSpace(entry.Remark); remark != "" {
		builder.WriteString(gray("Remark: "))
		builder.WriteString(asciiDisplayText(remark))
		builder.WriteString("\n")
	}

	return builder.String()
}

func renderScheduleWeek(view schedule.WeekView) string {
	var builder strings.Builder

	end := view.Begin.AddDate(0, 0, 6)
	builder.WriteString(bold(cyan(view.Begin.Format("2006-01-02") + " ~ " + end.Format("2006-01-02"))))
	builder.WriteString("\n")
	builder.WriteString(gray("Profile: "))
	builder.WriteString(cyan(schedule.ProfileLabel(view.Profile)))
	builder.WriteString("\n\n")

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)

	header := table.Row{"Block"}
	var columnConfigs []table.ColumnConfig
	for idx := range view.Days {
		day := view.Days[idx]
		header = append(header, shortWeekdayLabel(day.Date.Weekday())+"\n"+day.Date.Format("01-02"))
		if view.Today == &view.Days[idx] {
			// Highlight today via go-pretty's own colors so the escape
			// sequences stay width-transparent to the table layout.
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Number:       idx + 2,
				ColorsHeader: text.Colors{text.FgCyan, text.Bold},
			})
		}
	}
	t.SetColumnConfigs(columnConfigs)
	t.AppendHeader(header)

	for order := 1; order <= 8; order++ {
		row := table.Row{schedule.Entry{Order: order}.BlockLabel(view.Profile)}
		for idx := range view.Days {
			row = append(row, weekCell(view.Days[idx], order))
		}
		t.AppendRow(row)
	}

	builder.WriteString(t.Render())
	builder.WriteString("\n")
	builder.WriteString(renderWeekOtherItems(view))
	return builder.String()
}

func weekCell(day schedule.DayView, order int) string {
	var lines []string
	for _, entry := range day.Entries {
		if entry.IsFreeBlock || entry.Order != order {
			continue
		}

		name := truncateForWeekCell(asciiDisplayText(entry.CourseName()))
		switch {
		case isSameEntry(day.Current, entry):
			name = bold(green(name))
		case entry.ScheduleType == 3:
			name = magenta(name)
		case entry.ScheduleType != 4:
			name = blue(name)
		}
		lines = append(lines, name)

		if room := truncateForWeekCell(asciiDisplayText(entry.Location)); room != "" {
			lines = append(lines, gray(room))
		}
	}

	if len(lines) == 0 {
		return gray("Free")
	}
	return strings.Join(lines, "\n")
}

// renderWeekOtherItems lists entries without a formal period slot (order 0 or
// beyond 8) that cannot fit the period grid.
func renderWeekOtherItems(view schedule.WeekView) string {
	lines := make([]string, 0)
	for idx := range view.Days {
		day := view.Days[idx]
		for _, entry := range day.Entries {
			if entry.IsFreeBlock || (entry.Order >= 1 && entry.Order <= 8) {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s %s %s",
				gray(day.Date.Format("01-02")),
				entry.TimeRange(),
				asciiDisplayText(entry.CourseName()),
			))
		}
	}
	if len(lines) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(yellow("Other items:"))
	builder.WriteString("\n")
	builder.WriteString(strings.Join(lines, "\n"))
	builder.WriteString("\n")
	return builder.String()
}

// truncateForWeekCell keeps week-grid cells narrow, cutting on a display-width
// budget so CJK names are truncated fairly. ASCII names prefer a word
// boundary when one sits near the cut.
func truncateForWeekCell(text string) string {
	if text == "" {
		return text
	}

	width := 0
	for idx, r := range text {
		runeWidth := 1
		if isWideRune(r) {
			runeWidth = 2
		}
		if width+runeWidth > weekCellMaxWidth-len(weekCellEllipsis) {
			return trimToWordBoundary(text[:idx]) + weekCellEllipsis
		}
		width += runeWidth
	}
	return text
}

// trimToWordBoundary falls back to the last separator when it is close to the
// cut point, so "AP Physics C Mec" becomes "AP Physics C" instead.
func trimToWordBoundary(prefix string) string {
	trimmed := strings.TrimRight(prefix, " ")
	if sep := strings.LastIndexAny(trimmed, " -/"); sep > 0 && len(trimmed)-sep <= 6 {
		return trimmed[:sep]
	}
	return trimmed
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0xA4CF, // CJK radicals through Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE4F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6:
		return true
	default:
		return false
	}
}

func isSameEntry(entry *schedule.Entry, candidate schedule.Entry) bool {
	if entry == nil {
		return false
	}

	return entry.ID == candidate.ID && entry.Start.Equal(candidate.Start)
}

func shortWeekdayLabel(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return "周一"
	case time.Tuesday:
		return "周二"
	case time.Wednesday:
		return "周三"
	case time.Thursday:
		return "周四"
	case time.Friday:
		return "周五"
	case time.Saturday:
		return "周六"
	default:
		return "周日"
	}
}

func weekdayLabel(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return "Monday / 周一"
	case time.Tuesday:
		return "Tuesday / 周二"
	case time.Wednesday:
		return "Wednesday / 周三"
	case time.Thursday:
		return "Thursday / 周四"
	case time.Friday:
		return "Friday / 周五"
	case time.Saturday:
		return "Saturday / 周六"
	default:
		return "Sunday / 周日"
	}
}
