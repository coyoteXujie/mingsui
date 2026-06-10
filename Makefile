APP_VERSION ?= dev
GO ?= go
WAILS ?= wails
WAILS_GO_ENV ?= GOCACHE=/tmp/mingsui-gocache GOMODCACHE=/tmp/mingsui-gomod
WAILS_TAGS ?= $(shell pkg-config --exists webkit2gtk-4.1 2>/dev/null && echo webkit2_41)
WAILS_TAGS_FLAG = $(if $(strip $(WAILS_TAGS)),-tags "$(WAILS_TAGS)",)
DIST_DIR ?= dist
DIST_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
NPM_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
NPM_PACKAGE_NAME ?= mingsui
MIHOMO_ASSETS_DIR ?= packaging/mihomo
REQUIRE_MIHOMO ?= 0
DEB_ARCHS ?= amd64 arm64
WAILS_DESKTOP_DIR ?= desktop/mingsui-desktop
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=$(APP_VERSION) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=$(COMMIT) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=$(DATE)

.PHONY: build test smoke dist desktop-deb wails-desktop wails-dev npm-package checksums clean

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mingsui ./cmd/mingsui
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mingsui-relay ./cmd/mingsui-relay
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mingsui-desktop ./cmd/mingsui-desktop

test:
	$(GO) test ./...

smoke:
	GO=$(GO) sh scripts/smoke-test.sh

dist:
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) DIST_PLATFORMS="$(DIST_PLATFORMS)" DEB_ARCHS="$(DEB_ARCHS)" NPM_PACKAGE_NAME="$(NPM_PACKAGE_NAME)" NPM_PLATFORMS="$(NPM_PLATFORMS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-dist.sh

desktop-deb:
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) DEB_ARCHS="$(DEB_ARCHS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-deb.sh

wails-desktop:
	cd $(WAILS_DESKTOP_DIR) && env $(WAILS_GO_ENV) $(WAILS) build $(WAILS_TAGS_FLAG) -clean -ldflags "$(LDFLAGS)"

wails-dev:
	cd $(WAILS_DESKTOP_DIR) && env $(WAILS_GO_ENV) $(WAILS) dev $(WAILS_TAGS_FLAG)

npm-package:
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) NPM_PACKAGE_NAME="$(NPM_PACKAGE_NAME)" NPM_PLATFORMS="$(NPM_PLATFORMS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-npm.sh

checksums:
	@files=$$(find $(DIST_DIR) -maxdepth 1 \( -name '*.tar.gz' -o -name '*.zip' -o -name '*.deb' -o -name '*.tgz' \) -printf '%f\n' | sort); \
	if [ -n "$$files" ]; then \
		(cd $(DIST_DIR) && sha256sum $$files > SHA256SUMS); \
		echo "checksums written to $(DIST_DIR)/SHA256SUMS"; \
	else \
		echo "no release archives found in $(DIST_DIR)" >&2; \
		exit 1; \
	fi

clean:
	rm -rf bin dist
