package ics

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func testStamp() time.Time {
	return time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
}

func contentLines(rendered string) []string {
	return strings.Split(strings.TrimSuffix(rendered, "\r\n"), "\r\n")
}

func TestRenderCalendarStructure(t *testing.T) {
	calendar := Calendar{
		Name: "MyXB Schedule 2026-09-14~2026-09-20 (Standard)",
		Events: []Event{{
			UID:      "myxb-42-1768416000@myxb",
			Summary:  "数学 Math",
			Location: "Room 301",
			Start:    testStamp(),
			End:      testStamp().Add(40 * time.Minute),
		}},
	}

	got := contentLines(calendar.Render(testStamp()))

	want := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//myxb//Schedule//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"X-WR-CALNAME:MyXB Schedule 2026-09-14~2026-09-20 (Standard)",
		"BEGIN:VEVENT",
		"UID:myxb-42-1768416000@myxb",
		"DTSTAMP:20260914T080000Z",
		"DTSTART:20260914T080000Z",
		"DTEND:20260914T084000Z",
		"SUMMARY:数学 Math",
		"LOCATION:Room 301",
		"END:VEVENT",
		"END:VCALENDAR",
	}
	if len(got) != len(want) {
		t.Fatalf("calendar rendered %d lines, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Errorf("line %d = %q, want %q", idx, got[idx], want[idx])
		}
	}
}

func TestRenderSkipsEmptyOptionalProperties(t *testing.T) {
	event := Event{
		UID:   "myxb-1-0@myxb",
		Start: testStamp(),
		End:   testStamp().Add(time.Hour),
	}

	rendered := Calendar{Events: []Event{event}}.Render(testStamp())
	if strings.Contains(rendered, "SUMMARY:") {
		t.Errorf("empty summary should be omitted:\n%s", rendered)
	}
	if strings.Contains(rendered, "DESCRIPTION:") || strings.Contains(rendered, "LOCATION:") {
		t.Errorf("empty description/location should be omitted:\n%s", rendered)
	}
	if strings.Contains(rendered, "X-WR-CALNAME") {
		t.Errorf("empty calendar name should be omitted:\n%s", rendered)
	}
}

func TestEscapeText(t *testing.T) {
	cases := map[string]string{
		"plain":            "plain",
		"a;b":              `a\;b`,
		"a,b":              `a\,b`,
		`back\slash`:       `back\\slash`,
		"line1\nline2":     `line1\nline2`,
		"line1\r\nline2":   `line1\nline2`,
		"cr\ronly":         `cr\nonly`,
		`semi;comma,both\`: `semi\;comma\,both\\`,
	}
	for input, want := range cases {
		if got := escapeText(input); got != want {
			t.Errorf("escapeText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderEscapesTextProperties(t *testing.T) {
	event := Event{
		UID:         "uid",
		Summary:     "Drama, Rehearsal;",
		Description: "Teacher: Li\nRoom: 301",
		Location:    `A\B`,
		Start:       testStamp(),
		End:         testStamp().Add(time.Hour),
	}

	rendered := Calendar{Events: []Event{event}}.Render(testStamp())
	for _, want := range []string{
		"SUMMARY:Drama\\, Rehearsal\\;",
		"DESCRIPTION:Teacher: Li\\nRoom: 301",
		`LOCATION:A\\B`,
	} {
		if !strings.Contains(rendered, want+"\r\n") {
			t.Errorf("rendered calendar missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderConvertsToUTC(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("Asia/Shanghai timezone unavailable: %v", err)
	}

	event := Event{
		UID:   "uid",
		Start: time.Date(2026, 9, 14, 8, 0, 0, 0, shanghai),
		End:   time.Date(2026, 9, 14, 8, 40, 0, 0, shanghai),
	}

	rendered := Calendar{Events: []Event{event}}.Render(testStamp())
	if !strings.Contains(rendered, "DTSTART:20260914T000000Z\r\n") {
		t.Errorf("DTSTART not converted to UTC:\n%s", rendered)
	}
	if !strings.Contains(rendered, "DTEND:20260914T004000Z\r\n") {
		t.Errorf("DTEND not converted to UTC:\n%s", rendered)
	}
}

func TestFoldLineBoundsAndUTF8Safety(t *testing.T) {
	// A long CJK summary plus a long property name forces multiple folds.
	summary := strings.Repeat("语", 100) // 300 octets
	line := "SUMMARY:" + summary

	chunks := foldLine(line)

	if len(chunks) < 2 {
		t.Fatalf("expected folding, got %d chunk(s)", len(chunks))
	}
	for idx, chunk := range chunks {
		limit := firstLineOctets
		if idx > 0 {
			limit = followLineOctets
		}
		if len(chunk) > limit {
			t.Errorf("chunk %d is %d octets, want <= %d", idx, len(chunk), limit)
		}
		if !utf8.ValidString(chunk) {
			t.Errorf("chunk %d splits a UTF-8 rune: %q", idx, chunk)
		}
	}

	// Re-joining the chunks must restore the original line: unfolding removes
	// the CRLF plus the single leading space of each continuation line.
	joined := chunks[0]
	for _, chunk := range chunks[1:] {
		joined += chunk
	}
	if joined != line {
		t.Errorf("unfolded chunks = %q, want %q", joined, line)
	}
}

func TestRenderFoldsLongLinesWithCRLF(t *testing.T) {
	event := Event{
		UID:     "uid",
		Summary: strings.Repeat("语", 100),
		Start:   testStamp(),
		End:     testStamp().Add(time.Hour),
	}

	rendered := Calendar{Events: []Event{event}}.Render(testStamp())
	if !strings.HasSuffix(rendered, "\r\n") {
		t.Error("calendar should end with CRLF")
	}
	for idx, line := range contentLines(rendered) {
		if strings.HasPrefix(line, " ") {
			continue // continuation lines carry one leading space
		}
		if len(line) > firstLineOctets {
			t.Errorf("line %d exceeds %d octets: %q", idx, firstLineOctets, line)
		}
	}
}
