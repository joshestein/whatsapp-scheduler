# WhatsApp Scheduler

Schedule WhatsApp messages from your own number. Runs on your machine, in the
background. Pair once with a QR scan, then pick a contact, write a message and
set a send time.

Personal, single-user tool. It uses an unofficial WhatsApp client library, not
the official Business API. WhatsApp's terms forbid this and the number can be
banned. Use at your own risk. See `docs/adr/`.

## Install

macOS (Apple Silicon or Intel) or Linux (amd64 or arm64).

```
curl -fsSL https://raw.githubusercontent.com/joshestein/whatsapp-scheduler/main/install.sh | sh
```

Then open <http://127.0.0.1:20648>. The page shows a QR code. On your phone:
WhatsApp > Settings > Linked devices > Link a device, and scan it.

The service starts at login and restarts if it exits. Run the same line again
to upgrade.

```
sh install.sh --uninstall           # remove the service and binary; keeps your data
WAS_VERSION=v0.1.0 sh install.sh    # a specific release
```

The script checks the download against the release's `checksums.txt`. It is
one file; read it before you run it if you like.

## Where things live

|         | macOS                                                            | Linux                                               |
| ------- | ---------------------------------------------------------------- | --------------------------------------------------- |
| Binary  | `~/.local/bin/whatsapp-scheduler`                                | same                                                |
| Service | `~/Library/LaunchAgents/com.joshestein.whatsapp-scheduler.plist` | `~/.config/systemd/user/whatsapp-scheduler.service` |
| Data    | `~/Library/Application Support/whatsapp-scheduler/`              | `~/.config/whatsapp-scheduler/`                     |
| Log     | `~/Library/Logs/whatsapp-scheduler.log`                          | `journalctl --user -u whatsapp-scheduler -f`        |

Data is one SQLite file: your scheduled messages and the WhatsApp session.

On Linux the service stops at logout. On a headless box, run once:

```
loginctl enable-linger
```

To move data to another machine, stop the service and use SQLite's backup,
never `cp`. The database is in WAL mode and a plain copy is corrupt.

```
sqlite3 "$HOME/Library/Application Support/whatsapp-scheduler/scheduler.db" ".backup /path/to/scheduler.db"
```

## Configuration

Environment variables. The defaults are what the installed service uses. On
Linux, override with `systemctl --user edit whatsapp-scheduler` and add
`Environment=LISTEN_ADDR=...` under `[Service]`.

| Variable       | Default           | Purpose                                            |
| -------------- | ----------------- | -------------------------------------------------- |
| `DATA_DIR`     | see table above   | Where the database and session live                |
| `LISTEN_ADDR`  | `127.0.0.1:20648` | HTTP listen address                                |
| `TICK`         | `60s`             | Scheduler poll interval                            |
| `GRACE_WINDOW` | `30m`             | How late a send may run before it counts as missed |

## Development

Go 1.26+ and `make`. One static Go binary:
[whatsmeow](https://github.com/tulir/whatsmeow), SQLite, `net/http` +
`html/template` + htmx. No JS build step.

```
make run                # foreground; same data dir as the service, so you pair once
make test
make build              # binary in the repo root
make install            # build, then `install.sh --local`: same result as the curl install
make uninstall
make build-linux-arm64  # cross-compile for an arm64 box; run `make install` there
```

`make run` fails with "address already in use" while the service is running.
That is deliberate: two processes must not share the session. Use
`make install` to test changes, or `make uninstall` first.
