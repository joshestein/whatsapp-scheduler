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
PREFIX="${WAS_PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
BIN_PATH="$BIN_DIR/$BIN"
URL="http://127.0.0.1:20648"

main() {
  os=$(uname -s)
  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *) die "unsupported OS: $os (macOS and Linux only)" ;;
  esac

  case "${1:-}" in
    --uninstall) uninstall; return ;;
    --local) [ -f "${2:-}" ] || die "--local needs a path to a binary"; src="$2" ;;
    "") download ;;
    *) die "unknown option: $1" ;;
  esac

  mkdir -p "$BIN_DIR"
  install -m 0755 "$src" "$BIN_PATH"
  say "Installed binary to $BIN_PATH"

  if [ "$os" = "darwin" ]; then install_launchd; else install_systemd; fi

  case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *) say "Note: $BIN_DIR is not on your PATH (only matters if you run the binary by hand)." ;;
  esac
  say ""
  say "Done. Open $URL to pair your phone (scan the QR once)."
}

# download fetches the release binary and its checksum into a temp dir and sets src to the binary's path.
download() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) die "unsupported CPU: $arch" ;;
  esac
  asset="$BIN-$os-$arch"
  if [ -n "${WAS_VERSION:-}" ]; then
    base="https://github.com/$REPO/releases/download/$WAS_VERSION"
  else
    base="https://github.com/$REPO/releases/latest/download"
  fi

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  say "Downloading $base/$asset"
  curl -fSL --progress-bar "$base/$asset" -o "$tmp/$asset" || die "download failed"
  curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || die "checksum download failed"

  want=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
  [ -n "$want" ] || die "no checksum for $asset in checksums.txt"
  got=$(sha256 "$tmp/$asset")
  [ "$got" = "$want" ] || die "checksum mismatch for $asset: want $want, got $got"
  src="$tmp/$asset"
}

# macOS. Log: ~/Library/Logs/whatsapp-scheduler.log
# No EnvironmentVariables: the app's defaults are the intended values.
# DATA_DIR = ~/Library/Application Support/whatsapp-scheduler, LISTEN_ADDR = 127.0.0.1:20648
install_launchd() {
  plist="$HOME/Library/LaunchAgents/$LABEL.plist"
  log="$HOME/Library/Logs/$BIN.log"
  mkdir -p "$(dirname "$plist")" "$(dirname "$log")"
  cat > "$plist" <<PLIST
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
    <string>$log</string>
    <key>StandardErrorPath</key>
    <string>$log</string>
</dict>
</plist>
PLIST
  # bootout fails when the agent is not loaded yet; normal on first install.
  launchctl bootout "gui/$(id -u)" "$plist" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist"
  say "Service installed: $plist"
}

# Linux. Log: journalctl --user -u whatsapp-scheduler -f
# No Environment= lines: the app's defaults are the intended values.
# DATA_DIR = ~/.config/whatsapp-scheduler, LISTEN_ADDR = 127.0.0.1:20648
# Override with `systemctl --user edit whatsapp-scheduler`, for example
# Environment=LISTEN_ADDR=100.x.y.z:20648 on a Tailscale box.
install_systemd() {
  unit="$HOME/.config/systemd/user/$BIN.service"
  mkdir -p "$(dirname "$unit")"
  cat > "$unit" <<UNIT
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
  say "Service installed: $unit"
  # A user service stops at logout unless lingering is on.
  if [ "$(loginctl show-user --property=Linger --value 2>/dev/null)" != "yes" ]; then
    say "Tip: for a headless machine, run: loginctl enable-linger"
  fi
}

# uninstall removes the service and the binary. The data directory (database
# and WhatsApp session) is kept, so a reinstall does not need a new QR scan.
uninstall() {
  if [ "$os" = "darwin" ]; then
    plist="$HOME/Library/LaunchAgents/$LABEL.plist"
    launchctl bootout "gui/$(id -u)" "$plist" 2>/dev/null || true
    rm -f "$plist"
    data="$HOME/Library/Application Support/$BIN"
    log="$HOME/Library/Logs/$BIN.log"
  else
    systemctl --user disable --now "$BIN.service" 2>/dev/null || true
    rm -f "$HOME/.config/systemd/user/$BIN.service"
    systemctl --user daemon-reload
    data="${XDG_CONFIG_HOME:-$HOME/.config}/$BIN"
    log=""
  fi
  rm -f "$BIN_PATH"
  say "Removed the service and $BIN_PATH."
  say ""
  say "Your data (scheduled messages and the WhatsApp session) was kept. To remove it too:"
  say "  rm -rf \"$data\"${log:+ \"$log\"}"
  say "Then on your phone: WhatsApp > Settings > Linked devices > unlink this device."
}

# sha256 prints the hex digest of a file. macOS has shasum, Linux has sha256sum.
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{ print $1 }'
  else die "need sha256sum or shasum to verify the download"
  fi
}

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

main "$@"
