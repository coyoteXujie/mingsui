# 明隧桌面端

这里是 MingSui 的原生桌面客户端工程，使用 Wails + React。用户看到的桌面窗口来自这个工程，不是普通网页预览。

桌面端和 CLI 共用同一份客户端配置，配置路径与 `mingsui config path` 一致。桌面端负责日常连接、订阅、节点、设置和诊断；CLI 负责终端、脚本和 AI Agent 场景。

## 本地开发

在这个目录启动原生桌面窗口：

```bash
wails dev -tags webkit2_41
```

从仓库根目录也可以运行：

```bash
make wails-dev
```

`wails dev` 会启动 Wails 应用窗口，并为前端提供热更新。浏览器调试地址只用于开发排查，不是产品入口。

只验证前端构建：

```bash
cd frontend
npm install
npm run build
```

## 构建

生成本机平台的桌面端发布产物：

```bash
wails build -tags webkit2_41
```

从仓库根目录也可以运行：

```bash
make wails-desktop
```

Linux 上如果 Wails 报 `gcc`、GTK 或 WebKit 相关错误，需要先安装原生构建依赖。Ubuntu 26.04 等新系统使用 WebKitGTK 4.1：

```bash
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

老系统如果软件源只有 WebKitGTK 4.0，则安装 `libwebkit2gtk-4.0-dev`，并直接运行 `wails dev` / `wails build`，不要加 `webkit2_41` 标签。仓库根目录的 `make wails-dev` 和 `make wails-desktop` 会自动检测 4.1 并加标签。

## 和根目录入口的区别

仓库根目录的 `cmd/mingsui-desktop` 是兼容调试入口：它会启动本机 HTTP 服务，并尝试用 Chrome/Chromium/Edge 的应用窗口模式打开界面。它适合脚本、冒烟测试或 Wails 环境不可用时临时使用。

桌面端成品体验应优先以本目录的 Wails 应用为准。
