.PHONY: build run test lint fmt clean

build:
	go build -o alt-codex ./cmd/alt-codex

run:
	go run ./cmd/alt-codex

test:
	go test ./... -race -cover

lint:
	golangci-lint run

fmt:
	gofmt -w .

clean:
	rm -f alt-codex
