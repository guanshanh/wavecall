#!/usr/bin/env bash
# 打包当前系统的 Wavecall 桌面安装包（Tauri）。
# Windows 产出 nsis/msi，macOS 产出 dmg，Linux 产出 deb 等，都在
# client/src-tauri/target/release/bundle/。
# 服务端托管的 Web 静态文件仍由 deploy 里的 npm run build 生成。
#
# 用法：
#   cp client/.env.example client/.env   # 填写 VITE_DISPATCH_URL
#   bash scripts/build-desktop.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

CLIENT_ENV="${ROOT}/client/.env"
if [[ -f "$CLIENT_ENV" ]]; then
  # shellcheck disable=SC1091
  set -a; source "$CLIENT_ENV"; set +a
fi
: "${VITE_DISPATCH_URL:?请在 client/.env 中设置 VITE_DISPATCH_URL（调度器对外 HTTP 基址），构建前必填}"

echo "==> 打包桌面端 (VITE_DISPATCH_URL=${VITE_DISPATCH_URL})"
(cd "${ROOT}/client" && npm run tauri build)

echo "==> 安装包目录: ${ROOT}/client/src-tauri/target/release/bundle"
