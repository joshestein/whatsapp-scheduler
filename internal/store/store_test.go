package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/message"
)

var now = time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

func open(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func create(t *testing.T, s *Store, sendAt time.Time) message.Message {
	t.Helper()
	m, err := s.Create(context.Background(), message.Message{
		RecipientJID: "1@s.whatsapp.net", RecipientName: "Test", Body: "hi", SendAt: sendAt,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestBeginSendingBeginsOnce(t *testing.T) {
	s := open(t)
	ctx := context.Background()
	create(t, s, now.Add(-time.Minute))
	create(t, s, now.Add(time.Minute)) // future, must stay pending

	first, err := s.BeginSending(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].State != message.Sending {
		t.Fatalf("first claim = %+v", first)
	}
	second, err := s.BeginSending(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("second claim returned %d rows, want 0", len(second))
	}
}

func TestMarkMissedRespectsGrace(t *testing.T) {
	s := open(t)
	ctx := context.Background()
	old := create(t, s, now.Add(-31*time.Minute))
	fresh := create(t, s, now.Add(-29*time.Minute))

	n, err := s.MarkMissed(ctx, now, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("marked %d, want 1", n)
	}
	if got, _ := s.Get(ctx, old.ID); got.State != message.Missed {
		t.Fatalf("old = %s, want missed", got.State)
	}
	if got, _ := s.Get(ctx, fresh.ID); got.State != message.Pending {
		t.Fatalf("fresh = %s, want pending", got.State)
	}
}

func TestSentAndFailedRequireSending(t *testing.T) {
	s := open(t)
	ctx := context.Background()
	m := create(t, s, now)

	if err := s.MarkSent(ctx, m.ID, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("MarkSent on pending = %v, want ErrNotFound", err)
	}
	if _, err := s.BeginSending(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed(ctx, m.ID, "rejected"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(ctx, m.ID)
	if got.State != message.Failed || got.Error != "rejected" || !got.NeedsAck() {
		t.Fatalf("got %+v", got)
	}
}

func TestRecoverSending(t *testing.T) {
	s := open(t)
	ctx := context.Background()
	m := create(t, s, now)
	if _, err := s.BeginSending(ctx, now); err != nil {
		t.Fatal(err)
	}

	n, err := s.RecoverSending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("recovered %d, want 1", n)
	}
	got, _ := s.Get(ctx, m.ID)
	if got.State != message.Failed || got.Error != "unknown outcome" {
		t.Fatalf("got %+v", got)
	}
}
