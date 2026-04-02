package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func maybeExportOutput(content string, reports []semesterReport, opts gpaCommandOptions) error {
	if !opts.ExportEnabled {
		return nil
	}

	exportPath, err := resolveExportPath(reports, opts)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(exportPath), 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	if err := os.WriteFile(exportPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to export output: %w", err)
	}

	if opts.showHumanChrome() {
		fmt.Println()
		printSuccess("Exported output to: " + exportPath)
	}

	return nil
}

func resolveExportPath(reports []semesterReport, opts gpaCommandOptions) (string, error) {
	target := opts.ExportTarget
	defaultDesktopTarget := target == "" || target == "__desktop__"
	if defaultDesktopTarget {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", err)
		}
		target = filepath.Join(home, "Desktop")
	}

	info, err := os.Stat(target)
	switch {
	case err == nil && info.IsDir():
		return filepath.Join(target, defaultExportFilename(reports, opts)), nil
	case err == nil && !info.IsDir():
		return target, nil
	case os.IsNotExist(err):
		if defaultDesktopTarget || isExplicitDirectoryPath(target) {
			return filepath.Join(target, defaultExportFilename(reports, opts)), nil
		}
		return target, nil
	default:
		return "", err
	}
}

func defaultExportFilename(reports []semesterReport, opts gpaCommandOptions) string {
	stamp := time.Now().Format("20060102_150405")
	scope := "report"
	if len(reports) == 1 {
		scope = fmt.Sprintf("%d-%d_s%d", reports[0].Semester.Year, reports[0].Semester.Year+1, reports[0].Semester.Semester)
	} else if len(reports) > 1 {
		scope = fmt.Sprintf("%d_semesters", len(reports))
	}

	formatName := "human"
	if opts.Format != "" && opts.Format != formatHuman {
		formatName = string(opts.Format)
	}

	return fmt.Sprintf("myxb_%s_%s_%s.txt", scope, formatName, stamp)
}

func isExplicitDirectoryPath(path string) bool {
	return strings.HasSuffix(path, string(filepath.Separator)) || strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")
}
