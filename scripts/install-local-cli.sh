#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-v0.1.0}
GO=${GO:-go}
DIST_DIR=${DIST_DIR:-dist}
NPM_CACHE=${NPM_CACHE:-${TMPDIR:-/tmp}/mingsui-npm-cache}
MIHOMO_ASSETS_DIR=${MIHOMO_ASSETS_DIR:-packaging/mihomo}
REQUIRE_MIHOMO=${REQUIRE_MIHOMO:-0}
NPM_PLATFORMS=${NPM_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}

case "$APP_VERSION" in
	v[0-9]*)
		NPM_VERSION=${APP_VERSION#v}
		;;
	[0-9]*)
		NPM_VERSION=$APP_VERSION
		;;
	*)
		NPM_VERSION=0.0.0-$(printf '%s' "$APP_VERSION" | tr -c 'A-Za-z0-9-' '-')
		;;
esac

GO="$GO" APP_VERSION="$APP_VERSION" DIST_DIR="$DIST_DIR" NPM_CACHE="$NPM_CACHE" MIHOMO_ASSETS_DIR="$MIHOMO_ASSETS_DIR" REQUIRE_MIHOMO="$REQUIRE_MIHOMO" NPM_PLATFORMS="$NPM_PLATFORMS" sh scripts/build-npm.sh
npm --cache "$NPM_CACHE" install -g "$DIST_DIR/mingsui-$NPM_VERSION.tgz"
NPM_PREFIX=${NPM_CONFIG_PREFIX:-$(npm prefix -g)}
PATH="$NPM_PREFIX/bin:$PATH"
mingsui version
