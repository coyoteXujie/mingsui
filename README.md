# MingSui 明隧

MingSui 是一个面向人和 AI Agent 的代理连接产品。

- 桌面端给人用：导入订阅、选择节点、点击连接。
- CLI 给 AI 和自动化用：导入订阅、连接、输出状态、给子命令注入代理环境。
- CLI 和桌面端共用同一套客户端配置。

当前优先支持 Mihomo 作为通用代理内核。机场订阅导入后，明隧会生成 Mihomo 配置并拉起内核；自建 relay profile 仍使用明隧自己的 relay 链路。

机场节点当前可直接导出并连接 `ss`、`vmess`、`trojan`、`vless`、`hysteria2`。订阅里其他协议会保存在配置中，但不会被写进 Mihomo 运行配置；可以通过 `mingsui config proxy list` 查看哪些节点当前可连接。

## 主要组件

- `mingsui`: CLI 客户端，给 AI Agent、脚本和开发者使用。
- `mingsui-desktop`: 桌面端控制台，给普通用户使用。
- `mingsui-relay`: 可选的自建 relay 服务端。
- `mihomo`: 默认通用代理内核，用来连接机场订阅里的节点。

## 安装

CLI 面向 AI Agent 和自动化环境，正式发布后通过 npm 安装：

```bash
npm install -g mingsui
mingsui version
```

桌面端面向普通用户：

- Linux 使用 `.deb` 安装包。
- Windows 使用带 `mingsui-desktop.exe` 的发布包。

源码开发、本地打包和本地 npm 安装见 [docs/development.md](docs/development.md)。

## 快速开始

导入机场订阅：

```bash
mingsui import -source "https://example.com/api/v1/client/subscribe?token=..."
mingsui status
```

连接：

```bash
mingsui connect
```

`mingsui connect` 会保持前台运行。停止这个进程就会断开连接。

让某个命令走明隧代理：

```bash
mingsui exec -- curl https://example.com
```

或者把代理环境变量写入当前 shell：

```bash
eval "$(mingsui env)"
curl https://example.com
```

`mingsui env` 只影响当前 shell 以及后续子进程，不会反向修改已经运行的 Codex、Claude、浏览器或系统代理。

让浏览器等普通应用走明隧代理：

```bash
mingsui system-proxy enable
```

关闭系统代理：

```bash
mingsui system-proxy disable
```

当前系统代理开关优先支持 Linux/GNOME 桌面环境。

## 桌面端

桌面端和 CLI 使用同一份配置。普通用户的目标流程是：

1. 打开明隧桌面端。
2. 粘贴机场订阅或登录账号。
3. 选择节点。
4. 点击连接。

当前开发版桌面控制台启动方式：

```bash
mingsui-desktop
```

命令会打印一个只监听本机的控制台地址，例如 `http://127.0.0.1:18200`。

## 能力边界

明隧 CLI 默认不修改系统代理，也不开启 TUN/虚拟网卡。需要影响浏览器时，显式执行 `mingsui system-proxy enable`。

- 给 AI CLI、脚本、`curl`、`npm`、`git` 用：推荐 `mingsui exec` 或 `mingsui env`。
- 给浏览器用：当前 Linux/GNOME 可用 `mingsui system-proxy enable`；其他桌面环境先手动设置系统代理，后续补齐。
- 本机默认 SOCKS5 端口：`127.0.0.1:18080`。
- 本机默认 HTTP 代理端口：`127.0.0.1:18081`。

## AI Agent

仓库内置 Codex Skill：

```bash
mkdir -p ~/.codex/skills
cp -R skills/mingsui-cli ~/.codex/skills/
```

新启动的 Codex 会在需要代理联网、检查明隧状态、给命令注入代理环境变量时触发 `mingsui-cli` Skill。

## 自建 relay

如果不用机场订阅，也可以部署明隧 relay。先在同一个终端里生成 token，然后启动 relay：

```bash
TOKEN=$(mingsui-relay token)
mingsui-relay config init -path ./relay.json -token "$TOKEN" -allow-private -max-connections 256
mingsui-relay check -config ./relay.json
mingsui-relay serve -config ./relay.json
```

继续使用同一个 `TOKEN`，初始化并启动客户端：

```bash
mingsui config init -path ./client.json -relay 127.0.0.1:9443 -token "$TOKEN"
mingsui doctor -config ./client.json
mingsui run -config ./client.json
```

测试：

```bash
curl --socks5-hostname 127.0.0.1:18080 https://example.com
curl -x http://127.0.0.1:18081 https://example.com
```

## 配置

CLI 和桌面端默认共用同一个客户端配置路径：

```bash
mingsui config path
mingsui config show -path ./client.json
```

`config show` 默认会隐藏 token 和本地代理密码；只有显式加 `-secrets` 才会输出真实敏感值。

常用命令：

```bash
mingsui import -source <机场订阅地址>
mingsui status
mingsui config proxy list
mingsui config proxy select <节点名称>
mingsui config subscription add airport -url <机场订阅地址>
mingsui config subscription sync airport
mingsui kernel export -output /tmp/mingsui-mihomo.yaml
```

订阅 URL 可能包含访问密钥，不要把完整 URL、导出的 Mihomo 配置或节点链接发到日志和工单里。

自建 relay profile 管理：

```bash
mingsui config profile add tokyo -relay tokyo.example.com:9443 -token "$TOKEN"
mingsui config profile select tokyo
mingsui config profile list
```

诊断命令：

```bash
mingsui doctor -config ./client.json
mingsui-relay check -config ./relay.json
mingsui doctor -json -config ./client.json
mingsui-relay check -json -config ./relay.json
```

`mingsui doctor` 会根据当前选择自动诊断：机场节点模式检查本地监听地址、Mihomo 内核、临时内核配置和 Mihomo 自检；自建 relay 模式检查本地监听地址，并通过协议级 `health` 指令验证 relay 地址和 token。如果 relay 版本支持，还会打印服务端活跃连接、累计连接和上下行字节数。`mingsui-relay check` 会检查 relay 配置、TLS 证书和监听地址是否可用。两个诊断命令都支持 `-json`，方便桌面端、部署脚本或监控系统读取结果。

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

1. 桌面端系统代理/TUN 开关。
2. Mihomo 内核随安装包分发，减少本机依赖。
3. 节点延迟测试、自动选择和失败重连。
4. 账号登录、套餐状态、设备授权和计费。
5. 自动更新、签名发布和崩溃/日志诊断。
