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
- macOS (launchd) or Linux with systemd, for the install below

## Run it from source

```
make run
```

Open <http://127.0.0.1:20648>. On first run the page shows a QR code: on your
phone, WhatsApp > Settings > Linked devices > Link a device, and scan it.

Data (database and WhatsApp session) lives in the platform config directory,
`~/Library/Application Support/whatsapp-scheduler` on macOS and
`~/.config/whatsapp-scheduler` on Linux. `make run` and the installed agent
below share it, so you pair once.

## Install (runs at login)

Installs the binary to `~/.local/bin` and a user service that starts it at
login and restarts it if it exits. No sudo. `make` picks the service manager
from `uname`: launchd on macOS, a systemd user unit on Linux.

```
make install
```

Then open <http://127.0.0.1:20648>. If you already paired via `make run`, it is
already connected.

| | macOS | Linux |
|---|---|---|
| Binary | `~/.local/bin/whatsapp-scheduler` | same |
| Service | `~/Library/LaunchAgents/com.joshestein.whatsapp-scheduler.plist` | `~/.config/systemd/user/whatsapp-scheduler.service` |
| Data | `~/Library/Application Support/whatsapp-scheduler/` | `~/.config/whatsapp-scheduler/` |
| Log | `~/Library/Logs/whatsapp-scheduler.log` | `journalctl --user -u whatsapp-scheduler` |

On Linux a user service stops at logout. For a headless box, turn on
lingering once so it keeps running:

```
loginctl enable-linger
```

Day to day:

```
make restart    # rebuild, reinstall the binary, restart the service
make uninstall  # stop the service, remove it and the binary; data is kept
tail -f ~/Library/Logs/whatsapp-scheduler.log         # macOS
journalctl --user -u whatsapp-scheduler -f            # Linux
```

Moving the data directory to another machine: stop the service first, then copy
the database with SQLite's own backup, never with `cp`. The file is in WAL
mode and a plain copy of a recently used database is corrupt.

```
sqlite3 "$HOME/Library/Application Support/whatsapp-scheduler/scheduler.db" ".backup /path/to/scheduler.db"
```

While the service is running, `make run` fails with "address already in use".
That is deliberate: two processes must not share the database and session.
Use `make restart` to test changes, or `make uninstall` first.

## Configuration

By environment variable. The defaults are what the installed service uses.
On Linux, override with a drop-in: `systemctl --user edit whatsapp-scheduler`
and add `Environment=LISTEN_ADDR=...` under `[Service]`.

| Variable | Default | Purpose |
|---|---|---|
| `DATA_DIR` | platform config dir, see above | Where the database and session live |
| `LISTEN_ADDR` | `127.0.0.1:20648` | HTTP listen address |
| `TICK` | `60s` | Scheduler poll interval |
| `GRACE_WINDOW` | `30m` | How late a send may run before it counts as missed |

## Build

```
make build              # local binary in the repo root
make build-linux-arm64  # cross-compile for an arm64 box; run `make install` there
make test
```
