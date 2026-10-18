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

// upcomingLookaheadDays bounds how far now/next search for the next class
// once today is over (covers weekends and short holidays).
const upcomingLookaheadDays = 14

const (
	weekUsage     = "Show the weekly timetable (on a weekend without classes, shows next week)"
	weekArgsUsage = "[+1 | -1 | next | last | this | YYYY-MM-DD | weekday]"
)

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
				Usage:   "Show the next class, looking ahead to later days when today is done",
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
				Usage:     weekUsage,
				ArgsUsage: weekArgsUsage,
				Flags:     scheduleDayFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleWeekCommand(c, scheduleSelectorArg(c))
				},
			},
			newScheduleICSCommand(),
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
			Name:    "now",
			Aliases: []string{"n"},
			Usage:   "Show the class happening right now (shortcut for 'schedule now')",
			Flags:   scheduleCommonFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleFocusCommand(c, focusNow)
			},
		},
		{
			Name:  "next",
			Usage: "Show the next class, looking ahead to later days (shortcut for 'schedule next')",
			Flags: scheduleCommonFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleFocusCommand(c, focusNext)
			},
		},
		{
			Name:      "day",
			Aliases:   []string{"d"},
			Usage:     "Show the timetable for a date or weekday (shortcut for 'schedule day')",
			ArgsUsage: "[date-or-weekday]",
			Flags:     scheduleDayFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleDayCommand(c, scheduleSelectorArg(c))
			},
		},
		{
			Name:      "week",
			Aliases:   []string{"w"},
			Usage:     weekUsage + " (shortcut for 'schedule week')",
			ArgsUsage: weekArgsUsage,
			Flags:     scheduleDayFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				return runScheduleWeekCommand(c, scheduleSelectorArg(c))
			},
		},
		newScheduleICSShortcutCommand(),
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
		Usage:   "Date or weekday to view: YYYY-MM-DD, monday, 周一, today, tomorrow, +2, or 下周五",
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

	var upcoming *schedule.Entry
	if view.Next == nil {
		upcoming, err = service.UpcomingEntry(profile, c.Bool("refresh"), upcomingLookaheadDays)
		if err != nil {
			return err
		}
	}

	fmt.Print(renderScheduleFocus(view, mode, upcoming, service.Now()))
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

	var target time.Time
	advanced := false
	if selector == "" {
		target, advanced, err = service.DefaultWeekTarget(profile, c.Bool("refresh"))
	} else {
		target, err = service.ResolveWeek(selector)
	}
	if err != nil {
		return err
	}

	view, err := service.GetWeekView(target, profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	if advanced {
		printInfo(gray("No classes this weekend, showing next week (this week: myxb week this)"))
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
			apiClient, err := ensureLogin(false)
			if err != nil {
				return nil, fmt.Errorf("login failed: %w (run 'myxb login' again)", err)
			}
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

// renderScheduleFocus shows the live class and what comes after it. upcoming
// is the first class on a later day, used once today has nothing left.
func renderScheduleFocus(view schedule.DayView, mode focusMode, upcoming *schedule.Entry, now time.Time) string {
	var builder strings.Builder

	builder.WriteString(renderScheduleHeader(view))
	builder.WriteString("\n")

	writeNext := func() {
		switch {
		case view.Next != nil:
			builder.WriteString(renderScheduleCard("NEXT", *view.Next, view.Profile, cyan))
			builder.WriteString(gray("Starts in " + humanDuration(view.Next.Start.Sub(now))))
			builder.WriteString("\n")
		case upcoming != nil:
			builder.WriteString(gray("No more classes today."))
			builder.WriteString("\n\n")
			label := "NEXT " + shortWeekdayLabel(upcoming.Start.Weekday()) + " " + upcoming.Start.Format("01-02")
			builder.WriteString(renderScheduleCard(label, *upcoming, view.Profile, cyan))
			builder.WriteString(gray("Starts in " + humanDuration(upcoming.Start.Sub(now))))
			builder.WriteString("\n")
		default:
			builder.WriteString(yellow("No upcoming classes found in the next two weeks."))
			builder.WriteString("\n")
		}
	}

	if view.Current != nil {
		builder.WriteString(renderScheduleCard("NOW", *view.Current, view.Profile, green))
		builder.WriteString(gray("Ends in " + humanDuration(view.Current.End.Sub(now))))
		builder.WriteString("\n")
		builder.WriteString("\n")
		writeNext()
		return builder.String()
	}

	if mode == focusNow {
		builder.WriteString(yellow("No class is in session right now."))
		builder.WriteString("\n\n")
	}
	writeNext()
	return builder.String()
}

// humanDuration renders a coarse countdown such as "23 min", "1 h 05 min",
// or "2 d 3 h".
func humanDuration(d time.Duration) string {
	minutes := int((d + time.Minute - 1) / time.Minute)
	switch {
	case minutes < 1:
		return "less than a minute"
	case minutes < 60:
		return fmt.Sprintf("%d min", minutes)
	case minutes < 24*60:
		return fmt.Sprintf("%d h %02d min", minutes/60, minutes%60)
	default:
		return fmt.Sprintf("%d d %d h", minutes/(24*60), minutes%(24*60)/60)
	}
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
	if label := weekRelativeLabel(view); label != "" {
		builder.WriteString(" ")
		builder.WriteString(gray(label))
	}
	builder.WriteString("\n")
	builder.WriteString(gray("Profile: "))
	builder.WriteString(cyan(schedule.ProfileLabel(view.Profile)))
	builder.WriteString("\n\n")

	// Weekend columns only appear when they hold something (e.g. 调休).
	days := make([]int, 0, len(view.Days))
	for idx := range view.Days {
		weekday := view.Days[idx].Date.Weekday()
		if (weekday == time.Saturday || weekday == time.Sunday) && !dayHasRealEntries(view.Days[idx]) {
			continue
		}
		days = append(days, idx)
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatDefault
	// Row separators make merged double periods read as one block.
	t.Style().Options.SeparateRows = true

	header := table.Row{"Block"}
	columnConfigs := []table.ColumnConfig{{Number: 1, Colors: text.Colors{text.FgHiBlack}}}
	for column, idx := range days {
		day := view.Days[idx]
		header = append(header, shortWeekdayLabel(day.Date.Weekday())+" "+day.Date.Format("01-02"))
		config := table.ColumnConfig{Number: column + 2, AutoMerge: true}
		if view.Today == &view.Days[idx] {
			// Highlight today via go-pretty's own colors so the escape
			// sequences stay width-transparent to the table layout.
			config.ColorsHeader = text.Colors{text.FgCyan, text.Bold}
		}
		columnConfigs = append(columnConfigs, config)
	}
	t.SetColumnConfigs(columnConfigs)
	t.AppendHeader(header)

	for order := 1; order <= 8; order++ {
		label := schedule.Entry{Order: order}.BlockLabel(view.Profile)
		if start, _, ok := schedule.BlockTime(order, view.Profile); ok {
			label += " " + start
		}
		row := table.Row{label}
		for _, idx := range days {
			row = append(row, weekCell(view.Days[idx], order))
		}
		t.AppendRow(row)
	}

	builder.WriteString(t.Render())
	builder.WriteString("\n")
	builder.WriteString(renderWeekOtherItems(view))
	return builder.String()
}

// weekRelativeLabel names the week relative to today.
func weekRelativeLabel(view schedule.WeekView) string {
	switch view.Offset {
	case 0:
		return "(this week)"
	case 1:
		return "(next week)"
	case -1:
		return "(last week)"
	default:
		return fmt.Sprintf("(%+d weeks)", view.Offset)
	}
}

func dayHasRealEntries(day schedule.DayView) bool {
	for _, entry := range day.Entries {
		if !entry.IsFreeBlock {
			return true
		}
	}
	return false
}

// weekCell renders one grid cell. Consecutive identical cells in a column are
// merged by the table, so a double period shows up once.
func weekCell(day schedule.DayView, order int) string {
	var lines []string
	for _, entry := range day.Entries {
		if entry.IsFreeBlock || entry.Order != order {
			continue
		}

		name := truncateForWeekCell(abbreviateCourseName(asciiDisplayText(entry.CourseName())))
		switch {
		case isSameEntry(day.Current, entry):
			name = bold(green("> " + name))
		case entry.ScheduleType == 3:
			name = magenta(name)
		case entry.ScheduleType != 4:
			name = yellow(name)
		}
		lines = append(lines, name)

		if room := truncateForWeekCell(asciiDisplayText(entry.Location)); room != "" {
			lines = append(lines, gray(room))
		}
	}

	if len(lines) == 0 {
		return gray("-")
	}
	return strings.Join(lines, "\n")
}

// renderWeekOtherItems lists entries without a formal period slot (order 0 or
// beyond 8) that cannot fit the period grid. Back-to-back sessions of the same
// item on one day collapse into a single time range.
func renderWeekOtherItems(view schedule.WeekView) string {
	lines := make([]string, 0)
	for idx := range view.Days {
		day := view.Days[idx]
		var items []schedule.Entry
		for _, entry := range day.Entries {
			if entry.IsFreeBlock || (entry.Order >= 1 && entry.Order <= 8) {
				continue
			}
			if last := len(items) - 1; last >= 0 && items[last].CourseName() == entry.CourseName() &&
				!entry.Start.After(items[last].End.Add(15*time.Minute)) {
				if entry.End.After(items[last].End) {
					items[last].End = entry.End
				}
				continue
			}
			items = append(items, entry)
		}

		for _, entry := range items {
			line := fmt.Sprintf("  %s %s  %s",
				gray(shortWeekdayLabel(day.Date.Weekday())+" "+day.Date.Format("01-02")),
				entry.TimeRange(),
				asciiDisplayText(entry.CourseName()),
			)
			if room := asciiDisplayText(entry.Location); room != "" {
				line += gray(" @ " + room)
			}
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n")
	builder.WriteString(yellow("Other items:"))
	builder.WriteString("\n")
	builder.WriteString(strings.Join(lines, "\n"))
	builder.WriteString("\n")
	return builder.String()
}

// courseNameReplacements shortens common long words so week-grid cells keep
// the distinguishing part of a course name visible.
var courseNameReplacements = []struct{ long, short string }{
	{"Literature and Composition", "Lit"},
	{"Language and Composition", "Lang"},
	{"Literature", "Lit"},
	{"Language", "Lang"},
	{"Composition", "Comp"},
	{"English", "Eng"},
	{"Chemistry", "Chem"},
	{"Physics", "Phys"},
	{"Biology", "Bio"},
	{"Mathematics", "Math"},
	{"Statistics", "Stats"},
	{"Calculus", "Calc"},
	{"Psychology", "Psych"},
	{"Economics", "Econ"},
	{"Macroeconomics", "Macro"},
	{"Microeconomics", "Micro"},
	{"Computer Science", "CS"},
	{"Linear Algebra", "Lin Alg"},
	{"Multivariable", "Multivar"},
	{"Culture", "Cult"},
	{"History", "Hist"},
	{"Government", "Gov"},
	{"Geography", "Geo"},
	{"Environmental", "Env"},
	{"Science", "Sci"},
	{"Physical Education", "PE"},
	{" and ", " & "},
}

// abbreviateCourseName applies the replacements only when the name would not
// fit a week cell, so short names stay untouched.
func abbreviateCourseName(name string) string {
	if displayWidth(name) <= weekCellMaxWidth {
		return name
	}
	for _, r := range courseNameReplacements {
		name = strings.ReplaceAll(name, r.long, r.short)
	}
	return name
}

func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		if isWideRune(r) {
			width += 2
		} else {
			width++
		}
	}
	return width
}

// truncateForWeekCell keeps week-grid cells narrow, cutting on a display-width
// budget so CJK names are truncated fairly. ASCII names prefer a word
// boundary when one sits near the cut.
func truncateForWeekCell(text string) string {
	if displayWidth(text) <= weekCellMaxWidth {
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
