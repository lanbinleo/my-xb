package main

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func asciiDisplayText(text string) string {
	decomposed := norm.NFD.String(text)

	var out strings.Builder
	out.Grow(len(decomposed))

	for _, r := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r == '¿':
			out.WriteByte('?')
		case r == '¡':
			out.WriteByte('!')
		case r == '\u2018' || r == '\u2019' || r == '\u2032':
			out.WriteByte('\'')
		case r == '\u201C' || r == '\u201D' || r == '\u2033':
			out.WriteByte('"')
		case r == '\u2013' || r == '\u2014':
			out.WriteByte('-')
		case r == '\u2026':
			out.WriteString("...")
		case r == '\u00A0':
			out.WriteByte(' ')
		case r <= unicode.MaxASCII && !unicode.IsControl(r):
			out.WriteRune(r)
		case unicode.IsControl(r) || unicode.In(r, unicode.Cf):
			continue
		default:
			out.WriteRune(r)
		}
	}

	return out.String()
}
