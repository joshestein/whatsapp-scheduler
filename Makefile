BIN    := whatsapp-scheduler
LABEL  := com.joshestein.whatsapp-scheduler
PREFIX ?= $(HOME)/.local
OS     := $(shell uname -s)

.PHONY: build run test build-linux-arm64 install restart uninstall

build:
	go build -o $(BIN) ./cmd/$(BIN)

run:
	go run ./cmd/$(BIN)

test:
	go test ./...

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BIN)-linux-arm64 ./cmd/$(BIN)

# `make install` once, then `make restart` after each rebuild. The service
# manager depends on the host OS: launchd on macOS, a systemd user unit
# everywhere else. Both start the binary at login and restart it if it exits.

ifeq ($(OS),Darwin)
# macOS launchd user agent (plan.md, phase 6).
# Log: ~/Library/Logs/whatsapp-scheduler.log
PLIST := $(HOME)/Library/LaunchAgents/$(LABEL).plist

install: build
	install -d "$(PREFIX)/bin" "$(dir $(PLIST))"
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	sed -e 's|__BIN__|$(PREFIX)/bin/$(BIN)|g' -e 's|__HOME__|$(HOME)|g' deploy/$(LABEL).plist > "$(PLIST)"
	launchctl bootout gui/$$(id -u) "$(PLIST)" 2>/dev/null || true
	launchctl bootstrap gui/$$(id -u) "$(PLIST)"

restart: build
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	launchctl kickstart -k gui/$$(id -u)/$(LABEL)

uninstall:
	launchctl bootout gui/$$(id -u) "$(PLIST)" 2>/dev/null || true
	rm -f "$(PLIST)" "$(PREFIX)/bin/$(BIN)"

else
# Linux systemd user unit. Log: journalctl --user -u whatsapp-scheduler -f
# The unit stops at logout unless lingering is on: loginctl enable-linger
UNIT := $(HOME)/.config/systemd/user/$(BIN).service

install: build
	install -d "$(PREFIX)/bin" "$(dir $(UNIT))"
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	sed -e 's|__BIN__|$(PREFIX)/bin/$(BIN)|g' deploy/$(BIN).service > "$(UNIT)"
	systemctl --user daemon-reload
	systemctl --user enable $(BIN).service
	systemctl --user restart $(BIN).service

restart: build
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	systemctl --user restart $(BIN).service

uninstall:
	systemctl --user disable --now $(BIN).service 2>/dev/null || true
	rm -f "$(UNIT)" "$(PREFIX)/bin/$(BIN)"
	systemctl --user daemon-reload
endif
