VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT        := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS       := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

.PHONY: build build-cli build-mcp test cover lint clean install

build: build-cli build-mcp

build-cli:
	go build $(LDFLAGS) -o bin/hppycli ./cmd/hppycli

build-mcp:
	go build $(LDFLAGS) -o bin/hppymcp ./cmd/hppymcp

test:
	go test ./... -v -count=1

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html

install:
	go install $(LDFLAGS) ./cmd/hppycli ./cmd/hppymcp
