# WebRTC / SFU 基础概念

本文解释 Wavecall 里常用的网络与媒体概念，帮助区分「调度」「信令」「媒体」三条线。实现细节见 [架构设计](architecture.md)、[信令协议](protocol.md)。

## 三条线先分清

```
调度（HTTP）     → 这个房间连哪台 SFU？（只问路，不传声音）
信令（WebSocket） → 进房、交换 SDP / ICE candidate、静音状态
媒体（UDP）       → 真正的 Opus 音频包
```

客户端进房顺序：先问调度 → 再连该 SFU 的 WebSocket → 再用 ICE 打通 UDP。

---

## SFU

**SFU（Selective Forwarding Unit）**：选择性转发单元。

房间中间的媒体服务器。每个人把自己的音频推给 SFU，SFU **原样转发**给其他人，不做混音、不做降噪。

| 模型 | 做法 | 特点 |
|------|------|------|
| P2P | 人与人直连 | 人一多连接数爆炸 |
| MCU | 服务器混音后再下发 | CPU 重、延迟通常更高 |
| **SFU** | 服务器只转发各路流 | 适合小房间低延迟语音 |

Wavecall 的 SFU 是 Go + Pion：收到 A 的 RTP，复制发给 B、C。

---

## PeerConnection

`PeerConnection` 是浏览器 / Pion 里的一条 **WebRTC 连接对象**。

- 客户端本地一条 `RTCPeerConnection`
- 服务端每个进房用户对应一条（`sfu.Peer` 内的 pion PC）

它负责：

1. 用 SDP 协商传什么（音频轨、Opus 等）
2. 用 ICE 找出能通的 UDP 路径
3. 连通后收发 RTP 媒体

可以把它想成「我和这台 SFU 之间的专用媒体管道」。房间 3 个人 = 服务端 3 条 PeerConnection；SFU 在这几条之间转发，不混音。

---

## WebSocket 信令做什么

连上 SFU 的 `/ws` 之后，这条连接只传 **JSON 信令**，不传 Opus。

常见消息：

| 方向 | 类型 | 作用 |
|------|------|------|
| C→S | `join` / `leave` | 进房、离房 |
| S→C | `joined` / `peerJoined` / `peerLeft` / `peerMuted` | 房间状态 |
| S→C | `offer` | 服务端发起协商（初始 + 重协商） |
| C→S | `answer` | 回复 offer |
| 双向 | `candidate` | 交换 ICE 候选 |
| C→S | `mute` / `unmute` | 静音状态 |

Wavecall 约定：**服务端发 offer，客户端只回 answer**，客户端从不发 offer。

---

## SDP

**SDP（Session Description Protocol）**：会话描述协议。

一段文本，描述「这次通话怎么建」——音频/视频、编码能力等。协议里 `offer` / `answer` 的 `sdp` 字段就是它。

流程：

1. SFU `createOffer` → 得到 SDP → WebSocket 发给客户端
2. 客户端设为远端描述 → `createAnswer` → 把自己的 SDP 发回
3. 双方再靠 ICE candidate 把实际 UDP 地址补齐

SDP 主要谈「媒体会话长什么样」；「包发到哪个 IP:端口」常常由 Trickle ICE 的 `candidate` 另传。

---

## ICE 与 host 候选

**ICE（Interactive Connectivity Establishment）**：交互式连通建立。

两边都知道要传音频，但还不知道对端的公网 IP 和 UDP 端口。ICE 收集「我可以在这些地址收包」的候选，交给对方试连通。

**ICE host 候选**：本机网卡地址直接报出的候选（不是经 STUN 反射、也不是 TURN 中继）。

云主机上进程看到的常是内网 IP（如 `10.0.0.8`），外面连不到。Wavecall 用集群表里的 `public_ip`（或 `-public-ip`）经 pion `SetNAT1To1IPs` 写进 host 候选，再配合 `udp_port`（默认 18081），对端看到的是类似 `公网IP:18081` 的 UDP 地址。

