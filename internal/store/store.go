package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/message"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}

	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.UpContext(ctx, s.db, "migrations")
}

const columns = `id, recipient_jid, recipient_name, body, send_at, state, error, sent_at, acknowledged_at, created_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanMessage(r scanner) (message.Message, error) {
	var (
		m               message.Message
		sendAt, created int64
		errStr          sql.NullString
		sentAt, ackAt   sql.NullInt64
	)
	err := r.Scan(&m.ID, &m.RecipientJID, &m.RecipientName, &m.Body, &sendAt, &m.State, &errStr, &sentAt, &ackAt, &created)
	if err != nil {
		return m, err
	}

	m.SendAt = time.Unix(sendAt, 0).UTC()
	m.CreatedAt = time.Unix(created, 0).UTC()
	m.Error = errStr.String
	if sentAt.Valid {
		t := time.Unix(sentAt.Int64, 0).UTC()
		m.SentAt = &t
	}
	if ackAt.Valid {
		t := time.Unix(ackAt.Int64, 0).UTC()
		m.AcknowledgedAt = &t
	}
	return m, nil
}

func (s *Store) Create(ctx context.Context, m message.Message, now time.Time) (message.Message, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO messages (recipient_jid, recipient_name, body, send_at, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING `+columns,
		m.RecipientJID, m.RecipientName, m.Body, m.SendAt.Unix(), message.Pending, now.Unix())
	return scanMessage(row)
}

