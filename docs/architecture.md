# 架构说明

MingSui 当前采用一个很小的客户端/relay 架构。

```text
browser/app
   |
   | SOCKS5 或 HTTP/CONNECT
   v
mingsui client
   |
   | MingSui relay protocol over TCP or TLS
   v
mingsui-relay
   |
   | TCP
   v
target host
```

## 客户端

客户端监听本地 SOCKS5 地址，默认是 `127.0.0.1:18080`。新建配置时还会写入 HTTP 代理地址 `127.0.0.1:18081`。

客户端收到 SOCKS5 CONNECT 或 HTTP CONNECT 请求后，会连接 relay，发送带 token 的 `ConnectRequest`，relay 接受后开始双向转发字节流。普通 HTTP 请求会被转换成 origin-form 后转发给目标服务器。

## Relay 服务端

relay 监听公网或内网地址，默认是 `0.0.0.0:9443`。它负责：

- 校验协议版本。
- 校验共享 token。
- 校验目标地址。
- 按 `max_connections` 限制活跃转发连接数。
- 拨出 TCP 连接。
- 在客户端连接和目标连接之间复制字节流。

默认情况下，relay 会拒绝本地、私有、链路本地、多播等目标地址，降低被滥用访问内网服务的风险。

relay 可以直接监听 TCP，也可以启用 TLS。自签名证书可以用 `mingsui-relay cert` 生成；生产环境建议使用正式 CA 签发的证书。

Linux 服务器上可以用 `mingsui-relay systemd` 生成服务文件，再交给 systemd 托管运行。

## 通信协议

客户端和 relay 之间先交换一个长度前缀 JSON 消息。普通代理连接使用 `connect` 指令：

```json
{
  "version": 1,
  "command": "connect",
  "token": "shared-secret",
  "network": "tcp",
  "address": "example.com:443"
}
```

relay 返回：

```json
{
  "version": 1,
  "ok": true
}
```

返回成功后，连接切换为原始字节流转发。

健康检查使用 `health` 指令：

```json
{
  "version": 1,
  "command": "health",
  "token": "shared-secret"
}
```

relay 对健康检查只校验协议版本和 token，不拨出目标地址。

健康检查成功响应会带上 relay 运行指标：

```json
{
  "version": 1,
  "ok": true,
  "metrics": {
    "active_connections": 0,
    "total_connections": 12,
    "upload_bytes": 4096,
    "download_bytes": 8192
  }
}
```

## 桌面端

桌面端计划使用 Wails。Wails 的 Go 后端可以直接调用 `internal/client`，前端只负责展示状态、节点选择、日志和设置。桌面端的“连接测试”可以复用 `CheckRelayStatus`，避免重复实现网络诊断逻辑。

本地代理认证属于客户端入口安全能力，不参与 relay 协议。启用 `local_auth` 后，SOCKS5 使用 username/password 方法，HTTP 代理使用 `Proxy-Authorization: Basic ...`。
