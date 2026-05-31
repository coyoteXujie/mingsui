"use strict";

const fs = require("fs");
const path = require("path");
const { ensureBinary, nativeTarget } = require("./platform");

try {
  const binary = ensureBinary();
  const target = nativeTarget();
  if (target.goos !== "windows") {
    fs.chmodSync(binary, 0o755);
    const mihomo = path.join(path.dirname(binary), "mihomo");
    if (fs.existsSync(mihomo)) {
      fs.chmodSync(mihomo, 0o755);
    }
  }
} catch (err) {
  console.error(`明隧 CLI 安装失败: ${err.message}`);
  process.exit(1);
}
