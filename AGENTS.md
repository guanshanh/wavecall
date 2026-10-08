# Wavecall — 实时语音开黑工具

## 项目简介

Wavecall 是一个轻量级实时语音通话工具，专为游戏开黑场景设计。目标支持 5 人低延迟语音通话，部署在 3M 带宽轻量云服务器上。

## 技术栈

| 层 | 技术 |
|----|------|
| 服务端 | Go + [Pion WebRTC](https://github.com/pion/webrtc) v4 (SFU 架构) |
| 信令 | WebSocket ([gorilla/websocket](https://github.com/gorilla/websocket)) |
| 客户端 | [Tauri 2.0](https://v2.tauri.app/) (Rust 壳) + React 18 + TypeScript |
| 构建 | Vite |
| 状态管理 | Zustand |
| 样式 | Tailwind CSS |
| 音频编码 | Opus (WebRTC 默认) |
| NAT 穿透 | 集群表 `public_ip` + `udp_port` 写入 ICE host 候选；TURN 暂缓，STUN 可选 |

## 项目结构

```
wavecall/
├── server/          # Go 服务端
│   ├── cmd/server/  # SFU + 信令
│   ├── cmd/dispatch/# 独立调度进程
│   ├── configs/     # cluster.example.toml（调度与 SFU 共用）
│   ├── internal/    # signaling / sfu / room / config / dispatch
│   └── pkg/proto/   # 公共消息结构体
├── client/          # Tauri + React 客户端
│   ├── src-tauri/   # Rust 层 (窗口/托盘/快捷键，不碰业务逻辑)
│   └── src/         # React 前端 (所有业务逻辑)
└── docs/            # 基础概念、协议、架构、部署、扩容
```

## 开发命令

```bash
# 先复制 server/configs/users.example.toml 为 server/configs/users.toml，填入 secret 和账号

# SFU + 信令（与调度共用集群表与用户名单）
cd server && go run ./cmd/server -cluster-config configs/cluster.example.toml -node n1 -users configs/users.toml

# 调度器（客户端进房前必问；单节点也要跑）
cd server && go run ./cmd/dispatch -config configs/cluster.example.toml -users configs/users.toml

# 客户端（Tauri 桌面模式；VITE_DISPATCH_URL 可选，默认 http://127.0.0.1:18090，登录页「服务器地址」可覆盖）
cd client && npm run tauri dev

# 客户端（纯 Web 开发模式，无需 Rust 环境）
cd client && npm run dev

# 桌面端安装包（当前系统；读 client/.env 的 VITE_DISPATCH_URL）
bash scripts/build-desktop.sh
```

默认 `VITE_DISPATCH_URL=http://127.0.0.1:18090`。用户可在登录页「服务器地址」修改；成功登录后写入 localStorage，优先于构建默认。

## 编码规范

### Go (server/)

- 遵循 [Effective Go](https://go.dev/doc/effective_go)，`gofmt` 格式化
- 使用 `internal/` 隔离实现细节，仅 `pkg/` 下的代码可被外部引用
- 错误处理显式返回 `error`，不用 panic 做流程控制
- 日志使用 `log/slog` 结构化日志

### TypeScript (client/src/)

- 严格模式 (`"strict": true`)
- 函数组件 + Hooks，禁止 class 组件
- 组件文件名 PascalCase (`UserCard.tsx`)，工具函数 camelCase (`useAudio.ts`)
- 类型定义集中在 `types/` 目录

### Rust (client/src-tauri/)

- Rust 层**仅负责系统集成**：窗口管理、系统托盘、全局快捷键
- 禁止在 Rust 层写业务逻辑，确保前端代码 Web 和 Tauri 通用

## 架构约束

1. **SFU 只转发不混音** — 服务端收到一个 Peer 的音频 Track 后，直接转发给同房间其他 Peer，不做音频处理
2. **信令协议文档优先** — 所有 WebSocket 消息格式统一在 `docs/protocol.md` 定义，Go 端 `pkg/proto/messages.go` 和 TS 端 `types/protocol.ts` 从该文档派生
3. **前端逻辑与 Tauri 解耦** — `hooks/` 和 `lib/` 中的代码不直接依赖 `@tauri-apps/api`，Tauri 特有功能通过条件检测 (`window.__TAURI__`) 启用
4. **房间状态纯内存** — MVP 阶段不引入数据库，房间和用户信息存在 Go 进程内存中

## 关键设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| SFU 引擎 | Pion (Go) | 部署简单，单二进制，与后端语言统一 |
| 桌面框架 | Tauri 2.0 | 安装包 <10MB，内存占用 ~50MB，远优于 Electron |
| 状态管理 | Zustand | 状态简单（房间+用户+音频），无需 Redux 重型方案 |
| 信令传输 | 原生 WebSocket | 语音信令只需轻量双向通信，Socket.IO 过重 |
| 进房调度 | 独立进程 + 共享静态集群表 | 客户端问调度拿 signaling；SFU 用同表 `public_ip`/`udp_port`；暂不做动态注册 |
| WebRTC 协商 | 服务端发起 offer | 客户端只回 answer，避免双向协商状态机 |
| CSS 方案 | Tailwind | 快速原型，无需维护独立样式文件 |

## 带宽参考

5 人语音 (Opus 32kbps, SFU 架构)：
- 服务端上行峰值 ≈ 0.83 Mbps（所有人同时说话）
- 3M 带宽轻量云完全够用，余量约 2/3

## Agent 角色定义

项目按职责划分了三个 Agent 角色提示词，位于 `.agents/agents/`（通用目录，跨工具可用）：

- [client-dev.md](.agents/agents/client-dev.md) — Tauri + React 前端开发
- [server-dev.md](.agents/agents/server-dev.md) — Go 服务端开发（SFU / 信令 / 房间）
- [protocol-design.md](.agents/agents/protocol-design.md) — 信令协议设计与前后端类型同步

修改信令协议时必须同步 `docs/protocol.md`、`server/pkg/proto/messages.go`、`client/src/types/protocol.ts` 三处，详见 protocol-design 角色定义。
