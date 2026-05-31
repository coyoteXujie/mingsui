#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-dev}
DEB_VERSION=${DEB_VERSION:-}
GO=${GO:-go}
DIST_DIR=${DIST_DIR:-dist}
DEB_ARCHS=${DEB_ARCHS:-amd64 arm64}
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

mkdir -p "$DIST_DIR"

for deb_arch in $DEB_ARCHS; do
	case "$deb_arch" in
		amd64)
			goarch=amd64
			;;
		arm64)
			goarch=arm64
			;;
		*)
			echo "unsupported deb architecture: $deb_arch" >&2
			exit 1
			;;
	esac

	pkg=mingsui-desktop_${DEB_VERSION}_${deb_arch}
	root=$DIST_DIR/deb/$pkg
	rm -rf "$root"
	mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/lib/mingsui" "$root/usr/share/applications" "$root/usr/share/doc/mingsui-desktop/configs"

	CGO_ENABLED=0 GOOS=linux GOARCH=$goarch "$GO" build -ldflags "$LDFLAGS" -o "$root/usr/bin/mingsui" ./cmd/mingsui
	CGO_ENABLED=0 GOOS=linux GOARCH=$goarch "$GO" build -ldflags "$LDFLAGS" -o "$root/usr/bin/mingsui-desktop" ./cmd/mingsui-desktop
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
Description: MingSui desktop and CLI client
 MingSui desktop provides a local desktop console and CLI client that share the same client configuration.
EOF

	dpkg-deb --build --root-owner-group "$root" "$DIST_DIR/$pkg.deb"
done
