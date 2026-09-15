// Package ics renders schedule data as an RFC 5545 iCalendar file.
package ics

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	utcTimeLayout = "20060102T150405Z"
	// RFC 5545 caps content lines at 75 octets; continuation lines spend one
	// octet on the leading space.
	firstLineOctets  = 75
	followLineOctets = 74
)

// Event is a single VEVENT with UTC timestamps.
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
}

// Calendar is a VCALENDAR wrapper around a set of events.
type Calendar struct {
	Name   string
	Events []Event
}

// Render writes the whole calendar. now becomes the DTSTAMP of every event.
func (c Calendar) Render(now time.Time) string {
	var builder strings.Builder

	writeProperty := func(line string) {
		for idx, chunk := range foldLine(line) {
			if idx > 0 {
				builder.WriteString("\r\n ")
			}
			builder.WriteString(chunk)
		}
		builder.WriteString("\r\n")
	}

	writeProperty("BEGIN:VCALENDAR")
	writeProperty("VERSION:2.0")
	writeProperty("PRODID:-//myxb//Schedule//EN")
	writeProperty("CALSCALE:GREGORIAN")
	writeProperty("METHOD:PUBLISH")
	if c.Name != "" {
		writeProperty("X-WR-CALNAME:" + escapeText(c.Name))
	}
	for _, event := range c.Events {
		for _, line := range event.lines(now) {
			writeProperty(line)
		}
	}
	writeProperty("END:VCALENDAR")

	return builder.String()
}

func (e Event) lines(now time.Time) []string {
	lines := []string{
		"BEGIN:VEVENT",
		"UID:" + escapeText(e.UID),
		"DTSTAMP:" + now.UTC().Format(utcTimeLayout),
		"DTSTART:" + e.Start.UTC().Format(utcTimeLayout),
		"DTEND:" + e.End.UTC().Format(utcTimeLayout),
	}
	if e.Summary != "" {
		lines = append(lines, "SUMMARY:"+escapeText(e.Summary))
	}
	if e.Location != "" {
		lines = append(lines, "LOCATION:"+escapeText(e.Location))
	}
	if e.Description != "" {
		lines = append(lines, "DESCRIPTION:"+escapeText(e.Description))
	}
	return append(lines, "END:VEVENT")
}

// escapeText applies RFC 5545 text escaping to property values.
func escapeText(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		";", "\\;",
		",", "\\,",
		"\r\n", "\\n",
		"\n", "\\n",
		"\r", "\\n",
	)
	return replacer.Replace(value)
}

// foldLine splits a content line into octet-bounded chunks that never cut a
// UTF-8 rune in half.
func foldLine(line string) []string {
	var chunks []string
	limit := firstLineOctets
	for len(line) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		if cut == 0 {
			// Defensive: a rune wider than the whole budget. Emit it whole
			// instead of splitting a UTF-8 sequence.
			cut = limit
			for cut < len(line) && !utf8.RuneStart(line[cut]) {
				cut++
			}
		}
		chunks = append(chunks, line[:cut])
		line = line[cut:]
		limit = followLineOctets
	}
	return append(chunks, line)
}
