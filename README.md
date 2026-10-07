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

### 启动服务端

```bash
cd server
go run ./cmd/server
```

服务端默认监听 `:8080`。

### 启动客户端（Web 开发模式）

```bash
cd client
npm install
npm run dev
```

浏览器打开 `http://localhost:1420`。

### 启动客户端（Tauri 桌面模式）

```bash
cd client
npm install
npm run tauri dev
```

### Docker 部署

```bash
docker compose up -d
```

## 项目结构

```
wavecall/
├── server/          # Go 服务端 (SFU + 信令)
│   ├── cmd/server/  # 入口
│   ├── internal/    # signaling / sfu / room / config
│   └── pkg/proto/   # 消息结构体
├── client/          # Tauri + React 客户端
│   ├── src-tauri/   # Rust 层 (窗口/托盘/快捷键)
│   └── src/         # React 前端
├── docs/            # 协议文档 + 架构说明
└── docker-compose.yml
```

## 文档

- [架构设计](docs/architecture.md)
- [信令协议](docs/protocol.md)
- [部署指南](docs/deployment.md)

## License

Apache-2.0
