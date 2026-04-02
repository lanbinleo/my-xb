package main

import (
	"context"
	"fmt"
	"myxb/internal/api"
	"myxb/internal/config"
	"myxb/internal/schedule"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"
)

func newScheduleCommand() *cli.Command {
	return &cli.Command{
		Name:    "schedule",
		Aliases: []string{"s"},
		Usage:   "View today's classes, upcoming lessons, and daily timetables",
		Flags:   scheduleDayFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runScheduleDayCommand(c, strings.TrimSpace(c.String("date")))
		},
		Commands: []*cli.Command{
			{
				Name:    "now",
				Aliases: []string{"n"},
				Usage:   "Show the class happening right now",
				Flags:   scheduleCommonFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleNowCommand(c)
				},
			},
			{
				Name:    "next",
				Aliases: []string{"ne"},
				Usage:   "Show the next class for today",
				Flags:   scheduleCommonFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return runScheduleNextCommand(c)
				},
			},
			{
				Name:      "day",
				Aliases:   []string{"d"},
				Usage:     "Show the timetable for a date or weekday",
				ArgsUsage: "[date-or-weekday]",
				Flags:     scheduleDayFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					selector := strings.TrimSpace(c.Args().First())
					if selector == "" {
						selector = strings.TrimSpace(c.String("date"))
					}
					return runScheduleDayCommand(c, selector)
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
		Usage:   "Date or weekday to view: YYYY-MM-DD, monday, 周一, today, or tomorrow",
	})
	return flags
}

func runScheduleNowCommand(c *cli.Command) error {
	apiClient := requireScheduleAPIClient()
	service, err := newScheduleService(apiClient)
	if err != nil {
		return err
	}
	profile, err := resolveScheduleProfile(c.String("profile"))
	if err != nil {
		return err
	}

	view, err := service.GetDayView(service.Today(), profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	fmt.Print(renderScheduleFocus(view, "now"))
	return nil
}

func runScheduleNextCommand(c *cli.Command) error {
	apiClient := requireScheduleAPIClient()
	service, err := newScheduleService(apiClient)
	if err != nil {
		return err
	}
	profile, err := resolveScheduleProfile(c.String("profile"))
	if err != nil {
		return err
	}

	view, err := service.GetDayView(service.Today(), profile, c.Bool("refresh"))
	if err != nil {
		return err
	}

	fmt.Print(renderScheduleFocus(view, "next"))
	return nil
}

func runScheduleDayCommand(c *cli.Command, selector string) error {
	apiClient := requireScheduleAPIClient()
	service, err := newScheduleService(apiClient)
	if err != nil {
		return err
	}
	profile, err := resolveScheduleProfile(c.String("profile"))
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

		fmt.Printf("%s %s\n", bold("Saved schedule profile:"), yellow("Not set"))
		fmt.Printf("%s\n", gray("Set one before using timetable commands:"))
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

func requireScheduleAPIClient() *api.API {
	apiClient, err := ensureLogin(true)
	if err != nil {
		printError("You need to login first")
		fmt.Println()
		printInfo("Run: myxb login")
		os.Exit(1)
	}

	printSuccess("Authentication successful!")
	fmt.Println()
	return apiClient
}

func newScheduleService(apiClient *api.API) (*schedule.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	accountKey := ""
	if cfg != nil {
		accountKey = cfg.Username
	}

	return schedule.NewService(apiClient, accountKey), nil
}

func resolveScheduleProfile(rawOverride string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	configuredProfile, ok := cfg.ConfiguredScheduleProfile()
	if !ok {
		return "", fmt.Errorf("schedule profile not set: run 'myxb schedule profile standard' or 'myxb schedule profile highschool'")
	}

	if strings.TrimSpace(rawOverride) != "" {
		profile := config.NormalizeScheduleProfile(rawOverride)
		if profile == "" {
			return "", fmt.Errorf("invalid schedule profile %q: use standard or highschool", rawOverride)
		}
		return profile, nil
	}

	return configuredProfile, nil
}

func renderScheduleDay(view schedule.DayView) string {
	var builder strings.Builder

	builder.WriteString(bold(cyan(view.Date.Format("2006-01-02"))))
	builder.WriteString(" ")
	builder.WriteString(gray(weekdayLabel(view.Date.Weekday())))
	builder.WriteString("\n")
	builder.WriteString(gray("Profile: "))
	builder.WriteString(cyan(schedule.ProfileLabel(view.Profile)))
	builder.WriteString("\n")

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

func renderScheduleFocus(view schedule.DayView, mode string) string {
	var builder strings.Builder

	builder.WriteString(bold(cyan(view.Date.Format("2006-01-02"))))
	builder.WriteString(" ")
	builder.WriteString(gray(weekdayLabel(view.Date.Weekday())))
	builder.WriteString("\n")
	builder.WriteString(gray("Profile: "))
	builder.WriteString(cyan(schedule.ProfileLabel(view.Profile)))
	builder.WriteString("\n\n")

	switch mode {
	case "next":
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

func isSameEntry(entry *schedule.Entry, candidate schedule.Entry) bool {
	if entry == nil {
		return false
	}

	return entry.ID == candidate.ID && entry.Start.Equal(candidate.Start)
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
