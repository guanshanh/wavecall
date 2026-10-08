# Wavecall 部署指南

MVP 采用**裸二进制部署**：SFU（`cmd/server`）和调度器（`cmd/dispatch`）各一个 Go 二进制，跑在带公网 IP 的云服务器上，不依赖 Docker。单节点时两者可以在同一台机器。

调度 / 信令 / 媒体、ICE 与集群表字段含义见 [基础概念](concepts.md)。

## 构建与上传

### 一键部署脚本（推荐）

```bash
cp .env.example .env     # 填写 DEPLOY_HOST / DEPLOY_USER
bash scripts/deploy.sh
```

脚本会交叉编译服务端、构建客户端，并通过 scp 上传到远程目录
（`/opt/wavecall/wavecall-server` + `/opt/wavecall/web/`）。

### 手动构建上传

```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wavecall-server ./cmd/server
scp wavecall-server user@<server-ip>:/opt/wavecall/

cd ../client
npm run build
scp -r dist/. user@<server-ip>:/opt/wavecall/web/
```

## 运行

推荐与调度器共用同一份集群表（见下文），用 `-cluster-config` + `-node` 启动：

```bash
./wavecall-server -cluster-config /opt/wavecall/cluster.toml -node n1 -web /opt/wavecall/web -users /opt/wavecall/users.toml
```

| flag | 默认值 | 说明 |
|------|--------|------|
| `-cluster-config` | （空） | 共享集群 TOML（与调度器 `-config` 同一文件） |
| `-node` | （空） | 本机在表中的 id（如 `n1`）；与 `-cluster-config` 成对使用 |
| `-port` | 18080 | HTTP/WebSocket 监听端口；未显式指定时从节点 `signaling` 解析 |
| `-public-ip` | （空） | ICE host 候选；未显式指定时用节点 `public_ip` |
| `-udp-port` | 18081 | 媒体 UDPMux 端口；未显式指定且表中 `udp_port`≠0 时用表内值 |
| `-stun` | （空） | 可选 STUN 回退 |
| `-web` | （空） | Web 客户端静态文件目录（SPA 回退） |
| `-users` | `configs/users.toml` | 与调度器同一份用户名单；缺失、空 `secret` 或没有账号时进程退出 |

本地快速调试仍可不用集群表，直接传 `-port` / `-public-ip` / `-udp-port`。

### 托管 Web 客户端（推荐）

客户端进房前会先请求调度器（见下文「调度器」），再在返回的 SFU 节点上建立 WebSocket。
构建前在 `client/.env` 中设置 `VITE_DISPATCH_URL`（调度器对外 HTTP 基址）。

```bash
cd client
cp .env.example .env                  # 填写 VITE_DISPATCH_URL
npm install && npm run build          # 产出 client/dist/
scp -r dist user@<server-ip>:/opt/wavecall/web
```

```bash
./wavecall-server -cluster-config /opt/wavecall/cluster.toml -node n1 -web /opt/wavecall/web -users /opt/wavecall/users.toml
```

> 不设置 `-web` 时服务端仅提供信令（配合 `npm run dev` 本地开发）；客户端仍须配置 `VITE_DISPATCH_URL`。

### systemd 守护

创建运行用户并写入 `/etc/systemd/system/wavecall.service`：

```bash
sudo useradd -r -s /usr/sbin/nologin wavecall
sudo chown wavecall:wavecall /opt/wavecall
```

```ini
[Unit]
Description=Wavecall SFU server
After=network-online.target

[Service]
ExecStart=/opt/wavecall/wavecall-server -cluster-config /opt/wavecall/cluster.toml -node n1 -web /opt/wavecall/web -users /opt/wavecall/users.toml
Restart=always
RestartSec=3
User=wavecall
WorkingDirectory=/opt/wavecall

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now wavecall
journalctl -u wavecall -f
```

## 共享集群表与调度器

调度器与 SFU **读同一份** `cluster.toml`（示例见 `server/configs/cluster.example.toml`）：

- 调度：按 `roomId` 哈希，向客户端返回某节点的 `signaling`（`host:port`）
- SFU：`-node n1` 读取自己的 `public_ip` / `udp_port`，并把 `signaling` 的端口当作监听端口

不做动态注册；加减节点 = 改表 + 滚动重启。见 [多节点扩容](scaling.md)。

### 构建调度器

```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wavecall-dispatch ./cmd/dispatch
scp wavecall-dispatch user@<server-ip>:/opt/wavecall/
```

### 配置与运行

复制示例为 `/opt/wavecall/cluster.toml` 并按环境填写。单节点只保留一条即可。

```toml
[dispatch]
bind = ":18090"

[nodes.n1]
signaling = "<sfu-public-host-or-ip>:18080"
public_ip = "<sfu-public-ip>"
udp_port = 18081
```

```bash
./wavecall-dispatch -config /opt/wavecall/cluster.toml -users /opt/wavecall/users.toml
./wavecall-server -cluster-config /opt/wavecall/cluster.toml -node n1 -web /opt/wavecall/web -users /opt/wavecall/users.toml
```

将 `server/configs/users.example.toml` 复制为服务器上的 `/opt/wavecall/users.toml`，不要把真名单放进仓库。密码以原文写在 TOML 中。修改用户名单后需重启调度器与全部 SFU 节点。

客户端构建时必须设置 `VITE_DISPATCH_URL` 为调度器对外基址（如 `https://dispatch.example.com` 或 `http://<server-ip>:18090`），见 `client/.env.example`。

**调度地址与信令 WebSocket：** 客户端根据 `VITE_DISPATCH_URL` 的 scheme 选择 `wss` 或 `ws` 连接返回的 SFU 节点。若调度对外是 **https**，信令会使用 **wss**；SFU 仅监听明文 `:18080` 的部署请对调度使用 **http://**（本地开发常用 `http://127.0.0.1:18090`），不要用 https 调度地址去连未配 TLS 的 SFU。

| 组件 | 端口 | 协议 |
|------|------|------|
| SFU 信令 / 健康检查 | 18080 | TCP |
| SFU WebRTC 媒体 | 18081 | UDP |
| 调度器 `GET /dispatch`、`POST /login`、`/health` | 18090 | TCP |

## 防火墙 / 安全组

| 端口 | 协议 | 用途 |
|------|------|------|
| 18080 | TCP | SFU：HTTP 健康检查 + WebSocket 信令 |
| 18081 | UDP | SFU：WebRTC 媒体（单端口复用，入站必须放行） |
| 18090 | TCP | 调度器：房间分配 + 健康检查 |

## 验证

```bash
curl http://<server-ip>:18080/health   # SFU，期望 ok
curl http://<server-ip>:18090/health   # 调度器，期望 ok
curl "http://<server-ip>:18090/dispatch?room=test"   # 期望 JSON {"node":"..."}
```

两台不同网络的设备打开客户端加入同一房间，确认能互相对讲。

## NAT / STUN / TURN

- 客户端从 NAT 后主动向服务端公网地址发起连接。
- 单端口复用（`-udp-port`）下所有媒体走同一个 UDP 端口，安全组只需放行一个口。

`internal/config` 已预留 `TURNServer/TURNUser/TURNPass` 字段；若后续真实跨网
测试遇到连接失败需要中继兜底，再启用 coturn 并为 `main.go` 接线 `-turn`
相关 flag。