你们实现里服务器用 **UDPMux**：多条 PeerConnection 共用同一个 UDP 端口，不是每人占一个服务端口。

候选可能比 answer 先到，服务端会先缓存，等远端描述设好再灌进去。

---

## STUN

**STUN（Session Traversal Utilities for NAT）**：帮你从 NAT 后面查出「外面看我是什么公网地址」。

1. 程序向 STUN 服务器发 UDP 请求  
2. STUN 看到来源公网 IP:端口并告诉你  
3. 你把该地址写成 ICE 候选（常称 **srflx**，server reflexive）

STUN **只发现地址，不转发媒体**。  
Wavecall 默认不依赖外部 STUN；云上 1:1 NAT 场景用表内 `public_ip` 直接声明。`-stun` 仍可作为可选回退。

---

## TURN

**TURN（Traversal Using Relays around NAT）**：中继服务器。

STUN / 直连失败时（对称 NAT、严格企业网等），两端都连 TURN，音频走：

`A → TURN → B`

| | 干什么 | 媒体怎么走 |
|--|--------|------------|
| STUN | 查出公网地址 | 仍尽量直连 |
| TURN | 提供中继 | 必须经 TURN 转发 |
| SFU | 房间内业务转发 | 人 ↔ SFU ↔ 人 |

SFU ≠ TURN。SFU 是应用层「房间媒体转发」；TURN 是 ICE 打洞失败时的网络层兜底。配置里预留了 TURN 字段，当前未接线。

---

## 调度服务

独立进程 `cmd/dispatch`，默认 `:18090`。

客户端：

```http
GET /dispatch?room=<roomId>
→ {"node":"host:port"}
```

`host:port` 来自共享集群表的 **`signaling`**，客户端只用来拼 WebSocket（`ws://host:port/ws`）。

调度：

- 不在媒体路径上  
- 不写 ICE / SDP  
- 当前是静态表 + 房间哈希，不做动态注册  

SFU 用 **同一份** `cluster.toml`，通过 `-cluster-config` + `-node` 读取自己的 `public_ip` / `udp_port`。  
调度返回的地址 ≠ ICE 候选；两者常常对应同一台机器，但是两条配置用途。

| 来源 | 字段 | 用途 |
|------|------|------|
| 集群表 `signaling` | `host:18080` | 只连 WebSocket |
| 集群表 `public_ip` + `udp_port` | IP + UDP | 只写 ICE host 候选 |

详见 [部署指南](deployment.md)、[多节点扩容](scaling.md)。

---

## 域名和 IP

正式环境 **信令侧** 一般用域名（`VITE_DISPATCH_URL`、`signaling = "sfu1.example.com:18080"`、`wss://`）。

**媒体侧（ICE 候选）** 正规产品同样使用 **公网 IP:端口**。WebRTC / pion 的 host 候选本质上是 IP，不是靠域名发音频。

常见形态：

- 域名：给人、给浏览器找信令入口  
- `public_ip`：该 SFU 实例在 ICE 里报的媒体地址  
- DNS：`sfu1.example.com` → 同一台机器的公网 IP  

动态扩缩容时，变的是「节点如何登记进调度表」，不是「ICE 改成域名」。每台 SFU 仍要有自己的公网 IP（或日后用 STUN 自测），媒体候选还是 IP。

---

## 一次进房串起来

```
1. GET 调度 /dispatch?room=xxx
   ← {"node":"sfu1.example.com:18080"}

2. WebSocket → ws(s)://sfu1.example.com:18080/ws
   → join
   ← joined（userId、peers、iceServers）
   ← offer（SDP）
   → answer（SDP）
   ↔ candidate（ICE，含 SFU 的 public_ip:udp_port）

3. UDP 打通后
   麦克风 RTP ↔ SFU :udp_port ↔ 房间其他人
```

记三个词：**调度选门、WebSocket 商量、UDP 传声**。
