# 发布流程

明隧可以用 shell 脚本或 Makefile 生成发布包。发布前先确认工作区干净，并跑完整测试：

```bash
make test
```

生成发布产物：

```bash
sh scripts/fetch-mihomo.sh
APP_VERSION=v0.1.0 REQUIRE_MIHOMO=1 sh scripts/build-dist.sh
```

如果本机安装了 `make`，也可以使用等价命令：

```bash
make dist APP_VERSION=v0.1.0 REQUIRE_MIHOMO=1
```

上面的命令生成跨平台 CLI、relay、兼容桌面入口压缩包和 npm 包。Linux 原生桌面端单独生成 `.deb`：

```bash
make desktop-deb APP_VERSION=v0.1.0 GO=/home/jie/env/go/bin/go WAILS=/home/jie/env/gopath/bin/wails
make checksums
```

`make desktop-deb` 打包的是 `desktop/mingsui-desktop` 下的 Wails 原生桌面端，并同时安装 CLI。这个包是面向普通用户的 Linux 桌面安装包，只支持在当前主机架构上构建。旧的 `cmd/mingsui-desktop` 本机服务/浏览器兼容入口只保留给调试和脚本场景，需要时显式运行：

```bash
make compat-desktop-deb APP_VERSION=v0.1.0
```

Wails 原生桌面端也可以只构建二进制，不生成 `.deb`：

```bash
make wails-desktop GO=/home/jie/env/go/bin/go WAILS=/home/jie/env/gopath/bin/wails
```

CLI 也可以单独生成 npm 安装包，方便 AI Agent 或自动化环境用 npm 安装：

```bash
make npm-package APP_VERSION=v0.1.0
npm install -g ./dist/mingsui-0.1.0.tgz
mingsui version
```

如果 npm 上的 `mingsui` 名称不可用，发布时可以指定 scoped 包名：

```bash
make npm-package APP_VERSION=v0.1.0 NPM_PACKAGE_NAME=@coyote-xujie/mingsui
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
- `mingsui-desktop_版本_当前主机架构.deb`
- `mingsui-版本.tgz`
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

`.deb` 中的 CLI 和 GUI 默认使用同一个客户端配置路径，和 `mingsui config path` 一致。Windows zip 中的兼容桌面端产物是 `mingsui-desktop.exe`；Wails 原生 Windows 产物由 `desktop/mingsui-desktop` 的 `wails build` 生成。

`make dist` 默认不再生成旧兼容 `.deb`，避免把本机服务/浏览器壳当成桌面安装包发布。确实需要兼容 `.deb` 时，使用 `make compat-desktop-deb` 或 `BUILD_COMPAT_DEB=1 sh scripts/build-dist.sh` 显式生成。

npm 包只包含 `mingsui` CLI 和 Mihomo 内核，不包含 `mingsui-desktop` 和 `mingsui-relay`。默认会内置 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64` 和 `windows/arm64` 六个平台的 CLI 二进制；安装后会提供全局 `mingsui` 命令。

正式发布包应内置 Mihomo 内核。`scripts/fetch-mihomo.sh` 默认会下载 `v1.19.25` 的 Linux、macOS 和 Windows `amd64/arm64` 内核到 `packaging/mihomo/`；`REQUIRE_MIHOMO=1` 会在缺少内核资产时让打包失败，避免产出不能一键连接的包。

校验发布包：

```bash
cd dist
sha256sum -c SHA256SUMS
```

如果本机 `go` 不在 `PATH` 中，可以显式指定：

```bash
GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0 REQUIRE_MIHOMO=1 sh scripts/build-dist.sh
make dist GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
make desktop-deb GO=/home/jie/env/go/bin/go WAILS=/home/jie/env/gopath/bin/wails APP_VERSION=v0.1.0
make checksums
make compat-desktop-deb GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
make npm-package GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
```
