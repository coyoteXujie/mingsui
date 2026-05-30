# MingSui 明隧

MingSui 是一个 Go 编写的桌面端与命令行网络代理产品雏形。当前仓库先落地核心链路：本地 SOCKS5/HTTP 客户端连接远端 relay，relay 负责拨出目标 TCP 连接。

> 当前版本是产品 MVP 的基础骨架，不是已经可公开运营的成熟代理服务。

## 主要组件

- `mingsui`: 客户端 CLI，启动本地 SOCKS5 和 HTTP 代理。
- `mingsui-relay`: 远端 relay 服务，负责鉴权和转发。
- `mingsui-desktop`: 本机桌面控制台，提供启动、停止、检测、profile 和订阅管理界面。
- `internal/client`: 本地 SOCKS5 和 relay 连接逻辑。
- `internal/relay`: relay 服务端逻辑。
- `internal/protocol`: 客户端和 relay 之间的轻量协议。
- `desktop`: 桌面端路线说明，后续用 Wails 复用同一套 Go core。

## 快速开始

构建：

```bash
mkdir -p bin
go build -o bin/mingsui ./cmd/mingsui
go build -o bin/mingsui-relay ./cmd/mingsui-relay
go build -o bin/mingsui-desktop ./cmd/mingsui-desktop
```

先在同一个终端里生成 token，然后启动 relay：

```bash
TOKEN=$(./bin/mingsui-relay token)
./bin/mingsui-relay config init -path ./relay.json -token "$TOKEN" -allow-private -max-connections 256
./bin/mingsui-relay check -config ./relay.json
./bin/mingsui-relay serve -config ./relay.json
```

继续使用同一个 `TOKEN`，初始化并启动客户端：

```bash
./bin/mingsui config init -path ./client.json -relay 127.0.0.1:9443 -token "$TOKEN"
./bin/mingsui doctor -config ./client.json
./bin/mingsui run -config ./client.json
```

测试 SOCKS5：

```bash
curl --socks5-hostname 127.0.0.1:18080 https://example.com
```

测试 HTTP 代理：

```bash
curl -x http://127.0.0.1:18081 https://example.com
```

启动本机桌面控制台：

```bash
./bin/mingsui-desktop -config ./client.json
```

命令会打印一个只监听本机的控制台地址，例如 `http://127.0.0.1:18200`。

如果本地代理需要监听到局域网地址，建议启用本地代理认证：

```bash
./bin/mingsui config init -path ./client.json \
  -relay 127.0.0.1:9443 \
  -token "$TOKEN" \
  -local 0.0.0.0:18080 \
  -http 0.0.0.0:18081 \
  -auth-user local-user \
  -auth-pass local-pass

curl --socks5-hostname local-user:local-pass@127.0.0.1:18080 https://example.com
curl -x http://local-user:local-pass@127.0.0.1:18081 https://example.com
```

如果本机安装了 `make`，也可以直接运行：

```bash
make build
make test
make dist APP_VERSION=v0.1.0
make desktop-deb APP_VERSION=v0.1.0
```

跨平台发布包会生成到 `dist/`，并附带 `SHA256SUMS`。Linux 桌面端会生成 `.deb`，Windows 包内包含 `mingsui-desktop.exe`。详细流程见 [docs/release.md](docs/release.md)。

开发时可以直接跑完整测试：

```bash
go test ./...
```

其中 `internal/e2e` 会启动本机 relay、客户端和目标服务，验证 SOCKS5 与 HTTP 代理完整链路。如果当前环境禁止监听本机端口，这组集成测试会自动跳过。

## 配置

示例配置在 `configs/`：

- `configs/client.example.json`
- `configs/relay.example.json`

默认配置路径：

```bash
mingsui config path
mingsui-relay config path
mingsui config show -path ./client.json
mingsui-relay config show -path ./relay.json
```

`mingsui` CLI 和 `mingsui-desktop` 默认共用同一个客户端配置路径，也就是 `mingsui config path` 输出的位置。只有显式传 `-config` 时才会使用另外的配置文件。

`config show` 默认会隐藏 token 和本地代理密码；只有显式加 `-secrets` 才会输出真实敏感值。

客户端支持多个 relay profile。可以把不同服务器写入同一个客户端配置，然后选择默认 profile，或启动时临时指定：

```bash
mingsui config profile add tokyo -path ./client.json -relay tokyo.example.com:9443 -token "$TOKEN"
mingsui config profile add tls-node -path ./client.json -relay relay.example.com:9443 -token "$TOKEN" -tls -server-name relay.example.com
mingsui config profile check tokyo -path ./client.json
mingsui config profile select tokyo -path ./client.json
mingsui config profile list -path ./client.json
mingsui config profile rename tokyo jp-tokyo -path ./client.json
mingsui config profile remove jp-tokyo -path ./client.json
mingsui run -config ./client.json -profile tokyo
```

