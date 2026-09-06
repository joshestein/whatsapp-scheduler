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
	SendZone       string // IANA name of the zone the message was created in; empty means the machine's local zone
	State          State
	Error          string
	SentAt         *time.Time
	AcknowledgedAt *time.Time
	CreatedAt      time.Time
}

func (m Message) IsDue(now time.Time) bool {
	return !now.Before(m.SendAt)
}

// location is the zone the message was created in, falling back to the
// machine's local zone when the stored name is empty or unknown.
func (m Message) location() *time.Location {
	if m.SendZone != "" {
		if loc, err := time.LoadLocation(m.SendZone); err == nil {
			return loc
		}
	}
	return time.Local
}

// SendAtZoned is the send instant rendered in its creation zone, so the list
// shows the same wall-clock the user entered, whatever zone the server runs in.
func (m Message) SendAtZoned() time.Time {
	return m.SendAt.In(m.location())
}

func (m Message) PastGrace(now time.Time, grace time.Duration) bool {
	return !now.Before(m.SendAt.Add(grace))
}

// NeedsAck reports whether the UI should show a badge for this message.
func (m Message) NeedsAck() bool {
	return (m.State == Missed || m.State == Failed) && m.AcknowledgedAt == nil
}
