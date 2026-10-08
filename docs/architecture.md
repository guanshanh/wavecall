# Wavecall 架构

SFU、PeerConnection、ICE、SDP、STUN/TURN、调度与媒体地址等名词说明见 [基础概念](concepts.md)。

## 总体架构

```
┌─────────────────────────────────────────────────────────┐
│                    客户端 (Tauri + React)                 │
│                                                          │
│  ┌──────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │  React   │  │  WebRTC API  │  │  Tauri 系统集成    │  │
│  │  UI 层   │  │  音频采集/播放 │  │  托盘/快捷键/窗口  │  │
│  └────┬─────┘  └──────┬───────┘  └───────────────────┘  │
│       │               │                                  │
│  ┌────▼───────────────▼────────┐                        │
│  │  先 HTTP 问调度，再 WebSocket │                        │
│  └──────────────┬──────────────┘                        │
└─────────────────┼───────────────────────────────────────┘
                  │
         ┌────────▼────────┐
         │   互联网 / NAT   │
         └────────┬────────┘
                  │
     ┌────────────▼────────────┐
     │  调度器 cmd/dispatch     │
     │  GET /dispatch?room=    │
     │  共享集群表，哈希选一台   │
     └────────────┬────────────┘
                  │ {"node":"host:port"}
┌─────────────────▼───────────────────────────────────────┐
│                  SFU 节点 (Go)                            │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │  HTTP Server                                      │   │
│  │  ├── /ws       → WebSocket 信令                   │   │
│  │  ├── /health   → 健康检查                         │   │
│  │  └── /         → 可选托管 Web 客户端（-web）       │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                    │
│  ┌──────────────────▼───────────────────────────────┐   │
│  │  Signaling Handler                                │   │
│  │  服务端发起 offer；answer / candidate 回到本连接   │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                    │
│  ┌──────────────────▼──────────┐ ┌──────────────────┐   │
│  │  Room Manager               │ │  SFU Router       │   │
│  │  房间与成员（纯内存）        │ │  Pion WebRTC      │   │
│  │                             │ │  RTP 原样转发      │   │
│  └─────────────────────────────┘ └──────────────────┘   │
│  媒体：集群表 udp_port + public_ip 写入 ICE host 候选     │
└─────────────────────────────────────────────────────────┘
```

## SFU 媒体流转发

```
玩家A ──audio──▶ ┌─────────┐ ──audio──▶ 玩家B
                 │         │ ──audio──▶ 玩家C
玩家B ──audio──▶ │   SFU   │ ──audio──▶ 玩家A
                 │ Router  │ ──audio──▶ 玩家C
玩家C ──audio──▶ │         │ ──audio──▶ 玩家A
                 └─────────┘ ──audio──▶ 玩家B

每个玩家上传 1 路音频，服务器转发给房间内其他所有人。
不做混音，纯转发。
```

## 数据流

### 进房与信令（先调度，再 WebSocket）

```
POST /login → { token, account }
GET /dispatch?room=… → {"node":"host:port"}
    → 对该节点建立 WebSocket
    → join（含 token 与房间密码）→ joined（含 userId、已有成员、iceServers）
    → 服务端发 offer（S→C），客户端回 answer（C→S）
    → candidate 双向 Trickle ICE
    → 新人发布音频时，服务端再发一次重协商 offer
```

客户端不发送 offer。`iceServers` 在配置了节点 `public_ip` 时通常为空，媒体地址写在服务端的 host 候选里。

### 媒体流（WebRTC / UDP）

```
麦克风 → getUserMedia → AudioTrack → PeerConnection
    → RTP over UDP → 服务器 Pion PeerConnection
    → Router 转发 → 其他客户端 PeerConnection → AudioTrack → 扬声器
```

## 模块职责

| 模块 | 位置 | 职责 |
|------|------|------|
| Config | `server/internal/config/` | SFU 端口、公网 IP、可选 STUN/TURN |
| Dispatch | `server/cmd/dispatch`、`server/internal/dispatch/` | 独立进程：读共享集群表的 signaling，按房间哈希返回 |
| Signaling | `server/internal/signaling/` | WebSocket 连接管理、消息解析、发起协商 |
| Room | `server/internal/room/` | 房间生命周期、成员管理 |
| SFU | `server/internal/sfu/` | PeerConnection 管理、RTP 转发 |
| Proto | `server/pkg/proto/` | 信令消息结构体定义 |

## 部署架构

单节点时调度器与 SFU 可以在同一台机器上，客户端路径与多节点相同。前端由 SFU 的 `-web` 托管，不单独放到静态站点。

```
浏览器 / Tauri
    │  GET /dispatch?room=
    ▼
调度器 :18090 与 SFU 共用 cluster.toml
    │  调度只返回 signaling
    │  {"node":"host:18080"}
    ▼
SFU 节点（-cluster-config + -node）
    :18080 TCP   信令 + 可选 Web 静态文件
    :18081 UDP   全部 WebRTC 媒体（UDPMux）
    ICE host 候选 = 表内 public_ip（TURN 未接线）
```

加节点只改共享集群表。房间仍整房落在被选中的那一台 SFU 上，节点之间不转发媒体。见 [多节点扩容](scaling.md)。
