# MingSui 明隧

MingSui 是一个 Go 编写的桌面端与命令行网络代理产品雏形。当前仓库先落地核心链路：本地 SOCKS5 客户端连接远端 relay，relay 负责拨出目标 TCP 连接。

> 当前版本是产品 MVP 的基础骨架，不是已经可公开运营的成熟代理服务。

## 主要组件

- `mingsui`: 客户端 CLI，启动本地 SOCKS5 代理。
- `mingsui-relay`: 远端 relay 服务，负责鉴权和转发。
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
```

启动 relay：

```bash
./bin/mingsui-relay config init -path ./relay.json -token dev-secret -allow-private
./bin/mingsui-relay serve -config ./relay.json
```

启动客户端：

```bash
./bin/mingsui config init -path ./client.json -relay 127.0.0.1:9443 -token dev-secret
./bin/mingsui run -config ./client.json
```

测试 SOCKS5：

```bash
curl --socks5-hostname 127.0.0.1:18080 https://example.com
```

如果本机安装了 `make`，也可以直接运行：

```bash
make build
make test
```

## 配置

示例配置在 `configs/`：

- `configs/client.example.json`
- `configs/relay.example.json`

默认配置路径：

```bash
mingsui config path
mingsui-relay config path
```

## 安全边界

第一版 relay 使用共享 token 鉴权。生产环境至少需要：

- 为 relay 启用 TLS。
- 使用高熵 token，并定期轮换。
- 将 relay 放在受控服务器上，不要使用默认 token。
- 保持 `allow_private_networks=false`，避免 relay 被用来访问内网地址。
- 增加用户体系、设备授权、限速、审计和滥用检测。

## 后续路线

1. 稳定核心代理链路：SOCKS5、HTTP CONNECT、TLS relay、连接状态。
2. 增加配置订阅、节点选择、自动重连和健康检查。
3. 用 Wails 构建桌面端，复用 Go core。
4. 增加账号、授权、计费、流量统计和服务端控制台。
5. 做跨平台打包和自动更新。
