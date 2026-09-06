BIN := whatsapp-scheduler

.PHONY: build run test build-linux-arm64 install restart uninstall

build:
	go build -o $(BIN) ./cmd/$(BIN)

run:
	go run ./cmd/$(BIN)

test:
	go test ./...

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BIN)-linux-arm64 ./cmd/$(BIN)

# install.sh owns the service setup (launchd on macOS, systemd user unit on
# Linux). `--local` installs the binary built here instead of a release.
# Rerunning it restarts the service, so `make restart` is the same thing.
#   macOS log: ~/Library/Logs/whatsapp-scheduler.log
#   Linux log: journalctl --user -u whatsapp-scheduler -f

install: build
	sh install.sh --local ./$(BIN)

restart: install

# Stops and removes the service and the binary. Data is kept.
uninstall:
	sh install.sh --uninstall
