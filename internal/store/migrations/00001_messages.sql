-- +goose Up
CREATE TABLE messages (
    id              INTEGER PRIMARY KEY,
    recipient_jid   TEXT    NOT NULL,
    recipient_name  TEXT    NOT NULL,
    body            TEXT    NOT NULL,
    send_at         INTEGER NOT NULL,
    state           TEXT    NOT NULL
                    CHECK (state IN ('pending', 'sending', 'sent', 'missed', 'failed')),
    error           TEXT,
    sent_at         INTEGER,
    acknowledged_at INTEGER,
    created_at      INTEGER NOT NULL
);

CREATE INDEX messages_state_send_at ON messages (state, send_at);

-- +goose Down
DROP TABLE messages;
