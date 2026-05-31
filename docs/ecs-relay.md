# ECS relay 部署指南

这份文档面向一台海外 Linux 云服务器，例如阿里云新加坡 ECS。目标是把服务器变成明隧自建 relay，然后让 CLI 和桌面端共用同一份配置连接它。

## 服务器规格

开发和 MVP 阶段，`2 vCPU / 1 GiB` 可以先用。relay 的主要压力通常来自带宽、连接数和跨境线路质量，不是 CPU。上线前需要观察：

- 公网带宽上限和按量流量费用。
- 同时连接数。
- `journalctl` 中的错误率。
- 用户访问目标站点的延迟。

安全组先只开放：

- `22/tcp`：SSH 管理。
- `9443/tcp`：明隧 relay 默认端口。

生产环境建议绑定域名并启用 TLS；没有 TLS 时 token 会明文传输，只适合内测。

## 安装二进制

从发布包中取出 `mingsui-relay`，上传到服务器后安装：

```bash
sudo install -m 0755 ./mingsui-relay /usr/local/bin/mingsui-relay
mingsui-relay version
```

源码开发阶段也可以本地构建后再上传：

```bash
go build -o ./bin/mingsui-relay ./cmd/mingsui-relay
scp ./bin/mingsui-relay root@<服务器 IP>:/usr/local/bin/mingsui-relay
```

## 初始化配置

创建专用系统用户和目录：

```bash
sudo useradd --system --user-group --home-dir /var/lib/mingsui --shell /usr/sbin/nologin mingsui
sudo mkdir -p /etc/mingsui /var/lib/mingsui
sudo chown -R mingsui:mingsui /var/lib/mingsui
```

生成 relay 配置。命令会自动生成 token，请把终端里打印出来的 token 保存到本地密码管理器或部署记录中：

```bash
sudo mingsui-relay config init \
  -path /etc/mingsui/relay.json \
  -max-connections 128
```

收紧配置文件权限，让 systemd 服务可以读取，但普通用户不能读取：

```bash
sudo chown root:mingsui /etc/mingsui/relay.json
sudo chmod 0640 /etc/mingsui/relay.json
```

启动前先检查配置和监听端口：

```bash
sudo mingsui-relay check -config /etc/mingsui/relay.json
```

如果这里提示未启用 TLS，内测阶段可以先接受；正式对外提供服务前应处理。

## systemd 托管

生成服务文件：

```bash
mingsui-relay systemd \
  -binary /usr/local/bin/mingsui-relay \
  -config /etc/mingsui/relay.json \
  -output /tmp/mingsui-relay.service
```

安装并启动：

```bash
sudo install -m 0644 /tmp/mingsui-relay.service /etc/systemd/system/mingsui-relay.service
sudo systemctl daemon-reload
sudo systemctl enable --now mingsui-relay
sudo systemctl status mingsui-relay
```

查看日志：

```bash
sudo journalctl -u mingsui-relay -f
```

## 客户端接入

在本地电脑上添加 relay profile。`<TOKEN>` 使用服务器初始化时打印的 token：

```bash
mingsui config profile add singapore \
  -relay <服务器 IP>:9443 \
  -token "<TOKEN>" \
  -force
mingsui config profile select singapore
mingsui doctor
```

CLI 给 AI 或脚本使用时，推荐一条命令自带临时连接：

```bash
mingsui exec -connect -- curl https://example.com
```

也可以保持前台连接，再让其他命令走代理：

```bash
mingsui connect
```

另开一个终端：

```bash
mingsui exec -- curl https://example.com
```

桌面端和 CLI 读取同一份客户端配置。CLI 选择 `singapore` 后，桌面端也会看到同一个 relay profile。

## 启用 TLS

生产环境建议使用正式 CA 证书。如果只是自托管测试，可以先用明隧生成自签名证书：

```bash
sudo mingsui-relay cert \
  -host relay.example.com,<服务器 IP> \
  -cert /etc/mingsui/relay.crt \
  -key /etc/mingsui/relay.key
sudo chown root:mingsui /etc/mingsui/relay.crt /etc/mingsui/relay.key
sudo chmod 0640 /etc/mingsui/relay.crt /etc/mingsui/relay.key
```

编辑 `/etc/mingsui/relay.json`：

```json
{
  "tls": {
    "enabled": true,
    "cert_file": "/etc/mingsui/relay.crt",
    "key_file": "/etc/mingsui/relay.key"
  }
}
```

重启并检查：

```bash
sudo systemctl restart mingsui-relay
sudo mingsui-relay check -config /etc/mingsui/relay.json
```

客户端使用正式 CA 证书时，一般不需要 `-ca-file`：

```bash
mingsui config profile add singapore \
  -relay relay.example.com:9443 \
  -token "<TOKEN>" \
  -tls \
  -server-name relay.example.com \
  -force
```

如果使用自签名证书，把 `relay.crt` 放到本地，并添加 `-ca-file`：

```bash
mingsui config profile add singapore \
  -relay relay.example.com:9443 \
  -token "<TOKEN>" \
  -tls \
  -server-name relay.example.com \
  -ca-file ./relay.crt \
  -force
```

## 排障

服务器侧：

```bash
sudo systemctl status mingsui-relay
sudo journalctl -u mingsui-relay -n 100
sudo mingsui-relay check -config /etc/mingsui/relay.json
```

本地侧：

```bash
mingsui status
mingsui doctor
mingsui config profile list
mingsui exec -connect -- curl https://example.com
```

常见问题：

- `9443` 不通：检查云服务器安全组、防火墙和 `systemctl status`。
- `token` 错误：重新查看部署记录，必要时重新生成配置并更新客户端 profile。
- TLS 失败：检查 `server_name` 是否和证书域名一致，自签名证书时确认 `-ca-file` 路径存在。
- 速度慢：优先检查服务器地域、带宽上限、晚高峰丢包和目标站点线路。

