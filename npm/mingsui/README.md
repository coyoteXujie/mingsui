# MingSui CLI

这是明隧 CLI 的 npm 分发包，适合 AI Agent、自动化脚本和 Node.js 工具链安装使用。

安装后会提供 `mingsui` 命令。这个包只包装 Go 编译出来的原生命令行程序，不包含桌面端。

```bash
npm install -g mingsui
mingsui version
mingsui status
mingsui exec -connect -- curl https://example.com
```

`mingsui exec -connect` 会临时启动当前选择的连接，只把代理环境注入这个子命令，适合 AI Agent 和自动化脚本。

本地开发或发版前可以从仓库构建 npm tarball：

```bash
make npm-package APP_VERSION=v0.1.0
npm install -g ./dist/mingsui-0.1.0.tgz
```

如果 npm 上的 `mingsui` 名称不可用，发布时可以改成 scoped 包：

```bash
make npm-package APP_VERSION=v0.1.0 NPM_PACKAGE_NAME=@coyote-xujie/mingsui
```
