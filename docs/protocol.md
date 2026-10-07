# Wavecall 信令协议

客户端与 SFU 的通话信令通过 WebSocket 传输 JSON 消息。每条消息必须包含 `type` 字段。

进房前客户端先向调度器发 `GET /dispatch?room=<roomId>`，拿到 `{"node":"host:port"}` 后再对该节点建立 WebSocket。调度是独立 HTTP 接口，不在下面的消息类型里，协议里也没有 `redirect`。相关名词见 [基础概念](concepts.md)。

> SFU 模式下 WebRTC 协商由**服务端发起**：offer 为 S→C，answer 为 C→S，
> candidate 双向（Trickle ICE）。客户端从不发送 offer。

## 消息总览

| 方向 | type | 说明 |
|------|------|------|
| C→S | `join` | 加入房间 |
| C→S | `leave` | 离开房间 |
| C→S | `answer` | 回复服务端 SDP Offer |
| C→S | `candidate` | 发送 ICE Candidate |
| C→S | `mute` | 静音 |
| C→S | `unmute` | 取消静音 |
| S→C | `joined` | 确认已加入房间 |
| S→C | `offer` | 发起 SDP Offer（初始 + 每次重协商） |
| S→C | `candidate` | 下发 ICE Candidate |
| S→C | `peerJoined` | 新用户加入 |
| S→C | `peerLeft` | 用户离开 |
| S→C | `peerMuted` | 用户静音状态变化 |
| S→C | `speaking` | 用户说话状态变化 |
| S→C | `error` | 错误 |

---

## Client → Server

### join

客户端请求加入一个房间。如果房间不存在则自动创建。

```json
{
  "type": "join",
  "roomId": "room-123",
  "userName": "玩家A",
  "password": "optional-password"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| roomId | string | ✅ | 房间 ID |
| userName | string | ✅ | 显示昵称 |
| password | string | ❌ | 房间密码，无密码则为空 |

### leave

```json
{ "type": "leave" }
```

### answer

回复服务端发起的 SDP Offer。`targetId` 填自己的 userId（服务端忽略该字段）。

```json
{
  "type": "answer",
  "targetId": "u_abc123",
  "sdp": "v=0\r\no=- ..."
}
```

### candidate

发送 ICE Candidate（Trickle ICE）。候选可能在 answer 之前到达，服务端会先缓存。

`sdpMid` 与 `sdpMLineIndex` 至少传一个（浏览器 `addIceCandidate` 要求）；通常两者都带上。

```json
{
  "type": "candidate",
  "targetId": "u_abc123",
  "candidate": "candidate:842163049 1 udp ...",
  "sdpMid": "0",
  "sdpMLineIndex": 0
}
```

### mute / unmute

```json
{ "type": "mute" }
{ "type": "unmute" }
```

---

## Server → Client

### joined

确认加入成功，返回房间信息、已有成员列表和 ICE 服务器配置。

配置了节点 `public_ip`（或 `-public-ip`）时 `iceServers` 通常为 `[]`，媒体地址在服务端 SDP 的 host 候选里。只有配置了 `-stun` 或 TURN 时才会带上下面这种条目。

```json
{
  "type": "joined",
  "userId": "u_abc123",
  "roomId": "room-123",
  "peers": [
    { "userId": "u_def456", "userName": "玩家B", "muted": false },
    { "userId": "u_ghi789", "userName": "玩家C", "muted": true }
  ],
  "iceServers": []
}
```

### offer

服务端发起 SDP Offer。紧随 `joined` 之后的是初始协商（已包含房间内所有已发布的音频轨）；
之后每当有新成员发布音频，服务端会向房间内其他成员各发送一次重协商 offer。

```json
{
  "type": "offer",
  "targetId": "u_abc123",
  "sdp": "v=0\r\no=- ..."
}
```

### candidate

下发服务端 ICE Candidate（Trickle ICE），`targetId` 为接收者的 userId。字段同 C→S `candidate`（含 `sdpMid` / `sdpMLineIndex`）。

```json
{
  "type": "candidate",
  "targetId": "u_abc123",
  "candidate": "candidate:842163049 1 udp ...",
  "sdpMid": "0",
  "sdpMLineIndex": 0
}
```

### peerJoined

新用户加入房间。

```json
{
  "type": "peerJoined",
  "userId": "u_new001",
  "userName": "玩家D"
}
```

### peerLeft

用户离开房间。

```json
{
  "type": "peerLeft",
  "userId": "u_def456"
}
```

### peerMuted

用户静音状态变化。

```json
{
  "type": "peerMuted",
  "userId": "u_def456",
  "muted": true
}
```

### speaking

用户说话状态变化（由服务端音量检测或客户端上报）。

```json
{
  "type": "speaking",
  "userId": "u_def456",
  "speaking": true
}
```

### error

```json
{
  "type": "error",
  "code": "ROOM_FULL",
  "message": "房间已满（最多5人）"
}
```

错误码：

| code | 说明 |
|------|------|
| `INVALID_PARAM` | 房间号或昵称为空 |
| `WRONG_PASSWORD` | 密码错误 |
| `ROOM_FULL` | 房间已满（最多 5 人） |
| `PEER_SETUP_FAILED` | SFU PeerConnection 或初始协商失败 |

---

## 连接流程

```
客户端 A          调度器
  │                 │
  │── GET /dispatch?room= ──▶│
  │◀── {"node":"host:port"} ─│

客户端 A                        SFU                          客户端 B
  │                               │                            │
  │──── WebSocket connect ───────▶│                            │
  │──── join { roomId, userName } ▶│ 创建/查找房间，分配 userId
  │◀── joined { userId, peers } ──│──▶ peerJoined → 其他客户端
  │                               │                            │
  │◀── offer (初始, S→C) ─────────│ 创建 PeerConnection
  │──── answer ──────────────────▶│                            │
  │◀──▶ candidate (双向 Trickle) ─▶│                            │
  │                               │                            │
  │ ═══ A 的音频流上传到 SFU ═════ │                            │
  │                               │                            │
  │                               │      B 加入（同上流程，      │
  │                               │      初始 offer 已含 A 的轨） │
  │                               │◀── answer ─────────────────│
  │                               │                            │
  │◀── offer (重协商, 携带 B 的轨) ─│──▶ offer (重协商) ─────────▶│
  │──── answer ──────────────────▶│◀── answer ─────────────────│
  │                               │                            │
  │ ◀══ SFU 转发 B 的音频 ════════════ SFU 转发 A 的音频 ═══▶ │
  │                               │                            │
  │──── mute ────────────────────▶│──▶ peerMuted → 其他客户端
  │──── leave ───────────────────▶│──▶ peerLeft → 其他客户端
  │◀── WebSocket close ──────────│
```
