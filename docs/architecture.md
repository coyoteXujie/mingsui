# 架构说明

MingSui 当前采用一个很小的客户端/relay 架构。

```text
browser/app
   |
   | SOCKS5
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

客户端监听本地 SOCKS5 地址，默认是 `127.0.0.1:18080`。收到 CONNECT 请求后，客户端会连接 relay，发送带 token 的 `ConnectRequest`，relay 接受后开始双向转发字节流。

## Relay 服务端

relay 监听公网或内网地址，默认是 `0.0.0.0:9443`。它负责：

- 校验协议版本。
- 校验共享 token。
- 校验目标地址。
- 拨出 TCP 连接。
- 在客户端连接和目标连接之间复制字节流。

默认情况下，relay 会拒绝本地、私有、链路本地、多播等目标地址，降低被滥用访问内网服务的风险。

## 通信协议

客户端和 relay 之间先交换一个长度前缀 JSON 消息：

```json
{
  "version": 1,
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

## 桌面端

桌面端计划使用 Wails。Wails 的 Go 后端可以直接调用 `internal/client`，前端只负责展示状态、节点选择、日志和设置。
