-- +goose Up
ALTER TABLE messages ADD COLUMN send_zone TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE messages DROP COLUMN send_zone;
