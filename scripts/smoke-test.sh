#!/bin/sh
set -eu

GO=${GO:-go}
WORKDIR=${WORKDIR:-${TMPDIR:-/tmp}/mingsui-smoke}

case "$WORKDIR" in
	"" | "/")
		echo "WORKDIR 不安全: $WORKDIR" >&2
		exit 1
		;;
esac

rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
trap 'rm -rf "$WORKDIR"' EXIT INT TERM

bin="$WORKDIR/mingsui"
cfg="$WORKDIR/client.json"
sub="$WORKDIR/airport.txt"
kernel_cfg="$WORKDIR/mihomo.yaml"
fake_mihomo="$WORKDIR/mihomo"

echo "==> 构建 CLI"
"$GO" build -o "$bin" ./cmd/mingsui

echo "==> 准备测试订阅"
{
	printf '%s\r\n' 'ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo'
	printf '%s\r\n' 'vmess://eyJwcyI6Im9zYWthIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMjMifQ=='
} | base64 >"$sub"

echo "==> 导入机场订阅"
"$bin" import -path "$cfg" -source "$sub" >/dev/null

echo "==> 检查状态"
"$bin" status -config "$cfg" -json >"$WORKDIR/status.json"
grep -q '"selected_type": "proxy"' "$WORKDIR/status.json"
grep -q '"proxy_profiles": 2' "$WORKDIR/status.json"
grep -q 'Mihomo' "$WORKDIR/status.json"

echo "==> 检查代理环境变量"
"$bin" env -config "$cfg" >"$WORKDIR/env.sh"
grep -q "HTTP_PROXY='http://127.0.0.1:18081'" "$WORKDIR/env.sh"
grep -q "ALL_PROXY='socks5h://127.0.0.1:18080'" "$WORKDIR/env.sh"
"$bin" exec -config "$cfg" -- sh -c 'test "$HTTP_PROXY" = "http://127.0.0.1:18081"'

echo "==> 导出 Mihomo 配置"
"$bin" kernel export -config "$cfg" -output "$kernel_cfg" >/dev/null
grep -q 'socks-port: 18080' "$kernel_cfg"
grep -q 'port: 18081' "$kernel_cfg"
grep -q 'MATCH,明隧' "$kernel_cfg"

echo "==> 检查 connect 会调用 Mihomo"
cat >"$fake_mihomo" <<'EOF'
#!/bin/sh
set -eu
config=""
while [ "$#" -gt 0 ]; do
	case "$1" in
		-f)
			shift
			config=$1
			;;
	esac
	shift || true
done
test -n "$config"
test -s "$config"
grep -q 'MATCH,明隧' "$config"
exit 0
EOF
chmod +x "$fake_mihomo"
MINGSUI_MIHOMO_PATH="$fake_mihomo" "$bin" connect -config "$cfg" >"$WORKDIR/connect.out" 2>&1
grep -q '正在启动 Mihomo 内核' "$WORKDIR/connect.out"

echo "smoke test passed"
