# 开发与本地测试

这份文档只面向源码开发者。普通用户不需要运行这里的构建命令。

## 本地构建

```bash
go build -o bin/mingsui ./cmd/mingsui
go build -o bin/mingsui-relay ./cmd/mingsui-relay
go build -o bin/mingsui-desktop ./cmd/mingsui-desktop
```

上面的 `cmd/mingsui-desktop` 是兼容调试入口，会启动本机服务并尝试用浏览器应用窗口承载界面。它不是当前桌面端的主开发入口。

## 原生桌面端开发

原生桌面端工程在 `desktop/mingsui-desktop`，使用 Wails + React。运行桌面客户端时应从这个目录启动：

```bash
make wails-dev
```

`wails dev` 会打开独立桌面窗口，并启动前端热更新服务。浏览器里打开 Vite 地址只适合调试 DOM 和样式，不代表用户看到的桌面客户端。

如果本机还没有 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Linux 本机还需要 Wails 对应的系统依赖，包括 C 编译器和 GTK/WebKit 开发包。Ubuntu 26.04 等新系统通常需要：

```bash
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

老系统如果软件源只有 WebKitGTK 4.0，则安装 `libwebkit2gtk-4.0-dev`。仓库 Makefile 会自动检测 `webkit2gtk-4.1` 并给 Wails 加 `webkit2_41` 构建标签；直接运行 Wails CLI 时，4.1 系统需要手动加 `-tags webkit2_41`。

依赖缺失时，先用下面的前端构建命令验证界面代码：

```bash
cd desktop/mingsui-desktop/frontend
npm install
npm run build
```

构建原生桌面端发布产物：

```bash
make wails-desktop
```

如果不想通过 Makefile，也可以在 `desktop/mingsui-desktop` 目录直接运行 `wails dev -tags webkit2_41` 或 `wails build -tags webkit2_41`。如果系统安装的是 `libwebkit2gtk-4.0-dev`，不要加 `webkit2_41` 标签。

根目录兼容入口仍可用于脚本或只验证本机服务：

```bash
go build -o bin/mingsui-desktop ./cmd/mingsui-desktop
./bin/mingsui-desktop -open=false
./bin/mingsui-desktop -web
```

`-web` 明确是浏览器调试模式，不应在演示或验收时当作桌面客户端。

如果本机安装了 `make`：

```bash
make build
make test
```

完整测试：

```bash
go test ./...
```

产品级冒烟测试：

```bash
sh scripts/smoke-test.sh
```

这组测试会构建 CLI、导入一份本地测试订阅、检查共享配置、代理环境变量、relay/机场节点的 `mingsui exec -connect`、Mihomo 配置导出，以及 `mingsui connect` 是否会调用 Mihomo 内核。

## 本地安装 CLI

源码仓库里的 `dist/*.tgz` 是构建产物，不会随 git 保存。要在本机模拟 npm 全局安装，直接运行：

```bash
sh scripts/install-local-cli.sh
```

如果本机 Go 不在 `PATH`：

```bash
GO=/home/jie/env/go/bin/go sh scripts/install-local-cli.sh
```

只想生成 npm tarball 时，先构建再安装：

```bash
APP_VERSION=v0.1.0 GO=/home/jie/env/go/bin/go sh scripts/build-npm.sh
ls dist/mingsui-0.1.0.tgz
npm install -g ./dist/mingsui-0.1.0.tgz
```

安装后验证：

```bash
mingsui version
mingsui status
```

## 本地发布包

下载发布包内置的 Mihomo 内核：

```bash
sh scripts/fetch-mihomo.sh
```

生成发布产物：

```bash
APP_VERSION=v0.1.0 sh scripts/build-npm.sh
APP_VERSION=v0.1.0 sh scripts/build-dist.sh
make dist APP_VERSION=v0.1.0 REQUIRE_MIHOMO=1
```

`scripts/build-dist.sh` 和 `make dist` 等价；没有 `make` 时直接用脚本。

详细发布流程见 [release.md](release.md)。

## 持续集成

GitHub Actions 会在 push 和 pull request 时自动执行：

- `go test ./...`
- `sh scripts/smoke-test.sh`
- npm CLI 本地打包和安装验证
