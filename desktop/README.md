# 桌面端

当前仓库已经提供两类桌面相关入口：

- `desktop/mingsui-desktop`: Wails 原生桌面客户端，是当前产品体验的主入口。
- `cmd/mingsui-desktop`: 兼容调试入口，会启动本机 HTTP 服务，并尝试用浏览器应用窗口承载界面。

Wails 桌面端复用 `internal/desktop.App`，可以启动/停止客户端、检测 relay、管理 profile，并导入或同步节点订阅。

`mingsui-desktop` 默认读取 `mingsui config path` 指向的同一个客户端配置文件，因此 CLI 和桌面端会天然共享节点、订阅、本地监听和认证设置。

当前产品主方向是 Wails 原生窗口。Chrome/Chromium/Edge 应用窗口只保留给 `cmd/mingsui-desktop` 兼容调试，不应作为桌面端成品体验。

第一阶段桌面端功能：

- 启动/停止本地代理。
- 编辑 relay 地址、token 和 relay profile。
- 导入明隧 JSON 节点订阅，并同步已保存订阅。
- 选择、检测、测速选优和删除机场节点。
- 编辑本地代理认证用户名和密码。
- 调用 `CheckRelayStatus` 验证 relay 地址和 token，并展示 relay 运行指标。
- 显示连接状态、当前本地监听地址和日志。
- 后续接入账号、流量统计和自动更新。

当前仓库先实现 CLI、relay 和核心代理逻辑。桌面端接入时应避免重新实现网络层，只调用 `internal/client` 暴露的服务接口。

启动 Wails 原生桌面客户端：

```bash
make wails-dev
```

构建 Wails 桌面端：

```bash
make wails-desktop
```

启动兼容调试入口：

```bash
go build -o bin/mingsui-desktop ./cmd/mingsui-desktop
./bin/mingsui-desktop -config ./client.json
```

兼容调试入口默认会自动打开应用窗口；关闭窗口后，本机服务也会退出。重复启动时会复用已经运行的本机服务。脚本或测试环境可以加 `-open=false` 只启动本机服务；开发调试时可以加 `-web` 用默认浏览器打开调试界面。

Linux 桌面发布包使用 Debian 包：

```bash
sh scripts/build-deb.sh
sudo apt install ./dist/mingsui-desktop_0.0.0-dev_amd64.deb
```

当前根目录 Windows 发布包会包含兼容入口 `mingsui-desktop.exe`，命令行运行也使用同一套客户端配置。Wails 原生 Windows 桌面端由 `desktop/mingsui-desktop` 的 `wails build` 生成，后续需要收敛到同一条发布流水线。

CLI 诊断命令支持 `-json`。桌面端如果需要展示安装前诊断、端口占用、relay 健康状态或 TLS 证书状态，可以直接复用 JSON 报告结构。

配置查看命令支持默认脱敏输出。桌面端导出排障信息时，应复用脱敏后的配置结构，避免泄露 token 和本地代理密码。

后续 Wails 后端建议用 `client.Controller`：

- `Start(ctx)`：启动本地 SOCKS5/HTTP 代理。
- `Stop(ctx)`：停止本地代理。
- `Status()`：读取运行状态、本地监听地址、relay 地址和最近一次错误。

如果要直接做 Wails 绑定，建议优先绑定 `internal/desktop.App`：

- `ConfigPath()`：返回当前配置文件路径。
- `Config()`：读取当前客户端配置。
- `SaveConfig(cfg)`：保存配置，并刷新运行控制器。
- `UpsertRelayProfile(profile, replace)`：新增或更新 relay profile。
- `SelectRelayProfile(name)`：选择默认 relay profile。
- `RenameRelayProfile(oldName, newName)`：重命名 relay profile。
- `RemoveRelayProfile(name)`：删除 relay profile。
- `ImportRelayProfiles(data, replace, selectName)`：从明隧 JSON 订阅内容导入 profile。
- `UpsertRelaySubscription(sub, replace)`：保存 relay 订阅。
- `RemoveRelaySubscription(name)`：删除 relay 订阅。
- `SyncRelaySubscription(ctx, name, replace, selectName)`：同步 relay 订阅并导入 profile。
- `Start(ctx)` / `Stop(ctx)`：控制本地代理。
- `Status()`：返回桌面 UI 可展示的运行状态。
- `CheckRelay(ctx)`：执行 relay 连接测试。
- `CheckRelayStatus(ctx)`：执行 relay 连接测试，并返回 relay 活跃连接、累计连接和流量指标。
- `CheckRelayProfile(ctx, name)`：测试指定 relay profile。
- `CheckRelayProfileStatus(ctx, name)`：测试指定 relay profile，并返回 relay 指标。

`Status()` 会包含运行指标：

- `active_connections`：当前活跃转发连接数。
- `total_connections`：启动后累计转发连接数。
- `upload_bytes`：客户端发往 relay 的累计字节数。
- `download_bytes`：relay 返回客户端的累计字节数。
