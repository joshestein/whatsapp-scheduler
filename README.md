# WhatsApp Scheduler

A self-hosted tool to schedule WhatsApp messages from your own personal number.

It logs in to WhatsApp Web with your account (QR scan, once) and persists the
session. You pick a contact, write a message and set a send time.

This is a personal, single-user tool. It uses an unofficial WhatsApp client
library, not the official Business API. Use at your own risk.

## Stack

- Go, one static binary
- [whatsmeow](https://github.com/tulir/whatsmeow) for the WhatsApp connection
- SQLite, one file for app data and the WhatsApp session
- `net/http` + `html/template` + htmx, no JS build step

## Run it

```
make run
```

Serves on `127.0.0.1:8080` by default. On first run, scan the printed QR
code with WhatsApp on your phone.

Config is by environment variable:

| Variable | Default | Purpose |
|---|---|---|
| `DATA_DIR` | `./data` | Where the database file lives |
| `LISTEN_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `TICK` | `60s` | Scheduler poll interval |
| `GRACE_WINDOW` | `30m` | How late a send may run before it counts as missed |

## Build

```
make build              # local binary
make build-linux-arm64  # for the home box (systemd), see deploy/
```
