package session

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type State string

const (
	Connected    State = "connected"
	Disconnected State = "disconnected"
	LoggedOut    State = "logged out"
)

type Contact struct {
	JID  string
	Name string
}

type Session struct {
	client    *whatsmeow.Client
	log       *slog.Logger
	cancel    context.CancelFunc // stops run(); set by Start
	loggedOut atomic.Bool        // set by the LoggedOut event; cleared only by restart
	qr        atomic.Value       // string; "" when no code is on offer
	paired    atomic.Bool        // set when the phone scans a QR code in this process
}

func New(ctx context.Context, db *sql.DB, log *slog.Logger) (*Session, error) {
	store.DeviceProps.Os = proto.String("WhatsApp Scheduler")
	waLogger := waLog.Stdout("whatsmeow", "WARN", true)
	container := sqlstore.NewWithDB(db, "sqlite3", waLogger)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("upgrade whatsmeow store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	s := &Session{client: whatsmeow.NewClient(device, waLogger), log: log}
	s.qr.Store("")
	s.client.AddEventHandler(s.handleEvent)
	return s, nil
}

// Start connects in the background. If unpaired, it serves QR codes via QR()
// until the phone scans one.
func (s *Session) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	go s.run(ctx)
}

// Stop ends the connect/pair loop and closes the socket. Credentials stay in the db.
func (s *Session) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.client.Disconnect()
}

func (s *Session) State() State {
	switch {
	case s.loggedOut.Load() || s.client.Store.ID == nil:
		return LoggedOut
	case s.client.IsConnected():
		return Connected
	default:
		return Disconnected
	}
}

func (s *Session) Connected() bool { return s.State() == Connected }

func (s *Session) QR() string { return s.qr.Load().(string) }

// Paired reports whether pairing happened in this process. Pages loaded before
// the scan rendered an empty contact list; the UI uses this to ask for a reload.
func (s *Session) Paired() bool { return s.paired.Load() }

func (s *Session) Send(ctx context.Context, jid, text string) error {
	to, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("bad jid %q: %w", jid, err)
	}
	_, err = s.client.SendMessage(ctx, to, &waE2E.Message{Conversation: proto.String(text)})
	return err
}

// Contacts returns people and joined groups sorted by name, with the user first.
func (s *Session) Contacts(ctx context.Context) []Contact {
	// Read once: whatsmeow sets Store.ID to nil on LoggedOut, from its own goroutine.
	id := s.client.Store.ID
	if id == nil {
		return nil
	}
	all, err := s.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		s.log.Warn("contacts", "err", err)
	}
	out := make([]Contact, 0, len(all))
	for jid, c := range all {
		name := c.FullName
		if name == "" {
			name = c.PushName
		}
		if name != "" {
			out = append(out, Contact{JID: jid.String(), Name: name})
		}
	}
	if s.client.IsConnected() {
		groups, err := s.client.GetJoinedGroups(ctx)
		if err != nil {
			s.log.Warn("joined groups", "err", err)
		}
		for _, g := range groups {
			out = append(out, Contact{JID: g.JID.String(), Name: g.Name + " (group)"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	me := Contact{JID: id.ToNonAD().String(), Name: "Me"}
	return append([]Contact{me}, out...)
}

func (s *Session) run(ctx context.Context) {
	for ctx.Err() == nil {
		if s.client.Store.ID != nil {
			if err := s.client.Connect(); err == nil {
				return // whatsmeow auto-reconnects from here
			} else {
				s.log.Error("connect", "err", err)
				sleep(ctx, 10*time.Second)
			}
			continue
		}
		if s.pair(ctx) {
			return
		}
		sleep(ctx, 2*time.Second)
	}
}

// pair serves one QR channel. Returns true once the phone has scanned.
func (s *Session) pair(ctx context.Context) bool {
	ch, err := s.client.GetQRChannel(ctx)
	if err != nil {
		s.log.Error("qr channel", "err", err)
		return false
	}
	if err := s.client.Connect(); err != nil {
		s.log.Error("connect for pairing", "err", err)
		return false
	}
	defer s.qr.Store("")
	for item := range ch {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			s.qr.Store(item.Code)
		case whatsmeow.QRChannelSuccess.Event:
			s.paired.Store(true)
			return true
		default:
			s.log.Warn("pairing", "event", item.Event, "err", item.Error)
		}
	}
	return false // timed out; caller asks for a new channel
}

func (s *Session) handleEvent(evt any) {
	switch e := evt.(type) {
	case *events.LoggedOut:
		s.log.Error("logged out, restart to pair again", "reason", e.Reason)
		s.loggedOut.Store(true)
	case *events.Disconnected, *events.KeepAliveTimeout, *events.StreamReplaced,
		*events.TemporaryBan, *events.ClientOutdated:
		s.log.Warn("session down", "event", fmt.Sprintf("%T", e))
	case *events.Connected:
		s.log.Info("session connected")
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
