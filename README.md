# 🌊 Wavecall

轻量级实时语音通话工具，专为游戏开黑设计。

## 特性

- 🎮 **为开黑而生** — 低延迟语音，5 人同时通话
- 🪶 **极致轻量** — 桌面客户端 < 10MB，服务端单二进制
- 🌐 **Web + 桌面** — 浏览器打开即用，桌面端支持全局快捷键
- 🔒 **房间密码** — 可选密码保护，防止陌生人闯入
- 🖥️ **自部署** — 3M 带宽轻量云即可运行

## 技术栈

| 组件 | 技术 |
|------|------|
| 服务端 | Go + Pion WebRTC (SFU) |
| 客户端 | Tauri 2.0 + React + TypeScript |
| 信令 | WebSocket |
| 音频 | Opus (WebRTC 默认) |
| 样式 | Tailwind CSS |

## 快速开始

### 前置要求

- Go 1.22+
- Node.js 20+
- Rust 1.75+ (桌面端)

### 启动服务端（SFU + 调度器）

本地进房需要**同时**跑 SFU 与调度器；客户端默认 `VITE_DISPATCH_URL=http://127.0.0.1:18090`，也可在登录页「服务器地址」覆盖。

```bash
# 终端 1：SFU（从共享集群表读 n1 的 public_ip / udp_port / 端口）
cd server
go run ./cmd/server -cluster-config configs/cluster.example.toml -node n1

# 终端 2：调度器（同一份集群表）
cd server
go run ./cmd/dispatch -config configs/cluster.example.toml
```

### 启动客户端（Web 开发模式）

```bash
cd client
cp .env.example .env   # VITE_DISPATCH_URL=http://127.0.0.1:18090
npm install
npm run dev
```

浏览器打开 `http://localhost:1420`。

### 启动客户端（Tauri 桌面模式）

```bash
cd client
cp .env.example .env   # 可选；默认 VITE_DISPATCH_URL=http://127.0.0.1:18090，登录页「服务器地址」可覆盖
npm install
npm run tauri dev
```

### 部署到服务器

参见[部署指南](docs/deployment.md)。

## 项目结构

```
wavecall/
├── server/          # Go 服务端
│   ├── cmd/server/  # SFU + 信令
│   ├── cmd/dispatch/# 调度器
│   ├── configs/     # cluster.example.toml（调度 + SFU 共用）
│   ├── internal/    # signaling / sfu / room / config / dispatch
│   └── pkg/proto/   # 消息结构体
├── client/          # Tauri + React 客户端
│   ├── src-tauri/   # Rust 层 (窗口/托盘/快捷键)
│   └── src/         # React 前端
├── docs/            # 基础概念、协议、架构、部署、扩容
```

## 文档

- [基础概念（SFU / ICE / SDP / 调度）](docs/concepts.md)
- [架构设计](docs/architecture.md)
- [信令协议](docs/protocol.md)
- [部署指南](docs/deployment.md)
- [多节点扩容](docs/scaling.md)

## License

Apache-2.0
