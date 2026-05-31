APP_VERSION ?= dev
GO ?= go
DIST_DIR ?= dist
DIST_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
NPM_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
NPM_PACKAGE_NAME ?= mingsui
MIHOMO_ASSETS_DIR ?= packaging/mihomo
REQUIRE_MIHOMO ?= 0
DEB_ARCHS ?= amd64 arm64
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=$(APP_VERSION) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=$(COMMIT) -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=$(DATE)

.PHONY: build test smoke dist desktop-deb npm-package checksums clean

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
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o "$$work/mingsui-desktop$$ext" ./cmd/mingsui-desktop; \
		mihomo_asset="$(MIHOMO_ASSETS_DIR)/mihomo-$$os-$$arch$$ext"; \
		if [ -f "$$mihomo_asset" ]; then cp "$$mihomo_asset" "$$work/mihomo$$ext"; chmod 0755 "$$work/mihomo$$ext"; \
		elif [ "$(REQUIRE_MIHOMO)" = "1" ]; then echo "missing Mihomo asset: $$mihomo_asset" >&2; exit 1; fi; \
		cp README.md "$$work/README.md"; \
		cp configs/*.json "$$work/configs/"; \
		if [ "$$os" = "windows" ]; then \
			(cd $(DIST_DIR) && zip -qr "$$name.zip" "$$name"); \
		else \
			(cd $(DIST_DIR) && tar -czf "$$name.tar.gz" "$$name"); \
		fi; \
		rm -rf "$$work"; \
	done
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) DEB_ARCHS="$(DEB_ARCHS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-deb.sh
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) NPM_PACKAGE_NAME="$(NPM_PACKAGE_NAME)" NPM_PLATFORMS="$(NPM_PLATFORMS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-npm.sh
	$(MAKE) checksums

desktop-deb:
	APP_VERSION=$(APP_VERSION) GO=$(GO) DIST_DIR=$(DIST_DIR) DEB_ARCHS="$(DEB_ARCHS)" MIHOMO_ASSETS_DIR="$(MIHOMO_ASSETS_DIR)" REQUIRE_MIHOMO="$(REQUIRE_MIHOMO)" sh scripts/build-deb.sh

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
