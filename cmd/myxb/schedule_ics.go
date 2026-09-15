package main

import (
	"context"
	"fmt"
	"myxb/internal/ics"
	"myxb/internal/schedule"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

func newScheduleICSCommand() *cli.Command {
	return &cli.Command{
		Name:      "ics",
		Aliases:   []string{"i"},
		Usage:     "Export a week of the timetable as an .ics calendar file",
		ArgsUsage: "[date-or-weekday]",
		Flags:     scheduleICSFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runScheduleICSCommand(c, scheduleSelectorArg(c))
		},
	}
}

// newScheduleICSShortcutCommand exposes the export as `myxb ics` without the
// schedule group name.
func newScheduleICSShortcutCommand() *cli.Command {
	command := newScheduleICSCommand()
	command.Aliases = nil
	command.Usage = "Export a week of the timetable as an .ics calendar file (shortcut for 'schedule ics')"
	return command
}

func scheduleICSFlags() []cli.Flag {
	flags := scheduleDayFlags()
	flags = append(flags,
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Write the .ics to this file or directory (defaults to the Desktop)",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "Include clubs, events, and off-grid items along with regular classes",
		},
	)
	return flags
}

func runScheduleICSCommand(c *cli.Command, selector string) error {
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

	events := weekViewToEvents(view, c.Bool("all"))
	if len(events) == 0 {
		end := view.Begin.AddDate(0, 0, 6)
		return fmt.Errorf("no schedule entries to export for %s ~ %s", view.Begin.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	outputPath, err := resolveICSOutputPath(c.String("output"), view)
	if err != nil {
		return err
	}

	calendar := ics.Calendar{Name: icsCalendarName(view), Events: events}
	content := calendar.Render(time.Now())

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write calendar file: %w", err)
	}

	scope := "regular classes only"
	if c.Bool("all") {
		scope = "all items"
	}
	printSuccess(fmt.Sprintf("Exported %d events to: %s", len(events), outputPath))
	fmt.Println(gray(fmt.Sprintf(" - Week %s ~ %s, profile %s (%s)",
		view.Begin.Format("2006-01-02"),
		view.Begin.AddDate(0, 0, 6).Format("2006-01-02"),
		schedule.ProfileLabel(view.Profile),
		scope,
	)))
	return nil
}

// weekViewToEvents maps a week view to ICS events. Regular classes (formal
// blocks 1-8) are always included; includeAll additionally picks up clubs,
// one-off events, and off-grid items such as drama rehearsals.
func weekViewToEvents(view schedule.WeekView, includeAll bool) []ics.Event {
	events := make([]ics.Event, 0)
	for idx := range view.Days {
		for _, entry := range view.Days[idx].Entries {
			if entry.IsFreeBlock {
				continue
			}
			if !includeAll && !isRegularClassEntry(entry) {
				continue
			}
			events = append(events, entryToEvent(entry, view.Profile))
		}
	}
	return events
}

func isRegularClassEntry(entry schedule.Entry) bool {
	return entry.ScheduleType == 4 && entry.Order >= 1 && entry.Order <= 8
}

func entryToEvent(entry schedule.Entry, profile string) ics.Event {
	return ics.Event{
		UID:         fmt.Sprintf("myxb-%d-%d@myxb", entry.ID, entry.Start.Unix()),
		Summary:     entry.CourseName(),
		Location:    entry.Location,
		Description: entryDescription(entry, profile),
		Start:       entry.Start,
		End:         entry.End,
	}
}

func entryDescription(entry schedule.Entry, profile string) string {
	var parts []string
	if teachers := entry.TeacherSummary(); teachers != "" {
		parts = append(parts, "Teacher: "+teachers)
	}
	if entry.Order >= 1 {
		parts = append(parts, "Block: "+entry.BlockLabel(profile))
	}
	if !isRegularClassEntry(entry) {
		parts = append(parts, "Type: "+entry.TypeLabel())
	}
	if remark := strings.TrimSpace(entry.Remark); remark != "" {
		parts = append(parts, "Remark: "+remark)
	}
	return strings.Join(parts, "\n")
}

func icsCalendarName(view schedule.WeekView) string {
	return fmt.Sprintf("MyXB Schedule %s~%s (%s)",
		view.Begin.Format("2006-01-02"),
		view.Begin.AddDate(0, 0, 6).Format("2006-01-02"),
		schedule.ProfileLabel(view.Profile),
	)
}

func resolveICSOutputPath(target string, view schedule.WeekView) (string, error) {
	defaultName := fmt.Sprintf("myxb_schedule_%s_%s.ics",
		view.Begin.Format("20060102"),
		view.Begin.AddDate(0, 0, 6).Format("20060102"),
	)

	useDefault := strings.TrimSpace(target) == ""
	if useDefault {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", err)
		}
		target = filepath.Join(home, "Desktop")
	}

	info, err := os.Stat(target)
	switch {
	case err == nil && info.IsDir():
		return filepath.Join(target, defaultName), nil
	case err == nil:
		return target, nil
	case os.IsNotExist(err):
		if useDefault || isExplicitDirectoryPath(target) {
			return filepath.Join(target, defaultName), nil
		}
		return target, nil
	default:
		return "", err
	}
}
