package main

import "testing"

func TestAsciiDisplayTextSpanish(t *testing.T) {
	input := "C1-Presentación: ¿Y Eso?"
	want := "C1-Presentacion: ?Y Eso?"

	if got := asciiDisplayText(input); got != want {
		t.Fatalf("asciiDisplayText(%q) = %q, want %q", input, got, want)
	}
}

func TestAsciiDisplayTextCombiningAccent(t *testing.T) {
	input := "C2-¿Por que\u0301? Porque..."
	want := "C2-?Por que? Porque..."

	if got := asciiDisplayText(input); got != want {
		t.Fatalf("asciiDisplayText(%q) = %q, want %q", input, got, want)
	}
}
