APP_VERSION ?= dev
GO ?= go
DIST_DIR ?= dist
DIST_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=$(APP_VERSION) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=$(COMMIT) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=$(DATE)

.PHONY: build test dist checksums clean

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mingsui ./cmd/mingsui
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mingsui-relay ./cmd/mingsui-relay

test:
	$(GO) test ./...

dist:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	@set -e; \
	for platform in $(DIST_PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		name=mingsui-$(APP_VERSION)-$${os}-$${arch}; \
		work=$(DIST_DIR)/$${name}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "building $$name"; \
		mkdir -p "$$work/configs"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o "$$work/mingsui$$ext" ./cmd/mingsui; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o "$$work/mingsui-relay$$ext" ./cmd/mingsui-relay; \
		cp README.md "$$work/README.md"; \
		cp configs/*.json "$$work/configs/"; \
		if [ "$$os" = "windows" ]; then \
			(cd $(DIST_DIR) && zip -qr "$$name.zip" "$$name"); \
		else \
			(cd $(DIST_DIR) && tar -czf "$$name.tar.gz" "$$name"); \
		fi; \
		rm -rf "$$work"; \
	done
	$(MAKE) checksums

checksums:
	@if ls $(DIST_DIR)/*.tar.gz $(DIST_DIR)/*.zip >/dev/null 2>&1; then \
		(cd $(DIST_DIR) && sha256sum *.tar.gz *.zip > SHA256SUMS); \
		echo "checksums written to $(DIST_DIR)/SHA256SUMS"; \
	else \
		echo "no release archives found in $(DIST_DIR)" >&2; \
		exit 1; \
	fi

clean:
	rm -rf bin dist
