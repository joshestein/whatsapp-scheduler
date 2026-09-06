BIN    := whatsapp-scheduler
LABEL  := com.joshestein.whatsapp-scheduler
PLIST  := $(HOME)/Library/LaunchAgents/$(LABEL).plist
PREFIX ?= $(HOME)/.local

.PHONY: build run test build-linux-arm64 install restart uninstall

build:
	go build -o $(BIN) ./cmd/$(BIN)

run:
	go run ./cmd/$(BIN)

test:
	go test ./...

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BIN)-linux-arm64 ./cmd/$(BIN)

# macOS launchd user agent (plan.md, phase 6). `make install` once, then
# `make restart` after each rebuild. Log: ~/Library/Logs/whatsapp-scheduler.log
install: build
	install -d "$(PREFIX)/bin" "$(dir $(PLIST))"
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	sed -e 's|__BIN__|$(PREFIX)/bin/$(BIN)|' -e 's|__HOME__|$(HOME)|g' deploy/$(LABEL).plist > "$(PLIST)"
	launchctl bootout gui/$$(id -u) "$(PLIST)" 2>/dev/null || true
	launchctl bootstrap gui/$$(id -u) "$(PLIST)"

restart: build
	install -m 0755 $(BIN) "$(PREFIX)/bin/$(BIN)"
	launchctl kickstart -k gui/$$(id -u)/$(LABEL)

uninstall:
	launchctl bootout gui/$$(id -u) "$(PLIST)" 2>/dev/null || true
	rm -f "$(PLIST)" "$(PREFIX)/bin/$(BIN)"
