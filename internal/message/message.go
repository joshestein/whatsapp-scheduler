package message

import "time"

type State string

const (
	Pending State = "pending"
	Sending State = "sending"
	Sent    State = "sent"
	Missed  State = "missed"
	Failed  State = "failed"
)

type Message struct {
	ID             int64
	RecipientJID   string
	RecipientName  string
	Body           string
	SendAt         time.Time
	State          State
	Error          string
	SentAt         *time.Time
	AcknowledgedAt *time.Time
	CreatedAt      time.Time
}

func (m Message) IsDue(now time.Time) bool {
	return !now.Before(m.SendAt)
}

func (m Message) PastGrace(now time.Time, grace time.Duration) bool {
	return now.After(m.SendAt.Add(grace))
}

// NeedsAck reports whether the UI should show a badge for this message.
func (m Message) NeedsAck() bool {
	return (m.State == Missed || m.State == Failed) && m.AcknowledgedAt == nil
}
