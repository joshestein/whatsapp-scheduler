package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/store"
)

// Sender is the part of session.Session the scheduler needs.
type Sender interface {
	Connected() bool
	Send(ctx context.Context, jid, text string) error
}

type Scheduler struct {
	store  *store.Store
	sender Sender
	log    *slog.Logger
	tick   time.Duration
	grace  time.Duration
	now    func() time.Time
}

func New(st *store.Store, sender Sender, log *slog.Logger, tick, grace time.Duration) *Scheduler {
	return &Scheduler{store: st, sender: sender, log: log, tick: tick, grace: grace, now: time.Now}
}

// Run recovers in-flight rows once, then ticks until ctx is cancelled.
// The next tick is armed only after the current one returns, so ticks never overlap.
func (s *Scheduler) Run(ctx context.Context) {
	if n, err := s.store.RecoverSending(ctx); err != nil {
		s.log.Error("recover sending", "err", err)
	} else if n > 0 {
		s.log.Warn("recovered in-flight messages as failed", "count", n)
	}
	for {
		s.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.tick):
		}
	}
}

// Tick is one pass of the poll loop (CONTEXT.md, Scheduler).
func (s *Scheduler) Tick(ctx context.Context) {
	now := s.now()

	// A miss is a timing fact, recorded whether or not the Session is up.
	if n, err := s.store.MarkMissed(ctx, now, s.grace); err != nil {
		s.log.Error("mark missed", "err", err)
		return
	} else if n > 0 {
		s.log.Warn("messages missed", "count", n)
	}

	if !s.sender.Connected() {
		s.log.Debug("session not connected, skipping tick")
		return
	}

	due, err := s.store.BeginSending(ctx, now)
	if err != nil {
		s.log.Error("claim due", "err", err)
		return
	}

	for _, m := range due {
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := s.sender.Send(sendCtx, m.RecipientJID, m.Body)
		cancel()

		if err != nil {
			s.log.Error("send failed", "id", m.ID, "to", m.RecipientJID, "err", err)
			if err := s.store.MarkFailed(ctx, m.ID, err.Error()); err != nil {
				s.log.Error("mark failed", "id", m.ID, "err", err)
			}
			continue
		}
		s.log.Info("sent", "id", m.ID, "to", m.RecipientJID)
		if err := s.store.MarkSent(ctx, m.ID, s.now()); err != nil {
			s.log.Error("mark sent", "id", m.ID, "err", err)
		}
	}
}
