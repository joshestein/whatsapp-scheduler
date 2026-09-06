package message

import (
	"testing"
	"time"
)

func TestSendAtZoned(t *testing.T) {
	// 13:00 UTC. In Johannesburg (UTC+2, no DST) this is 15:00.
	instant := time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC)

	m := Message{SendAt: instant, SendZone: "Africa/Johannesburg"}
	if got := m.SendAtZoned().Format("15:04"); got != "15:00" {
		t.Errorf("zoned time = %q, want 15:00", got)
	}
}

func TestZoneFallbackWhenEmptyOrUnknown(t *testing.T) {
	instant := time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC)

	for _, zone := range []string{"", "Not/AZone"} {
		m := Message{SendAt: instant, SendZone: zone}
		// Falls back to the machine's local zone: same instant as UTC.
		if !m.SendAtZoned().Equal(instant) {
			t.Errorf("zone %q: instant changed under fallback", zone)
		}
	}
}
