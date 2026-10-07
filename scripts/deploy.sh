#!/usr/bin/env bash
# Wavecall 一键部署：交叉编译服务端 + 构建客户端 + scp 上传到云服务器
#
# 用法：
#   cp .env.example .env    # 填写服务器信息
#   bash scripts/deploy.sh
#
# 上传后如需重启服务：
#   ssh -p <端口> <user>@<host> 'systemctl restart wavecall'
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "错误：未找到 .env，请先执行 cp .env.example .env 并填写服务器信息" >&2
  exit 1
fi
# shellcheck disable=SC1091
set -a; source .env; set +a

: "${DEPLOY_HOST:?请在 .env 中设置 DEPLOY_HOST}"
: "${DEPLOY_USER:?请在 .env 中设置 DEPLOY_USER}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/wavecall}"
DEPLOY_GOARCH="${DEPLOY_GOARCH:-amd64}"

SSH_OPTS=(-p "$DEPLOY_PORT" -o ConnectTimeout=10)
SCP_OPTS=(-P "$DEPLOY_PORT" -o ConnectTimeout=10)

echo "==> 目标 ${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_DIR} (ssh 端口 ${DEPLOY_PORT})"

CLUSTER_EXAMPLE="${ROOT}/server/configs/cluster.example.toml"
if [[ ! -f "$CLUSTER_EXAMPLE" ]]; then
  echo "错误：未找到 ${CLUSTER_EXAMPLE}，无法打包集群配置" >&2
  exit 1
fi

CLIENT_ENV="${ROOT}/client/.env"
if [[ -f "$CLIENT_ENV" ]]; then
  # shellcheck disable=SC1091
  set -a; source "$CLIENT_ENV"; set +a
fi
: "${VITE_DISPATCH_URL:?请在 client/.env 中设置 VITE_DISPATCH_URL（调度器对外 HTTP 基址），构建前必填}"

echo "==> [1/5] 交叉编译服务端与调度器 (linux/${DEPLOY_GOARCH})"
(
  cd server
  CGO_ENABLED=0 GOOS=linux GOARCH="$DEPLOY_GOARCH" go build -o wavecall-server ./cmd/server
  CGO_ENABLED=0 GOOS=linux GOARCH="$DEPLOY_GOARCH" go build -o wavecall-dispatch ./cmd/dispatch
)

echo "==> [2/5] 构建客户端 (VITE_DISPATCH_URL=${VITE_DISPATCH_URL})"
(cd client && npm run build)

echo "==> [3/5] 创建远程目录"
ssh "${SSH_OPTS[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "mkdir -p '${DEPLOY_DIR}/web'"

echo "==> [4/5] 上传二进制与静态文件"
scp "${SCP_OPTS[@]}" server/wavecall-server server/wavecall-dispatch "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_DIR}/"
ssh "${SSH_OPTS[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "chmod +x '${DEPLOY_DIR}/wavecall-server' '${DEPLOY_DIR}/wavecall-dispatch'"
scp "${SCP_OPTS[@]}" -r client/dist/. "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_DIR}/web/"

echo "==> [5/5] 上传集群表示例（请复制为 cluster.toml 并按环境修改后，调度与 SFU 共用）"
scp "${SCP_OPTS[@]}" "$CLUSTER_EXAMPLE" "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_DIR}/cluster.example.toml"

echo "==> 完成。示例启动："
echo "    ${DEPLOY_DIR}/wavecall-dispatch -config ${DEPLOY_DIR}/cluster.toml"
echo "    ${DEPLOY_DIR}/wavecall-server -cluster-config ${DEPLOY_DIR}/cluster.toml -node n1 -web ${DEPLOY_DIR}/web"
echo "    ssh -p ${DEPLOY_PORT} ${DEPLOY_USER}@${DEPLOY_HOST} 'systemctl restart wavecall'"
