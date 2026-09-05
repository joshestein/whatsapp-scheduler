package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/message"
	"github.com/joshestein/whatsapp-scheduler/internal/store"
)

var now = time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

type fakeSender struct {
	connected bool
	err       error
	sent      []string // JIDs, in order
}

func (f *fakeSender) Connected() bool { return f.connected }
func (f *fakeSender) Send(_ context.Context, jid, _ string) error {
	f.sent = append(f.sent, jid)
	return f.err
}

func setup(t *testing.T, sender *fakeSender) (*Scheduler, *store.Store) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	st, err := store.New(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := New(st, sender, log, time.Minute, 30*time.Minute)
	s.now = func() time.Time { return now }
	return s, st
}

func schedule(t *testing.T, st *store.Store, sendAt time.Time) message.Message {
	t.Helper()
	m, err := st.Create(context.Background(), message.Message{
		RecipientJID: "1@s.whatsapp.net", RecipientName: "Test", Body: "hi", SendAt: sendAt,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func state(t *testing.T, st *store.Store, id int64) message.Message {
	t.Helper()
	m, err := st.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTickSendsDue(t *testing.T) {
	sender := &fakeSender{connected: true}
	s, st := setup(t, sender)
	due := schedule(t, st, now.Add(-time.Minute))
	future := schedule(t, st, now.Add(time.Hour))

	s.Tick(context.Background())

	if len(sender.sent) != 1 {
		t.Fatalf("sent %d, want 1", len(sender.sent))
	}
	if got := state(t, st, due.ID); got.State != message.Sent || got.SentAt == nil {
		t.Fatalf("due = %+v", got)
	}
	if got := state(t, st, future.ID); got.State != message.Pending {
		t.Fatalf("future = %s, want pending", got.State)
	}
}

func TestTickMarksMissedWithoutSending(t *testing.T) {
	sender := &fakeSender{connected: true}
	s, st := setup(t, sender)
	old := schedule(t, st, now.Add(-31*time.Minute))

	s.Tick(context.Background())

	if len(sender.sent) != 0 {
		t.Fatalf("sent %d, want 0", len(sender.sent))
	}
	if got := state(t, st, old.ID); got.State != message.Missed {
		t.Fatalf("old = %s, want missed", got.State)
	}
}

func TestTickSkipsWhenDisconnected(t *testing.T) {
	sender := &fakeSender{connected: false}
	s, st := setup(t, sender)
	due := schedule(t, st, now.Add(-time.Minute))
	old := schedule(t, st, now.Add(-31*time.Minute))

	s.Tick(context.Background())

	if len(sender.sent) != 0 {
		t.Fatalf("sent %d, want 0", len(sender.sent))
	}
	if got := state(t, st, due.ID); got.State != message.Pending {
		t.Fatalf("due = %s, want pending (held for next tick)", got.State)
	}
	if got := state(t, st, old.ID); got.State != message.Missed {
		t.Fatalf("old = %s, want missed even while disconnected", got.State)
	}
}

func TestTickMarksRejectedAsFailed(t *testing.T) {
	sender := &fakeSender{connected: true, err: errors.New("blocked")}
	s, st := setup(t, sender)
	due := schedule(t, st, now.Add(-time.Minute))

	s.Tick(context.Background())
	s.Tick(context.Background()) // must not retry

	if len(sender.sent) != 1 {
		t.Fatalf("sent %d, want exactly 1 (no retry)", len(sender.sent))
	}
	if got := state(t, st, due.ID); got.State != message.Failed || got.Error != "blocked" {
		t.Fatalf("due = %+v", got)
	}
}

func TestRunRecoversInFlightOnStart(t *testing.T) {
	sender := &fakeSender{connected: false}
	s, st := setup(t, sender)
	m := schedule(t, st, now.Add(-time.Minute))
	if _, err := st.BeginSending(context.Background(), now); err != nil { // simulate crash mid-send
		t.Fatal(err)
	}

	// Live context for recovery and one tick; the deadline then ends the wait for tick two.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	s.Run(ctx)

	if got := state(t, st, m.ID); got.State != message.Failed || got.Error != "unknown outcome" {
		t.Fatalf("got %+v", got)
	}
}
