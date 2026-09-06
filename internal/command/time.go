package command

import (
	"strings"
	"time"
)

// Clock layouts, 24h and 12h. Input is lowercased first, so "pm" matches "PM".
var clocks = []string{"15:04", "3:04pm", "3pm", "3:04 pm", "3 pm"}

// ResolveAt turns an At: value into a UTC instant. anchor is when the user sent
// the Command. loc is the zone for wall-clock forms. The past check is the
// caller's; it needs now, and this function only knows anchor.
func ResolveAt(raw string, anchor time.Time, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	bad := Reject{"At: cannot read '" + raw + "'"}

	if rel, ok := strings.CutPrefix(raw, "+"); ok {
		d, err := time.ParseDuration(rel)
		if err != nil || d <= 0 {
			return time.Time{}, bad
		}
		return anchor.Add(d).UTC(), nil
	}
	in := strings.ToLower(raw)
	for _, c := range clocks {
		if t, err := time.ParseInLocation("2006-01-02 "+c, in, loc); err == nil {
			return t.UTC(), nil
		}
		if clock, err := time.ParseInLocation(c, in, loc); err == nil {
			// That wall-clock today in loc, "today" being the anchor's date
			// there. No rollover: an earlier time is past, and the caller
			// rejects it.
			a := anchor.In(loc)
			t := time.Date(a.Year(), a.Month(), a.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
			return t.UTC(), nil
		}
	}
	return time.Time{}, bad
}
