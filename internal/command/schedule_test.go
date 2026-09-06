package command

import (
	"errors"
	"testing"
)

func TestParseSchedule(t *testing.T) {
	cases := []struct {
		name, in string
		want     Schedule
	}{
		{
			"basic",
			"To: John\nAt: 15:00\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"labels swapped",
			"At: 15:00\nTo: John\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"label case",
			"to: John\nAT: 15:00\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"blank line inside header",
			"To: John\n\nAt: 15:00\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"blank line before body",
			"To: John\nAt: 15:00\n\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"multi-line body keeps inner newline",
			"To: John\nAt: 15:00\nline one\n\nline two",
			Schedule{To: "John", At: "15:00", Body: "line one\n\nline two"},
		},
		{
			"body line starting with label is body",
			"To: John\nAt: 15:00\nhello\nTo: be honest",
			Schedule{To: "John", At: "15:00", Body: "hello\nTo: be honest"},
		},
		{
			"no space after colon",
			"To:John\nAt:15:00\nhello",
			Schedule{To: "John", At: "15:00", Body: "hello"},
		},
		{
			"values trimmed",
			"To:   John Smith  \nAt: 2026-09-08 15:00 \n  hello  ",
			Schedule{To: "John Smith", At: "2026-09-08 15:00", Body: "hello"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseSchedule(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseScheduleRejects(t *testing.T) {
	cases := []struct {
		name, in, reply string
	}{
		{"missing to", "At: 15:00\nhello", "✗ To: not found"},
		{"missing at", "To: John\nhello", "✗ At: not found"},
		{"empty to value", "To:\nAt: 15:00\nhello", "✗ To: not found"},
		{"both missing reports to first", "hello", "✗ To: not found"},
		{"empty body", "To: John\nAt: 15:00", "✗ body empty"},
		{"whitespace body", "To: John\nAt: 15:00\n   \n", "✗ body empty"},
		{"repeated to", "To: John\nTo: Jane\nAt: 15:00\nhello", "✗ To: given twice"},
		{"repeated at", "To: John\nAt: 15:00\nAt: 16:00\nhello", "✗ At: given twice"},
		{"empty", "", "✗ To: not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseSchedule(c.in)
			var r Reject
			if !errors.As(err, &r) {
				t.Fatalf("want Reject, got %v", err)
			}
			if r.Error() != c.reply {
				t.Errorf("got %q, want %q", r.Error(), c.reply)
			}
		})
	}
}
