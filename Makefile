.PHONY: build run lint test clean install

BINARY := dpeek
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/dpeek

run:
	go run ./cmd/dpeek $(ARGS)

lint:
	golangci-lint run ./...

test:
	go test ./... -v -count=1

clean:
	rm -f $(BINARY)
	rm -f coverage.out

install: build
	mv $(BINARY) $(shell go env GOPATH)/bin/
