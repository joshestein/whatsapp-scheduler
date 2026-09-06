package command

import (
	"errors"
	"testing"
	"time"
)

var jhb = must(time.LoadLocation("Africa/Johannesburg"))

func must(l *time.Location, err error) *time.Location {
	if err != nil {
		panic(err)
	}
	return l
}

func TestResolveAt(t *testing.T) {
	// Saturday 6 Sep 2026, 14:00 in Johannesburg (UTC+2, no DST).
	anchor := time.Date(2026, 9, 6, 14, 0, 0, 0, jhb)
	at := func(d, h, m int) time.Time {
		return time.Date(2026, 9, d, h, m, 0, 0, jhb)
	}

	cases := []struct {
		name, in string
		want     time.Time
	}{
		{"relative minutes", "+30m", at(6, 14, 30)},
		{"relative mixed", "+1h30m", at(6, 15, 30)},
		{"relative hours", "+26h", at(7, 16, 0)},
		{"full date time", "2026-09-08 15:00", at(8, 15, 0)},
		{"full date time in the past still resolves", "2026-09-01 09:00", at(1, 9, 0)},
		{"bare later today", "15:00", at(6, 15, 0)},
		{"bare earlier is today, caller rejects as past", "13:00", at(6, 13, 0)},
		{"bare equal to anchor is today", "14:00", at(6, 14, 0)},
		{"bare single-digit hour", "9:00", at(6, 9, 0)},
		{"full date single-digit hour", "2026-09-08 9:05", at(8, 9, 5)},
		{"12h bare", "3pm", at(6, 15, 0)},
		{"12h bare with minutes", "3:30pm", at(6, 15, 30)},
		{"12h uppercase", "3PM", at(6, 15, 0)},
		{"12h with space", "3 pm", at(6, 15, 0)},
		{"12h am", "9am", at(6, 9, 0)},
		{"12h noon", "12pm", at(6, 12, 0)},
		{"12h midnight", "12am", at(6, 0, 0)},
		{"12h with date", "2026-09-08 3pm", at(8, 15, 0)},
		{"12h with date and minutes", "2026-09-08 3:15 PM", at(8, 15, 15)},
		{"outer whitespace", "  +2h  ", at(6, 16, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveAt(c.in, anchor, jhb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
			if got.Location() != time.UTC {
				t.Errorf("result not UTC: %v", got.Location())
			}
		})
	}
}

func TestResolveAtUsesLocNotAnchorZone(t *testing.T) {
	// Anchor arrives as UTC (whatsmeow gives UTC timestamps). 12:00Z is 14:00
	// in Johannesburg. Bare 15:00 must mean 15:00 Johannesburg, later today.
	anchor := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := ResolveAt("15:00", anchor, jhb)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 6, 15, 0, 0, 0, jhb)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveAtBareUsesAnchorDateInLoc(t *testing.T) {
	// 23:30Z on the 6th is 01:30 on the 7th in Johannesburg. Bare 09:00 means
	// 09:00 on the 7th there, not the 6th.
	anchor := time.Date(2026, 9, 6, 23, 30, 0, 0, time.UTC)
	got, err := ResolveAt("09:00", anchor, jhb)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 7, 9, 0, 0, 0, jhb)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveAtRejects(t *testing.T) {
	anchor := time.Date(2026, 9, 6, 14, 0, 0, 0, jhb)
	cases := []struct{ name, in string }{
		{"date only", "2026-09-08"},
		{"word", "tomorrow"},
		{"zero relative", "+0m"},
		{"negative relative", "-5m"},
		{"bad relative", "+abc"},
		{"iso T form", "2026-09-08T15:00"},
		{"hour only", "15"},
		{"12h out of range", "13pm"},
		{"24h with meridiem", "15:00pm"},
		{"seconds", "15:00:30"},
		{"empty", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ResolveAt(c.in, anchor, jhb)
			var r Reject
			if !errors.As(err, &r) {
				t.Fatalf("want Reject, got %v", err)
			}
			want := "✗ At: cannot read '" + c.in + "'"
			if r.Error() != want {
				t.Errorf("got %q, want %q", r.Error(), want)
			}
		})
	}
}
