package command

import (
	"regexp"
	"strings"
)

type Schedule struct {
	To, At, Body string
}

var label = regexp.MustCompile(`^(?i)(to|at):\s*(.*)$`)

func ParseSchedule(rest string) (Schedule, error) {
	lines := strings.Split(rest, "\n")
	fields := map[string]string{}
	i := 0
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		m := label.FindStringSubmatch(line)
		if m == nil {
			break
		}

		key := strings.ToLower(m[1])
		if _, dup := fields[key]; dup {
			return Schedule{}, Reject{title(key) + ": given twice"}
		}

		fields[key] = strings.TrimSpace(m[2])
	}

	for _, key := range []string{"to", "at"} {
		if fields[key] == "" {
			return Schedule{}, Reject{title(key) + ": not found"}
		}
	}

	body := strings.TrimSpace(strings.Join(lines[i:], "\n"))
	if body == "" {
		return Schedule{}, Reject{"body empty"}
	}

	return Schedule{To: fields["to"], At: fields["at"], Body: body}, nil
}

func title(key string) string {
	return strings.ToUpper(key[:1]) + key[1:]
}
