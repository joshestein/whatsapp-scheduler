package command

import "strings"

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
