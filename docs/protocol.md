# Wavecall 信令协议

所有客户端与服务端的通信通过 WebSocket 传输 JSON 消息。每条消息必须包含 `type` 字段。

## 消息总览

| 方向 | type | 说明 |
|------|------|------|
| C→S | `join` | 加入房间 |
| C→S | `leave` | 离开房间 |
| C→S | `offer` | 发送 SDP Offer |
| C→S | `answer` | 发送 SDP Answer |
| C→S | `candidate` | 发送 ICE Candidate |
| C→S | `mute` | 静音 |
| C→S | `unmute` | 取消静音 |
| S→C | `joined` | 确认已加入房间 |
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

### offer

发送 SDP Offer 给服务端，用于建立 WebRTC 连接。

```json
{
  "type": "offer",
  "targetId": "server",
  "sdp": "v=0\r\no=- ..."
}
```

### answer

回复 SDP Answer。

```json
{
  "type": "answer",
  "targetId": "server",
  "sdp": "v=0\r\no=- ..."
}
```

### candidate

发送 ICE Candidate（Trickle ICE）。

```json
{
  "type": "candidate",
  "targetId": "server",
  "candidate": "candidate:842163049 1 udp ..."
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

```json
{
  "type": "joined",
  "userId": "u_abc123",
  "roomId": "room-123",
  "peers": [
    { "userId": "u_def456", "userName": "玩家B", "muted": false },
    { "userId": "u_ghi789", "userName": "玩家C", "muted": true }
  ],
  "iceServers": [
    { "urls": ["stun:stun.l.google.com:19302"] },
    { "urls": ["turn:your-server:3478"], "username": "user", "credential": "pass" }
  ]
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
| `ROOM_FULL` | 房间已满 |
| `WRONG_PASSWORD` | 密码错误 |
| `ROOM_NOT_FOUND` | 房间不存在 |
| `INVALID_MESSAGE` | 消息格式错误 |

---

## 连接流程

```
客户端                          服务端
  │                               │
  │──── WebSocket connect ───────▶│
  │                               │
  │──── join { roomId, userName } ▶│
  │                               │ 创建/查找房间，分配 userId
  │◀── joined { userId, peers } ──│
  │                               │ 通知房间内其他人
  │                               │──▶ peerJoined → 其他客户端
  │                               │
  │──── offer { sdp } ───────────▶│
  │◀── answer { sdp } ───────────│ SFU 建立 PeerConnection
  │                               │
  │◀──▶ candidate (双向) ◀──▶│ Trickle ICE
  │                               │
  │ ═══ 音频流通过 WebRTC ════════ │
  │                               │
  │──── mute ────────────────────▶│
  │                               │──▶ peerMuted → 其他客户端
  │                               │
  │──── leave ───────────────────▶│
  │                               │──▶ peerLeft → 其他客户端
  │◀── WebSocket close ──────────│
```
