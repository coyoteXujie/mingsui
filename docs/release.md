# 发布流程

明隧当前使用 Makefile 生成跨平台命令行发布包。发布前先确认工作区干净，并跑完整测试：

```bash
make test
```

生成发布产物：

```bash
make dist APP_VERSION=v0.1.0
```

Linux 桌面端也可以单独生成 `.deb`：

```bash
make desktop-deb APP_VERSION=v0.1.0
```

默认会构建这些平台：

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

产物会放在 `dist/`：

- `mingsui-版本-系统-架构.tar.gz`
- `mingsui-版本-windows-amd64.zip`
- `mingsui-desktop_版本_amd64.deb`
- `mingsui-desktop_版本_arm64.deb`
- `SHA256SUMS`

每个压缩包包含：

- `mingsui`
- `mingsui-relay`
- `mingsui-desktop`
- `README.md`
- `configs/` 示例配置

Linux `.deb` 安装：

- `/usr/bin/mingsui`
- `/usr/bin/mingsui-desktop`
- `/usr/share/applications/mingsui-desktop.desktop`
- `/usr/share/doc/mingsui-desktop/README.md`

`.deb` 中的 CLI 和 GUI 默认使用同一个客户端配置路径，和 `mingsui config path` 一致。Windows zip 中的桌面端产物是 `mingsui-desktop.exe`。

校验发布包：

```bash
cd dist
sha256sum -c SHA256SUMS
```

如果本机 `go` 不在 `PATH` 中，可以显式指定：

```bash
make dist GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
make desktop-deb GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
```
