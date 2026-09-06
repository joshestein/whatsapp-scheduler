package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/message"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrNotFound = errors.New("message not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	return sql.Open("sqlite", dsn)
}

func New(ctx context.Context, db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
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

func scanMessages(rows *sql.Rows) ([]message.Message, error) {
	defer rows.Close()
	var out []message.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) Create(ctx context.Context, m message.Message, now time.Time) (message.Message, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO messages (recipient_jid, recipient_name, body, send_at, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING `+columns,
		m.RecipientJID, m.RecipientName, m.Body, m.SendAt.Unix(), message.Pending, now.Unix())
	return scanMessage(row)
}

func (s *Store) Get(ctx context.Context, id int64) (message.Message, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM messages WHERE id = ?`, id)
	m, err := scanMessage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) List(ctx context.Context) ([]message.Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+columns+` FROM messages ORDER BY send_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	return scanMessages(rows)
}

func (s *Store) BeginSending(ctx context.Context, now time.Time) ([]message.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		UPDATE messages SET state = ?
		WHERE state = ? AND send_at <= ?
		RETURNING `+columns,
		message.Sending, message.Pending, now.Unix())
	if err != nil {
		return nil, err
	}
	return scanMessages(rows)
}

// MarkMissed moves pending messages past their Grace Window to missed.
func (s *Store) MarkMissed(ctx context.Context, now time.Time, grace time.Duration) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE messages SET state = ?
		WHERE state = ? AND send_at <= ?`,
		message.Missed, message.Pending, now.Add(-grace).Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) MarkSent(ctx context.Context, id int64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET state = ?, sent_at = ? WHERE id = ? AND state = ?`,
		message.Sent, now.Unix(), id, message.Sending)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id int64, reason string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET state = ?, error = ? WHERE id = ? AND state = ?`,
		message.Failed, reason, id, message.Sending)
	return err
}

// RecoverSending runs once at startup. A row still in sending was in flight
// when the process died. Its outcome is unknown, so it becomes failed (ADR-0003).
func (s *Store) RecoverSending(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE messages SET state = ?, error = ? WHERE state = ?`,
		message.Failed, "unknown outcome", message.Sending)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Only pending messages are deleted.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ? AND state = ?`, id, message.Pending)
	return err
}

// Acknowledge dismisses the badge on a missed or failed message.
func (s *Store) Acknowledge(ctx context.Context, id int64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE messages SET acknowledged_at = COALESCE(acknowledged_at, ?)
		WHERE id = ? AND state IN (?, ?)`,
		now.Unix(), id, message.Missed, message.Failed)
	return err
}
