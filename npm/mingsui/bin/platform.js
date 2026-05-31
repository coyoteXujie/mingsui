"use strict";

const fs = require("fs");
const path = require("path");

const osMap = {
  linux: "linux",
  darwin: "darwin",
  win32: "windows",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

function nativeTarget() {
  const goos = osMap[process.platform];
  const goarch = archMap[process.arch];
  if (!goos || !goarch) {
    throw new Error(`暂不支持当前平台: ${process.platform}/${process.arch}`);
  }
  return { goos, goarch };
}

function resolveBinary() {
  if (process.env.MINGSUI_BINARY) {
    return process.env.MINGSUI_BINARY;
  }

  const { goos, goarch } = nativeTarget();
  const fileName = goos === "windows" ? "mingsui.exe" : "mingsui";
  return path.join(__dirname, "..", "native", `${goos}-${goarch}`, fileName);
}

function ensureBinary() {
  const binary = resolveBinary();
  if (!fs.existsSync(binary)) {
    throw new Error(`没有找到明隧 CLI 二进制文件: ${binary}`);
  }
  return binary;
}

module.exports = {
  ensureBinary,
  nativeTarget,
  resolveBinary,
};
