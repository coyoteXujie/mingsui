#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-dev}
GO=${GO:-go}
DIST_DIR=${DIST_DIR:-dist}
DIST_PLATFORMS=${DIST_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64}
NPM_PLATFORMS=${NPM_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}
NPM_PACKAGE_NAME=${NPM_PACKAGE_NAME:-mingsui}
MIHOMO_ASSETS_DIR=${MIHOMO_ASSETS_DIR:-packaging/mihomo}
REQUIRE_MIHOMO=${REQUIRE_MIHOMO:-0}
DEB_ARCHS=${DEB_ARCHS:-amd64 arm64}
BUILD_DEB=${BUILD_DEB:-1}
BUILD_NPM=${BUILD_NPM:-1}
WRITE_CHECKSUMS=${WRITE_CHECKSUMS:-1}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}
DATE=${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
LDFLAGS="-s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=${APP_VERSION} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=${COMMIT} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=${DATE}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for platform in $DIST_PLATFORMS; do
	os=${platform%/*}
	arch=${platform#*/}
	case "$os" in
		linux | darwin | windows)
			;;
		*)
			echo "unsupported OS: $os" >&2
			exit 1
			;;
	esac
	case "$arch" in
		amd64 | arm64)
			;;
		*)
			echo "unsupported architecture: $arch" >&2
			exit 1
			;;
	esac

	name=mingsui-$APP_VERSION-$os-$arch
	work=$DIST_DIR/$name
	ext=""
	if [ "$os" = "windows" ]; then
		ext=".exe"
	fi

	echo "building $name"
	mkdir -p "$work/configs"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch "$GO" build -ldflags "$LDFLAGS" -o "$work/mingsui$ext" ./cmd/mingsui
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch "$GO" build -ldflags "$LDFLAGS" -o "$work/mingsui-relay$ext" ./cmd/mingsui-relay
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch "$GO" build -ldflags "$LDFLAGS" -o "$work/mingsui-desktop$ext" ./cmd/mingsui-desktop

	mihomo_asset="$MIHOMO_ASSETS_DIR/mihomo-$os-$arch$ext"
	if [ -f "$mihomo_asset" ]; then
		cp "$mihomo_asset" "$work/mihomo$ext"
		chmod 0755 "$work/mihomo$ext"
	elif [ "$REQUIRE_MIHOMO" = "1" ]; then
		echo "missing Mihomo asset: $mihomo_asset" >&2
		exit 1
	fi

	cp README.md "$work/README.md"
	cp configs/*.json "$work/configs/"

	if [ "$os" = "windows" ]; then
		command -v zip >/dev/null 2>&1 || {
			echo "zip is required to build Windows archives" >&2
			exit 1
		}
		(cd "$DIST_DIR" && zip -qr "$name.zip" "$name")
	else
		(cd "$DIST_DIR" && tar -czf "$name.tar.gz" "$name")
	fi
	rm -rf "$work"
done

if [ "$BUILD_DEB" = "1" ]; then
	APP_VERSION=$APP_VERSION GO=$GO DIST_DIR=$DIST_DIR DEB_ARCHS="$DEB_ARCHS" MIHOMO_ASSETS_DIR="$MIHOMO_ASSETS_DIR" REQUIRE_MIHOMO=$REQUIRE_MIHOMO sh scripts/build-deb.sh
fi

if [ "$BUILD_NPM" = "1" ]; then
	APP_VERSION=$APP_VERSION GO=$GO DIST_DIR=$DIST_DIR NPM_PACKAGE_NAME="$NPM_PACKAGE_NAME" NPM_PLATFORMS="$NPM_PLATFORMS" MIHOMO_ASSETS_DIR="$MIHOMO_ASSETS_DIR" REQUIRE_MIHOMO=$REQUIRE_MIHOMO sh scripts/build-npm.sh
fi

if [ "$WRITE_CHECKSUMS" = "1" ]; then
	files=$(find "$DIST_DIR" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' -o -name '*.deb' -o -name '*.tgz' \) -exec basename {} \; | sort)
	if [ -z "$files" ]; then
		echo "no release archives found in $DIST_DIR" >&2
		exit 1
	fi
	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$DIST_DIR" && sha256sum $files >SHA256SUMS)
	elif command -v shasum >/dev/null 2>&1; then
		(cd "$DIST_DIR" && shasum -a 256 $files >SHA256SUMS)
	else
		echo "sha256sum or shasum is required to write checksums" >&2
		exit 1
	fi
	echo "checksums written to $DIST_DIR/SHA256SUMS"
fi

