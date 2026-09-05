BIN := whatsapp-scheduler

.PHONY: build run test build-linux-arm64

build:
	go build -o $(BIN) ./cmd/scheduler

run:
	go run ./cmd/scheduler

test:
	go test ./...

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BIN)-linux-arm64 ./cmd/scheduler
