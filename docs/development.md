# 开发与本地测试

这份文档只面向源码开发者。普通用户不需要运行这里的构建命令。

## 本地构建

```bash
go build -o bin/mingsui ./cmd/mingsui
go build -o bin/mingsui-relay ./cmd/mingsui-relay
go build -o bin/mingsui-desktop ./cmd/mingsui-desktop
```

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

这组测试会构建 CLI、导入一份本地测试订阅、检查共享配置、代理环境变量、Mihomo 配置导出，以及 `mingsui connect` 是否会调用 Mihomo 内核。

## 本地安装 CLI

源码仓库里的 `dist/*.tgz` 是构建产物，不会随 git 保存。要在本机模拟 npm 全局安装，直接运行：

```bash
sh scripts/install-local-cli.sh
```

如果本机 Go 不在 `PATH`：

```bash
GO=/home/jie/env/go/bin/go sh scripts/install-local-cli.sh
```

安装后验证：

```bash
mingsui version
mingsui status
```

## 本地发布包

```bash
APP_VERSION=v0.1.0 sh scripts/build-npm.sh
make dist APP_VERSION=v0.1.0
```

详细发布流程见 [release.md](release.md)。

## 持续集成

GitHub Actions 会在 push 和 pull request 时自动执行：

- `go test ./...`
- `sh scripts/smoke-test.sh`
- npm CLI 本地打包和安装验证
