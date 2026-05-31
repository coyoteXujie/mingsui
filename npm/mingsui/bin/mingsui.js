#!/usr/bin/env node
"use strict";

const { spawnSync } = require("child_process");
const { ensureBinary } = require("./platform");

let binary;
try {
  binary = ensureBinary();
} catch (err) {
  console.error(`明隧 CLI 启动失败: ${err.message}`);
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), {
  stdio: "inherit",
  windowsHide: false,
});

if (result.error) {
  console.error(`明隧 CLI 执行失败: ${result.error.message}`);
  process.exit(1);
}

if (result.signal) {
  process.kill(process.pid, result.signal);
}

process.exit(result.status === null ? 1 : result.status);
