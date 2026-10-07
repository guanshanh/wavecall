# Wavecall 部署指南

> TODO: 详细部署步骤将在 MVP 完成后补充。

## 快速概览

### 服务端

```bash
# 构建
cd server
go build -o wavecall-server ./cmd/server

# 运行
./wavecall-server -port 8080 -stun stun:stun.l.google.com:19302
```

### Docker 部署

```bash
docker compose up -d
```

### coturn (TURN 服务器)

```bash
# docker-compose.yml 中已包含 coturn 配置
# 需要修改 TURN 密码和域名
```

### 客户端

- Web 版：部署到 Cloudflare Pages
- 桌面版：`cd client && npm run tauri build`

## 端口

| 服务 | 端口 | 协议 |
|------|------|------|
| Wavecall Server | 8080 | TCP (HTTP/WS) |
| coturn TURN | 3478 | TCP/UDP |
| coturn TURNS | 5349 | TCP (TLS) |
| WebRTC 媒体 | 49152-65535 | UDP |

## 防火墙规则

需要开放：
- TCP 8080 (WebSocket 信令)
- TCP/UDP 3478 (TURN)
- UDP 49152-65535 (WebRTC 媒体)
