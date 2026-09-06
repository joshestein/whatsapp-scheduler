#!/bin/sh
# WhatsApp Scheduler installer. Downloads the release binary for this machine,
# verifies its checksum, installs a user service that starts it at login
# (launchd on macOS, a systemd user unit on Linux), and starts it now.
#
#   curl -fsSL https://raw.githubusercontent.com/joshestein/whatsapp-scheduler/main/install.sh | sh
#
# Options:
#   WAS_VERSION=v0.1.0 sh install.sh   install this release instead of the latest
#   WAS_PREFIX=~/.local sh install.sh  install the binary under $WAS_PREFIX/bin
#   sh install.sh --local ./binary     install a binary you built (what `make install` runs)
#   sh install.sh --uninstall          stop and remove the service and the binary; data is kept
#
# Run it again to upgrade.

set -eu

REPO="joshestein/whatsapp-scheduler"
BIN="whatsapp-scheduler"
LABEL="com.joshestein.whatsapp-scheduler"
BIN_PATH="${WAS_PREFIX:-$HOME/.local}/bin/$BIN"
URL="http://127.0.0.1:20648"

main() {
  case "$(uname -s)" in
    Darwin)
      OS=darwin
      SERVICE="$HOME/Library/LaunchAgents/$LABEL.plist"
      LOG="$HOME/Library/Logs/$BIN.log"
      DATA="$HOME/Library/Application Support/$BIN" ;;
    Linux)
      OS=linux
      SERVICE="$HOME/.config/systemd/user/$BIN.service"
      LOG=""
      DATA="${XDG_CONFIG_HOME:-$HOME/.config}/$BIN" ;;
    *) die "unsupported OS: $(uname -s) (macOS and Linux only)" ;;
  esac

  case "${1:-}" in
    --uninstall) uninstall; return ;;
    --local) [ -f "${2:-}" ] || die "--local needs a path to a binary"; src="$2" ;;
    "") download ;;
    *) die "unknown option: $1" ;;
  esac

  mkdir -p "$(dirname "$BIN_PATH")" "$(dirname "$SERVICE")"
  install -m 0755 "$src" "$BIN_PATH"
  say "Installed binary to $BIN_PATH"
  "install_$OS"
  say "Service installed: $SERVICE"
  say ""
  say "Done. Open $URL to pair your phone (scan the QR once)."
}

# download fetches the release binary, verifies it, and sets src to its path.
download() {
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "unsupported CPU: $(uname -m)" ;;
  esac
  asset="$BIN-$OS-$arch"
  # latest/download redirects to the newest release with no GitHub API call, so no rate limit.
  base="https://github.com/$REPO/releases/latest/download"
  [ -z "${WAS_VERSION:-}" ] || base="https://github.com/$REPO/releases/download/$WAS_VERSION"

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  say "Downloading $base/$asset"
  curl -fSL --progress-bar "$base/$asset" -o "$tmp/$asset" || die "download failed"
  curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || die "checksum download failed"

  want=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
  got=$( (sha256sum "$tmp/$asset" 2>/dev/null || shasum -a 256 "$tmp/$asset") | awk '{ print $1 }')
  [ -n "$want" ] && [ "$got" = "$want" ] || die "checksum mismatch for $asset: want '$want', got '$got'"
  src="$tmp/$asset"
}

# Log: $LOG. Env defaults are the intended values; see README for overrides.
install_darwin() {
  mkdir -p "$(dirname "$LOG")"
  cat > "$SERVICE" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$LABEL</string>
    <key>ProgramArguments</key>
    <array>
        <string>$BIN_PATH</string>
    </array>
    <!-- Start at login, restart whenever the process exits, at most every 10s. -->
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ThrottleInterval</key>
    <integer>10</integer>
    <key>StandardOutPath</key>
    <string>$LOG</string>
    <key>StandardErrorPath</key>
    <string>$LOG</string>
</dict>
</plist>
PLIST
  # bootout fails when the agent is not loaded yet; normal on first install.
  launchctl bootout "gui/$(id -u)" "$SERVICE" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$SERVICE"
}

# Log: journalctl --user -u whatsapp-scheduler -f. Override env with `systemctl --user edit`.
install_linux() {
  cat > "$SERVICE" <<UNIT
[Unit]
Description=WhatsApp scheduler
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=$BIN_PATH
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
UNIT
  systemctl --user daemon-reload
  systemctl --user enable "$BIN.service"
  systemctl --user restart "$BIN.service"
  if [ "$(loginctl show-user --property=Linger --value 2>/dev/null)" != "yes" ]; then
    say "Tip: a user service stops at logout. For a headless machine, run: loginctl enable-linger"
  fi
}

# Removes the service and the binary. Data (database and WhatsApp session) is
# kept, so a reinstall does not need a new QR scan.
uninstall() {
  if [ "$OS" = darwin ]; then
    launchctl bootout "gui/$(id -u)" "$SERVICE" 2>/dev/null || true
  else
    systemctl --user disable --now "$BIN.service" 2>/dev/null || true
  fi
  rm -f "$SERVICE" "$BIN_PATH"
  [ "$OS" = linux ] && systemctl --user daemon-reload
  say "Removed the service and $BIN_PATH. Data was kept. To remove it too:"
  say "  rm -rf \"$DATA\"${LOG:+ \"$LOG\"}"
  say "Then on your phone: WhatsApp > Settings > Linked devices > unlink this device."
}

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

main "$@"
