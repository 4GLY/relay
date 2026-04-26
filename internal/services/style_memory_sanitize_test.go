package services

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeReviewNotesStripsControlCharsAndCaps(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty stays empty", in: "", want: ""},
		{name: "trims surrounding whitespace", in: "  hello  ", want: "hello"},
		{name: "preserves newline and tab", in: "line1\nline2\tend", want: "line1\nline2\tend"},
		{name: "drops bell and backspace", in: "a\x07b\x08c", want: "abc"},
		{name: "drops vertical tab and form feed", in: "a\x0bb\x0cc", want: "abc"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeReviewNotes(tt.in)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeReviewNotesCapsAt200Runes(t *testing.T) {
	in := strings.Repeat("a", 250)
	got := sanitizeReviewNotes(in)
	if utf8.RuneCountInString(got) != 200 {
		t.Fatalf("expected 200 runes, got %d", utf8.RuneCountInString(got))
	}
}

func TestSanitizeReviewNotesCapsMultibyteAt200Runes(t *testing.T) {
	// Korean characters take 3 bytes each in UTF-8; the cap is rune-based so
	// the byte length will differ.
	in := strings.Repeat("가", 250)
	got := sanitizeReviewNotes(in)
	if utf8.RuneCountInString(got) != 200 {
		t.Fatalf("expected 200 runes, got %d", utf8.RuneCountInString(got))
	}
}
