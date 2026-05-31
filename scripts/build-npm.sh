#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-dev}
NPM_PACKAGE_NAME=${NPM_PACKAGE_NAME:-mingsui}
GO=${GO:-go}
DIST_DIR=${DIST_DIR:-dist}
NPM_STAGE=${NPM_STAGE:-$DIST_DIR/npm/mingsui}
NPM_CACHE=${NPM_CACHE:-${TMPDIR:-/tmp}/mingsui-npm-cache}
MIHOMO_ASSETS_DIR=${MIHOMO_ASSETS_DIR:-packaging/mihomo}
REQUIRE_MIHOMO=${REQUIRE_MIHOMO:-0}
NPM_PLATFORMS=${NPM_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}
DATE=${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
LDFLAGS="-s -w -X github.com/coyoteXujie/mingsui/internal/buildinfo.Version=${APP_VERSION} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Commit=${COMMIT} -X github.com/coyoteXujie/mingsui/internal/buildinfo.Date=${DATE}"

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

command -v node >/dev/null 2>&1 || {
	echo "node is required to build the npm package" >&2
	exit 1
}

command -v npm >/dev/null 2>&1 || {
	echo "npm is required to build the npm package" >&2
	exit 1
}

rm -rf "$NPM_STAGE"
mkdir -p "$NPM_STAGE"
cp -R npm/mingsui/. "$NPM_STAGE/"

node - "$NPM_STAGE/package.json" "$NPM_PACKAGE_NAME" "$NPM_VERSION" <<'NODE'
const fs = require("fs");

const [packagePath, packageName, packageVersion] = process.argv.slice(2);
const pkg = JSON.parse(fs.readFileSync(packagePath, "utf8"));
pkg.name = packageName;
pkg.version = packageVersion;
fs.writeFileSync(packagePath, `${JSON.stringify(pkg, null, 2)}\n`);
NODE

for platform in $NPM_PLATFORMS; do
	os=${platform%/*}
	arch=${platform#*/}
	case "$os" in
		linux | darwin | windows)
			;;
		*)
			echo "unsupported npm OS: $os" >&2
			exit 1
			;;
	esac
	case "$arch" in
		amd64 | arm64)
			;;
		*)
			echo "unsupported npm architecture: $arch" >&2
			exit 1
			;;
	esac

	ext=""
	if [ "$os" = "windows" ]; then
		ext=".exe"
	fi

	target_dir="$NPM_STAGE/native/$os-$arch"
	mkdir -p "$target_dir"
	echo "building npm binary $os/$arch"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch "$GO" build -ldflags "$LDFLAGS" -o "$target_dir/mingsui$ext" ./cmd/mingsui
	mihomo_source="$MIHOMO_ASSETS_DIR/mihomo-$os-$arch$ext"
	if [ -f "$mihomo_source" ]; then
		cp "$mihomo_source" "$target_dir/mihomo$ext"
		chmod 0755 "$target_dir/mihomo$ext"
	elif [ "$REQUIRE_MIHOMO" = "1" ]; then
		echo "missing Mihomo asset: $mihomo_source" >&2
		exit 1
	fi
done

mkdir -p "$NPM_CACHE"
npm --cache "$NPM_CACHE" pack "$NPM_STAGE" --pack-destination "$DIST_DIR"
