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
	printf '%s\r\n' 'tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future'
	printf '%s\r\n' 'vless://00000000-0000-0000-0000-000000000000@example.com:443?security=tls&sni=www.example.com#vless'
	printf '%s\r\n' 'trojan://secret@example.com:443?sni=www.example.com#trojan'
	printf '%s\r\n' 'hysteria2://pass@example.com:8443?sni=www.example.com#hy2'
	printf '%s\r\n' 'ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo'
	printf '%s\r\n' 'vmess://eyJwcyI6Im9zYWthIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMjMifQ=='
} | base64 >"$sub"

echo "==> 导入机场订阅"
"$bin" import -path "$cfg" -source "$sub" >/dev/null

echo "==> 检查状态"
"$bin" status -config "$cfg" -json >"$WORKDIR/status.json"
grep -q '"selected_type": "proxy"' "$WORKDIR/status.json"
grep -q '"selected_proxy": "vless"' "$WORKDIR/status.json"
grep -q '"proxy_profiles": 6' "$WORKDIR/status.json"
grep -q 'Mihomo' "$WORKDIR/status.json"

echo "==> 管理机场节点"
"$bin" config proxy list -path "$cfg" >"$WORKDIR/proxy-list.txt"
grep -q 'tokyo ss 可连接' "$WORKDIR/proxy-list.txt"
"$bin" config proxy select osaka -path "$cfg" >/dev/null
"$bin" config proxy select tokyo -path "$cfg" >/dev/null

echo "==> 检查代理环境变量"
"$bin" env -config "$cfg" >"$WORKDIR/env.sh"
grep -q "HTTP_PROXY='http://127.0.0.1:18081'" "$WORKDIR/env.sh"
grep -q "ALL_PROXY='socks5h://127.0.0.1:18080'" "$WORKDIR/env.sh"
"$bin" exec -config "$cfg" -- sh -c 'test "$HTTP_PROXY" = "http://127.0.0.1:18081"'

echo "==> 检查系统代理状态命令"
"$bin" system-proxy status >"$WORKDIR/system-proxy.json"
grep -q '"supported"' "$WORKDIR/system-proxy.json"

echo "==> 准备测试 Mihomo 内核"
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

echo "==> 诊断机场节点"
MINGSUI_MIHOMO_PATH="$fake_mihomo" "$bin" doctor -config "$cfg" -skip-local -json >"$WORKDIR/doctor.json"
grep -q '"mode": "proxy"' "$WORKDIR/doctor.json"
grep -q '"name": "mihomo_config_test"' "$WORKDIR/doctor.json"

echo "==> 导出 Mihomo 配置"
"$bin" kernel export -config "$cfg" -output "$kernel_cfg" >/dev/null
grep -q 'socks-port: 18080' "$kernel_cfg"
grep -q 'port: 18081' "$kernel_cfg"
grep -q 'type: "vless"' "$kernel_cfg"
grep -q 'type: "trojan"' "$kernel_cfg"
grep -q 'type: "hysteria2"' "$kernel_cfg"
grep -q 'MATCH,明隧' "$kernel_cfg"

echo "==> 检查 connect 会调用 Mihomo"
MINGSUI_MIHOMO_PATH="$fake_mihomo" "$bin" connect -config "$cfg" >"$WORKDIR/connect.out" 2>&1
grep -q '正在启动 Mihomo 内核' "$WORKDIR/connect.out"

echo "smoke test passed"
