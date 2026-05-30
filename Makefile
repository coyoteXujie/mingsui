APP_VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=$(APP_VERSION) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=$(COMMIT) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=$(DATE)

.PHONY: build test clean

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/mingsui ./cmd/mingsui
	go build -ldflags "$(LDFLAGS)" -o bin/mingsui-relay ./cmd/mingsui-relay

test:
	go test ./...

clean:
	rm -rf bin dist