也可以从明隧 JSON 节点订阅导入 profile。订阅内容可以是 `{"version":1,"profiles":[...]}`，也可以直接是 profile 数组：

```bash
mingsui config profile import -path ./client.json -source ./nodes.json -force
mingsui config profile import -path ./client.json -source https://example.com/mingsui/nodes.json -force -select tokyo
mingsui config subscription add team -path ./client.json -url https://example.com/mingsui/nodes.json
mingsui config subscription list -path ./client.json
mingsui config subscription sync team -path ./client.json
mingsui config subscription remove team -path ./client.json
mingsui config profile export -path ./client.json -output ./nodes.json -secrets
```

订阅 URL 可能包含访问密钥，因此 `config show` 和 `config subscription list` 默认会隐藏订阅 URL；需要排障时再显式加 `-secrets`。

诊断命令：

```bash
mingsui doctor -config ./client.json
mingsui-relay check -config ./relay.json
mingsui doctor -json -config ./client.json
mingsui-relay check -json -config ./relay.json
```

`mingsui doctor` 会检查本地监听地址是否可用，并通过协议级 `health` 指令验证 relay 地址和 token；如果 relay 版本支持，还会打印服务端活跃连接、累计连接和上下行字节数。`mingsui-relay check` 会检查 relay 配置、TLS 证书和监听地址是否可用。两个诊断命令都支持 `-json`，方便桌面端、部署脚本或监控系统读取结果。

生成 token：

```bash
mingsui token
mingsui-relay token
```

`mingsui-relay config init` 默认也支持自动生成 token：

```bash
mingsui-relay config init -path ./relay.json
```

自动生成后会在终端打印 token，需要把同一个 token 写入客户端配置。

## TLS

relay 可以启用 TLS。开发或自托管测试时，可以先生成自签名证书：

```bash
mingsui-relay cert -host example.com,127.0.0.1 -cert relay.crt -key relay.key
```

relay 配置中启用 TLS：

```json
{
  "tls": {
    "enabled": true,
    "cert_file": "relay.crt",
    "key_file": "relay.key"
  }
}
```

`mingsui-relay check` 会解析证书主机名和有效期；证书未生效、已过期会返回失败，30 天内过期会给出警告。

客户端配置中启用 TLS，并把 `ca_file` 指向同一个证书文件：

```json
{
  "tls": {
    "enabled": true,
    "server_name": "example.com",
    "ca_file": "relay.crt",
    "insecure_skip_verify": false
  }
}
```

生产环境建议使用正式 CA 签发的证书，不要开启 `insecure_skip_verify`。

## 部署 relay

服务器上可以用 systemd 托管 relay。先生成服务文件：

```bash
mingsui-relay systemd \
  -binary /usr/local/bin/mingsui-relay \
  -config /etc/mingsui/relay.json \
  -output mingsui-relay.service
```

典型安装步骤：

```bash
sudo install -m 0755 ./bin/mingsui-relay /usr/local/bin/mingsui-relay
sudo mkdir -p /etc/mingsui /var/lib/mingsui
sudo cp ./relay.json /etc/mingsui/relay.json
sudo useradd --system --home /var/lib/mingsui --shell /usr/sbin/nologin mingsui
sudo chown -R mingsui:mingsui /var/lib/mingsui
sudo cp ./mingsui-relay.service /etc/systemd/system/mingsui-relay.service
sudo systemctl daemon-reload
sudo systemctl enable --now mingsui-relay
sudo systemctl status mingsui-relay
```

如果使用 TLS 证书文件，也要确保 `mingsui` 用户能读取证书和私钥。

## 安全边界

第一版 relay 使用共享 token 鉴权。生产环境至少需要：

- 为 relay 启用 TLS。
- 使用高熵 token，并定期轮换。
- 如果客户端本地代理监听在非 loopback 地址，启用 `local_auth`。
- 根据服务器规格设置 `max_connections`，避免 relay 被耗尽资源。
- 将 relay 放在受控服务器上，不要使用默认 token。
- 保持 `allow_private_networks=false`，避免 relay 被用来访问内网地址。
- 增加用户体系、设备授权、限速、审计和滥用检测。

`mingsui-relay check` 会对默认 token、未启用 TLS、允许访问内网目标和未设置连接上限等情况给出警告。上线前应先处理这些警告。

## 后续路线

1. 稳定核心代理链路：SOCKS5、HTTP/CONNECT、TLS relay、连接状态。
2. 增加配置订阅、节点选择、自动重连和健康检查。
3. 用 Wails 构建桌面端，复用 Go core。
4. 增加账号、授权、计费、流量统计和服务端控制台。
5. 做跨平台打包和自动更新。
