package main

import (
	"context"
	"fmt"
	"myxb/internal/attendance"
	"myxb/internal/models"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"
)

func newAttendanceCommand() *cli.Command {
	return &cli.Command{
		Name:    "attendance",
		Aliases: []string{"att"},
		Usage:   "Show your attendance rate for the current school year",
		Action: func(ctx context.Context, c *cli.Command) error {
			return runAttendanceCommand()
		},
	}
}

func runAttendanceCommand() error {
	apiClient, err := ensureLogin(true)
	if err != nil {
		return fmt.Errorf("not logged in: run 'myxb login' first")
	}
	printSuccess("Authentication successful!")
	fmt.Println()

	view, err := attendance.NewService(apiClient).GetYearView()
	if err != nil {
		return err
	}

	fmt.Print(renderAttendanceView(view))
	return nil
}

func renderAttendanceView(view *attendance.YearView) string {
	var builder strings.Builder

	builder.WriteString(bold(cyan(fmt.Sprintf("%s ~ %s",
		view.Begin.Format("2006-01-02"),
		view.End.Format("2006-01-02"),
	))))
	builder.WriteString(" ")
	builder.WriteString(gray(fmt.Sprintf("School Year %d-%d", view.Year, view.Year+1)))
	builder.WriteString("\n\n")

	rate := view.Overall.AttendanceRate
	builder.WriteString(bold("Attendance Rate: "))
	builder.WriteString(bold(colorizeAttendanceRate(rate, fmt.Sprintf("%.2f%%", rate))))
	builder.WriteString("\n")
	builder.WriteString(renderAttendanceStateCounts(&view.Overall))

	if len(view.Subjects) == 0 {
		builder.WriteString("\n")
		builder.WriteString(yellow("No subject attendance records found for this school year."))
		builder.WriteString("\n")
		return builder.String()
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Subject", "Total", "Normal", "Late", "Early", "Absent", "Rate"})

	for _, row := range view.Subjects {
		absent := fmt.Sprintf("%d", row.Absent)
		if row.Absent > 0 {
			absent = red(absent)
		}

		t.AppendRow(table.Row{
			asciiDisplayText(row.Name),
			row.Total,
			row.Normal,
			row.Late,
			row.EarlyLeave,
			absent,
			colorizeAttendanceRate(row.RatePercent, fmt.Sprintf("%.2f%%", row.RatePercent)),
		})
	}

	builder.WriteString("\n")
	builder.WriteString(t.Render())
	builder.WriteString("\n")
	return builder.String()
}

func renderAttendanceStateCounts(overall *models.AttendanceStatisticData) string {
	counts := make([]models.AttendanceStateCount, len(overall.StateCountList))
	copy(counts, overall.StateCountList)
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].AttendanceState < counts[j].AttendanceState
	})

	parts := make([]string, 0, len(counts)+1)
	parts = append(parts, fmt.Sprintf("%s%d", gray("Total: "), overall.TotalCount))
	for _, stateCount := range counts {
		label := attendanceStateLabel(stateCount.AttendanceState)
		parts = append(parts, fmt.Sprintf("%s%d", gray(label+": "), stateCount.Count))
	}

	return strings.Join(parts, "   ") + "\n"
}

// attendanceStateLabel names the known attendance states; unknown ones stay
// conservative instead of guessing.
func attendanceStateLabel(state int) string {
	switch state {
	case 0:
		return "Normal"
	case 3:
		return "Absent"
	default:
		return fmt.Sprintf("State %d", state)
	}
}

func colorizeAttendanceRate(rate float64, text string) string {
	switch {
	case rate >= 95:
		return green(text)
	case rate >= 90:
		return cyan(text)
	case rate >= 80:
		return yellow(text)
	default:
		return red(text)
	}
}
