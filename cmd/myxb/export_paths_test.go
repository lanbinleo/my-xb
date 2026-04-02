package main

import (
	"path/filepath"
	"testing"

	"myxb/internal/models"
)

func TestResolveExportPathTreatsNewPathAsFile(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "report")

	got, err := resolveExportPath(nil, gpaCommandOptions{ExportTarget: target})
	if err != nil {
		t.Fatalf("resolveExportPath returned error: %v", err)
	}
	if got != target {
		t.Fatalf("resolveExportPath = %q, want %q", got, target)
	}
}

func TestResolveExportPathTreatsExplicitDirectoryAsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "exports") + string(filepath.Separator)
	reports := []semesterReport{
		{Semester: models.Semester{Year: 2025, Semester: 1}},
	}

	got, err := resolveExportPath(reports, gpaCommandOptions{ExportTarget: target, Format: formatTable})
	if err != nil {
		t.Fatalf("resolveExportPath returned error: %v", err)
	}

	wantDir := filepath.Clean(target)
	if filepath.Dir(got) != wantDir {
		t.Fatalf("resolveExportPath directory = %q, want %q", filepath.Dir(got), wantDir)
	}
	if filepath.Base(got) == "exports" {
		t.Fatalf("resolveExportPath = %q, want auto-generated filename inside directory", got)
	}
}
