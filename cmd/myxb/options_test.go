package main

import "testing"

func TestNormalizeCLIArgsBareFormattedFlagDefaultsToAI(t *testing.T) {
	args := []string{"myxb", "-f", "-t"}
	got := normalizeCLIArgs(args)
	want := []string{"myxb", "-f", "table", "-t"}

	if len(got) != len(want) {
		t.Fatalf("normalizeCLIArgs(%v) length = %d, want %d", args, len(got), len(want))
	}

	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("normalizeCLIArgs(%v)[%d] = %q, want %q", args, idx, got[idx], want[idx])
		}
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		input string
		want  outputFormat
		ok    bool
	}{
		{input: "", want: formatHuman, ok: true},
		{input: "table", want: formatTable, ok: true},
		{input: "plain", want: formatPlain, ok: true},
		{input: "md", want: formatMarkdown, ok: true},
		{input: "markdown", want: formatMarkdown, ok: true},
		{input: "json", want: formatJSON, ok: true},
		{input: "yaml", ok: false},
	}

	for _, tt := range tests {
		got, err := parseOutputFormat(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("parseOutputFormat(%q) returned unexpected error: %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("parseOutputFormat(%q) expected error", tt.input)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("parseOutputFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeCLIArgsBareExportFlagDefaultsToDesktopSentinel(t *testing.T) {
	args := []string{"myxb", "-e", "-f"}
	got := normalizeCLIArgs(args)
	want := []string{"myxb", "-e", "__desktop__", "-f", "table"}

	if len(got) != len(want) {
		t.Fatalf("normalizeCLIArgs(%v) length = %d, want %d", args, len(got), len(want))
	}

	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("normalizeCLIArgs(%v)[%d] = %q, want %q", args, idx, got[idx], want[idx])
		}
	}
}

func TestShowHumanChromeDependsOnlyOnClean(t *testing.T) {
	opts := gpaCommandOptions{Format: formatJSON}
	if !opts.showHumanChrome() {
		t.Fatalf("showHumanChrome() = false, want true when Clean is false")
	}
	if opts.suppressProgress() {
		t.Fatalf("suppressProgress() = true, want false when Clean is false")
	}

	opts.Clean = true
	if opts.showHumanChrome() {
		t.Fatalf("showHumanChrome() = true, want false when Clean is true")
	}
	if !opts.suppressProgress() {
		t.Fatalf("suppressProgress() = false, want true when Clean is true")
	}
}
