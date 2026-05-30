# 发布流程

明隧当前使用 Makefile 生成跨平台命令行发布包。发布前先确认工作区干净，并跑完整测试：

```bash
make test
```

生成发布产物：

```bash
make dist APP_VERSION=v0.1.0
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
- `SHA256SUMS`

每个压缩包包含：

- `mingsui`
- `mingsui-relay`
- `mingsui-desktop`
- `README.md`
- `configs/` 示例配置

校验发布包：

```bash
cd dist
sha256sum -c SHA256SUMS
```

如果本机 `go` 不在 `PATH` 中，可以显式指定：

```bash
make dist GO=/home/jie/env/go/bin/go APP_VERSION=v0.1.0
```
