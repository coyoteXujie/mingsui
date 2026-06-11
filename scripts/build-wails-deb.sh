#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-dev}
DEB_VERSION=${DEB_VERSION:-}
GO=${GO:-go}
WAILS=${WAILS:-wails}
DIST_DIR=${DIST_DIR:-dist}
WAILS_DESKTOP_DIR=${WAILS_DESKTOP_DIR:-desktop/mingsui-desktop}
WAILS_GO_ENV=${WAILS_GO_ENV:-GOCACHE=/tmp/mingsui-gocache GOMODCACHE=/tmp/mingsui-gomod}
GO_BUILD_ENV=${GO_BUILD_ENV:-$WAILS_GO_ENV}
WAILS_TAGS=${WAILS_TAGS:-$(pkg-config --exists webkit2gtk-4.1 2>/dev/null && echo webkit2_41 || true)}
MIHOMO_ASSETS_DIR=${MIHOMO_ASSETS_DIR:-packaging/mihomo}
REQUIRE_MIHOMO=${REQUIRE_MIHOMO:-0}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}
DATE=${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
LDFLAGS="-s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=${APP_VERSION} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=${COMMIT} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=${DATE}"

if [ -z "$DEB_VERSION" ]; then
	case "$APP_VERSION" in
		v[0-9]*)
			DEB_VERSION=${APP_VERSION#v}
			;;
		[0-9]*)
			DEB_VERSION=$APP_VERSION
			;;
		*)
			DEB_VERSION=0.0.0-${APP_VERSION}
			;;
	esac
fi

command -v dpkg-deb >/dev/null 2>&1 || {
	echo "dpkg-deb is required to build Linux desktop packages" >&2
	exit 1
}

command -v "$WAILS" >/dev/null 2>&1 || {
	echo "wails is required to build the native desktop package" >&2
	exit 1
}

host_arch=$(uname -m)
case "$host_arch" in
	x86_64 | amd64)
		deb_arch=amd64
		goarch=amd64
		;;
	aarch64 | arm64)
		deb_arch=arm64
		goarch=arm64
		;;
	*)
		echo "unsupported native desktop architecture: $host_arch" >&2
		exit 1
		;;
esac

mkdir -p "$DIST_DIR"

if [ -n "$WAILS_TAGS" ]; then
	(cd "$WAILS_DESKTOP_DIR" && env $WAILS_GO_ENV "$WAILS" build -tags "$WAILS_TAGS" -clean -ldflags "$LDFLAGS")
else
	(cd "$WAILS_DESKTOP_DIR" && env $WAILS_GO_ENV "$WAILS" build -clean -ldflags "$LDFLAGS")
fi

case "$WAILS_TAGS" in
	*webkit2_41*)
		webkit_dep=libwebkit2gtk-4.1-0
		;;
	*)
		webkit_dep=libwebkit2gtk-4.0-37
		;;
esac

wails_bin="$WAILS_DESKTOP_DIR/build/bin/mingsui-desktop"
if [ ! -x "$wails_bin" ]; then
	echo "missing Wails desktop binary: $wails_bin" >&2
	exit 1
fi

pkg=mingsui-desktop_${DEB_VERSION}_${deb_arch}
root=$DIST_DIR/deb/$pkg
rm -rf "$root"
mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/lib/mingsui" "$root/usr/share/applications" "$root/usr/share/doc/mingsui-desktop/configs"

env $GO_BUILD_ENV CGO_ENABLED=0 GOOS=linux GOARCH=$goarch "$GO" build -ldflags "$LDFLAGS" -o "$root/usr/bin/mingsui" ./cmd/mingsui
cp "$wails_bin" "$root/usr/bin/mingsui-desktop"

mihomo_source="$MIHOMO_ASSETS_DIR/mihomo-linux-$goarch"
if [ -f "$mihomo_source" ]; then
	cp "$mihomo_source" "$root/usr/lib/mingsui/mihomo"
elif [ "$REQUIRE_MIHOMO" = "1" ]; then
	echo "missing Mihomo asset: $mihomo_source" >&2
	exit 1
fi

cp packaging/deb/mingsui-desktop.desktop "$root/usr/share/applications/mingsui-desktop.desktop"
cp README.md "$root/usr/share/doc/mingsui-desktop/README.md"
cp configs/client.example.json "$root/usr/share/doc/mingsui-desktop/configs/client.example.json"
find "$root" -type d -exec chmod 0755 {} +
find "$root" -type f -exec chmod 0644 {} +
chmod 0755 "$root/usr/bin/mingsui" "$root/usr/bin/mingsui-desktop"
if [ -f "$root/usr/lib/mingsui/mihomo" ]; then
	chmod 0755 "$root/usr/lib/mingsui/mihomo"
fi

installed_size=$(du -ks "$root/usr" | awk '{print $1}')
cat >"$root/DEBIAN/control" <<EOF
Package: mingsui-desktop
Version: $DEB_VERSION
Section: net
Priority: optional
Architecture: $deb_arch
Maintainer: MingSui <mingsui@example.invalid>
Installed-Size: $installed_size
Depends: libgtk-3-0, $webkit_dep
Description: MingSui native desktop and CLI client
 MingSui desktop provides a Wails native desktop client and CLI client that share the same client configuration.
EOF

dpkg-deb --build --root-owner-group "$root" "$DIST_DIR/$pkg.deb"
