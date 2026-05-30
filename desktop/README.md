# 桌面端

桌面端建议使用 Wails 构建，原因是它能复用 Go core，并用 Web 技术快速开发跨平台 UI。

第一阶段桌面端功能：

- 启动/停止本地代理。
- 编辑 relay 地址和 token。
- 调用 `CheckRelayStatus` 验证 relay 地址和 token，并展示 relay 运行指标。
- 显示连接状态、当前本地监听地址和日志。
- 后续接入节点列表、账号、流量统计和自动更新。

当前仓库先实现 CLI、relay 和核心代理逻辑。桌面端接入时应避免重新实现网络层，只调用 `internal/client` 暴露的服务接口。

后续 Wails 后端建议用 `client.Controller`：

- `Start(ctx)`：启动本地 SOCKS5/HTTP 代理。
- `Stop(ctx)`：停止本地代理。
- `Status()`：读取运行状态、本地监听地址、relay 地址和最近一次错误。

如果要直接做 Wails 绑定，建议优先绑定 `internal/desktop.App`：

- `ConfigPath()`：返回当前配置文件路径。
- `Config()`：读取当前客户端配置。
- `SaveConfig(cfg)`：保存配置，并刷新运行控制器。
- `Start(ctx)` / `Stop(ctx)`：控制本地代理。
- `Status()`：返回桌面 UI 可展示的运行状态。
- `CheckRelay(ctx)`：执行 relay 连接测试。
- `CheckRelayStatus(ctx)`：执行 relay 连接测试，并返回 relay 活跃连接、累计连接和流量指标。

`Status()` 会包含运行指标：

- `active_connections`：当前活跃转发连接数。
- `total_connections`：启动后累计转发连接数。
- `upload_bytes`：客户端发往 relay 的累计字节数。
- `download_bytes`：relay 返回客户端的累计字节数。
