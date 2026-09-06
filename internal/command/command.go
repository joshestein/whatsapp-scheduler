package command

import "strings"

// Reject is a user-facing failure. Error() is the exact reply text.
type Reject struct{ Msg string }

func (r Reject) Error() string { return "✗ " + r.Msg }

// Verb splits "/schedule To: John\n..." into "schedule" and "To: John\n...".
// Returns "" when text does not start with "/".
func Verb(text string) (verb, rest string) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	first, rest, _ := strings.Cut(text, "\n")
	word, inline, _ := strings.Cut(first, " ")
	verb = strings.ToLower(strings.TrimPrefix(word, "/"))
	if inline = strings.TrimSpace(inline); inline != "" {
		rest = inline + "\n" + rest
	}
	return verb, rest
}
