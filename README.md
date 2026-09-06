# WhatsApp Scheduler

A self-hosted tool to schedule WhatsApp messages from your own personal number.

It logs in to WhatsApp Web with your account (QR scan, once) and persists the
session. You pick a contact, write a message and set a send time.

This is a personal, single-user tool. It uses an unofficial WhatsApp client
library, not the official Business API. WhatsApp's terms forbid this and the
number can be banned. Use at your own risk. See `docs/adr/`.

## Stack

- Go, one static binary
- [whatsmeow](https://github.com/tulir/whatsmeow) for the WhatsApp connection
- SQLite, one file for app data and the WhatsApp session
- `net/http` + `html/template` + htmx, no JS build step

## Requirements

- Go 1.26 or later (`go.mod` sets the version; `GOTOOLCHAIN=auto` fetches it if
  your local Go is older)
- `make`
- macOS for the launchd install below. Linux runs the binary fine; a systemd
  unit is planned under `deploy/`.

## Run it from source

```
make run
```

Serves on `127.0.0.1:20648` by default. On first run, scan the printed QR
code with WhatsApp on your phone.

Config is by environment variable:

| Variable | Default | Purpose |
|---|---|---|
| `DATA_DIR` | `./data` | Where the database file lives |
| `LISTEN_ADDR` | `127.0.0.1:20648` | HTTP listen address |
| `TICK` | `60s` | Scheduler poll interval |
| `GRACE_WINDOW` | `30m` | How late a send may run before it counts as missed |

## Build

```
make build              # local binary
make build-linux-arm64  # for the home box (systemd), see deploy/
```
