#!/bin/sh
set -eu

MIHOMO_VERSION=${MIHOMO_VERSION:-v1.19.25}
MIHOMO_ASSETS_DIR=${MIHOMO_ASSETS_DIR:-packaging/mihomo}
MIHOMO_PLATFORMS=${MIHOMO_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}
BASE_URL=${BASE_URL:-https://github.com/MetaCubeX/mihomo/releases/download}

command -v curl >/dev/null 2>&1 || {
	echo "curl is required to fetch Mihomo assets" >&2
	exit 1
}

mkdir -p "$MIHOMO_ASSETS_DIR"

for platform in $MIHOMO_PLATFORMS; do
	os=${platform%/*}
	arch=${platform#*/}
	case "$os/$arch" in
		linux/amd64 | linux/arm64 | darwin/amd64 | darwin/arm64 | windows/amd64 | windows/arm64)
			;;
		*)
			echo "unsupported Mihomo platform: $platform" >&2
			exit 1
			;;
	esac

	out="$MIHOMO_ASSETS_DIR/mihomo-$os-$arch"
	case "$os" in
		windows)
			asset="mihomo-$os-$arch-$MIHOMO_VERSION.zip"
			out="$out.exe"
			tmp="$MIHOMO_ASSETS_DIR/$asset"
			echo "fetching $asset"
			curl -fL "$BASE_URL/$MIHOMO_VERSION/$asset" -o "$tmp"
			python3 - "$tmp" "$out" <<'PY'
import sys
import zipfile

archive, output = sys.argv[1:]
with zipfile.ZipFile(archive) as zf:
    names = [name for name in zf.namelist() if name.lower().endswith(".exe")]
    if not names:
        raise SystemExit("zip archive does not contain an exe")
    with zf.open(names[0]) as src, open(output, "wb") as dst:
        dst.write(src.read())
PY
			rm -f "$tmp"
			;;
		*)
			asset="mihomo-$os-$arch-$MIHOMO_VERSION.gz"
			echo "fetching $asset"
			curl -fL "$BASE_URL/$MIHOMO_VERSION/$asset" | gzip -dc >"$out"
			chmod 0755 "$out"
			;;
	esac
done

echo "Mihomo assets written to $MIHOMO_ASSETS_DIR"
