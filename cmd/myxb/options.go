package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

type outputFormat string

const (
	formatHuman    outputFormat = "human"
	formatTable    outputFormat = "table"
	formatPlain    outputFormat = "plain"
	formatMarkdown outputFormat = "markdown"
	formatJSON     outputFormat = "json"
)

type gpaCommandOptions struct {
	ShowTasks        bool
	Clean            bool
	Format           outputFormat
	SemesterSelector string
	ExportTarget     string
	ExportEnabled    bool
}

func normalizeCLIArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	normalized := []string{args[0]}
	for idx := 1; idx < len(args); idx++ {
		arg := args[idx]

		switch arg {
		case "-f", "--formatted":
			normalized = append(normalized, arg)
			if idx+1 >= len(args) || strings.HasPrefix(args[idx+1], "-") {
				normalized = append(normalized, string(formatTable))
			} else {
				normalized = append(normalized, args[idx+1])
				idx++
			}
		case "-e", "--export":
			normalized = append(normalized, arg)
			if idx+1 >= len(args) || strings.HasPrefix(args[idx+1], "-") {
				normalized = append(normalized, "__desktop__")
			} else {
				normalized = append(normalized, args[idx+1])
				idx++
			}
		default:
			normalized = append(normalized, arg)
		}
	}

	return normalized
}

func parseGPACommandOptions(c *cli.Command) (gpaCommandOptions, error) {
	opts := gpaCommandOptions{
		ShowTasks:        c.Bool("tasks"),
		Clean:            c.Bool("clean"),
		Format:           formatHuman,
		SemesterSelector: strings.TrimSpace(c.String("semester")),
		ExportTarget:     strings.TrimSpace(c.String("export")),
	}

	format, err := parseOutputFormat(strings.TrimSpace(strings.ToLower(c.String("formatted"))))
	if err != nil {
		return gpaCommandOptions{}, err
	}
	opts.Format = format
	if opts.ExportTarget != "" {
		opts.ExportEnabled = true
	}

	return opts, nil
}

func parseOutputFormat(value string) (outputFormat, error) {
	switch value {
	case "":
		return formatHuman, nil
	case string(formatTable):
		return formatTable, nil
	case string(formatPlain):
		return formatPlain, nil
	case "md", string(formatMarkdown):
		return formatMarkdown, nil
	case string(formatJSON):
		return formatJSON, nil
	default:
		return "", fmt.Errorf("invalid format %q: use table, plain, markdown, or json", value)
	}
}

func (o gpaCommandOptions) showHumanChrome() bool {
	return !o.Clean
}

func (o gpaCommandOptions) suppressProgress() bool {
	return o.Clean
}
