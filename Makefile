build:
	go build -buildvcs=false -ldflags "-X main.Version=$$(git describe --tags --always 2>/dev/null || echo dev)" -o bin/vismo ./cmd/vismo

test:
	go test ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/vismo

.PHONY: build test lint run
