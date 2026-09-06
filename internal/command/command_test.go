package command

import "testing"

func TestVerb(t *testing.T) {
	cases := []struct {
		name, in, verb, rest string
	}{
		{"alone", "/schedule\nTo: John\nAt: 15:00\nhi", "schedule", "To: John\nAt: 15:00\nhi"},
		{"inline", "/schedule To: John\nAt: 15:00\nhi", "schedule", "To: John\nAt: 15:00\nhi"},
		{"uppercase verb", "/LIST", "list", ""},
		{"crlf", "/schedule\r\nTo: John\r\nhi", "schedule", "To: John\nhi"},
		{"outer whitespace", "  /schedule\nhi  ", "schedule", "hi"},
		{"no slash", "schedule To: John", "", ""},
		{"note", "buy milk", "", ""},
		{"empty", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			verb, rest := Verb(c.in)
			if verb != c.verb || rest != c.rest {
				t.Errorf("Verb(%q) = %q, %q; want %q, %q", c.in, verb, rest, c.verb, c.rest)
			}
		})
	}
}

func TestRejectIsReplyText(t *testing.T) {
	if got := (Reject{"body empty"}).Error(); got != "✗ body empty" {
		t.Errorf("got %q", got)
	}
}
