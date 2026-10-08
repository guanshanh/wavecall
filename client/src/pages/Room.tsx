import { useParams, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useCallback, useState, useRef } from "react";
import { useRoomStore } from "../stores/roomStore";
import { useAuthStore } from "../stores/authStore";
import { useSignaling } from "../hooks/useSignaling";
import { useWebRTC } from "../hooks/useWebRTC";
import { AudioLevelMonitor } from "../lib/audioLevels";
import type { ClientMessage, ServerMessage } from "../types/protocol";
import { resolveSignalingWsUrl } from "../lib/dispatch";
import type { ConnectionState } from "../lib/signaling";
import UserCard from "../components/UserCard";
import Controls from "../components/Controls";

export default function Room() {
  const { roomId } = useParams<{ roomId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { userName, password } = (location.state as { userName?: string; password?: string }) || {};

  const peers = useRoomStore((s) => s.peers);
  const isMuted = useRoomStore((s) => s.isMuted);
  const connected = useRoomStore((s) => s.connected);
  const setConnected = useRoomStore((s) => s.setConnected);
  const setPeers = useRoomStore((s) => s.setPeers);
  const addPeer = useRoomStore((s) => s.addPeer);
  const removePeer = useRoomStore((s) => s.removePeer);
  const setPeerMuted = useRoomStore((s) => s.setPeerMuted);
  const toggleMuteStore = useRoomStore((s) => s.toggleMute);
  const resetStore = useRoomStore((s) => s.reset);

  const [connectionStatus, setConnectionStatus] = useState<string>("正在连接服务器...");
  const [remoteStreams, setRemoteStreams] = useState<Map<string, MediaStream>>(new Map());

  // Refs to bridge callback wiring between the two hooks:
  // the WebRTC callbacks need `send` and `userId`, which only exist after
  // useSignaling/useRoomStore resolve — refs let later values reach earlier closures.
  const userId = useRoomStore((s) => s.userId);
  const sendRef = useRef<(msg: ClientMessage) => void>(() => {});
  const disconnectRef = useRef<() => void>(() => {});
  const token = useAuthStore((s) => s.token);
  const clearAuth = useAuthStore((s) => s.clear);
  const userIdRef = useRef<string | null>(null);
  userIdRef.current = userId;

  const { init, enableMic, handleServerOffer, addIceCandidate, setMuted, setCallbacks, reset } =
    useWebRTC();

  // 音量检测：本地麦克风与每一路远端流共用一个 AudioLevelMonitor
  const monitorRef = useRef<AudioLevelMonitor | null>(null);
  const [audioLevels, setAudioLevels] = useState<Map<string, number>>(new Map());
  const getMonitor = useCallback(() => {
    monitorRef.current ??= new AudioLevelMonitor();
    return monitorRef.current;
  }, []);

  // 组件卸载时停止检测并释放 AudioContext
  useEffect(() => () => monitorRef.current?.stop(), []);

  const attachRemoteTrack = useCallback((userId: string, track: MediaStreamTrack) => {
    const stream = new MediaStream([track]);
    monitorRef.current?.addStream(userId, stream);
    setRemoteStreams((prev) => {
      const next = new Map(prev);
      next.set(userId, stream);
      return next;
    });
  }, []);

  // 首次收到 joined 后置位；断线重连成功时据此触发重新进房
  const joinedOnceRef = useRef(false);
  const dispatchFailedRef = useRef(false);

  const rejoin = useCallback(() => {
    reset();
    monitorRef.current?.clear();
    setRemoteStreams(new Map());
    if (roomId && userName && token) {
      sendRef.current({ type: "join", roomId, userName, password, token });
    }
  }, [reset, roomId, userName, password, token]);

  // WebRTC → 信令出口：answer / candidate 发回服务端
  setCallbacks({
    onAnswer: (sdp) =>
      sendRef.current({ type: "answer", targetId: userIdRef.current ?? "", sdp }),
    onCandidate: (init) =>
      sendRef.current({
        type: "candidate",
        targetId: userIdRef.current ?? "",
        candidate: init.candidate ?? "",
        sdpMid: init.sdpMid ?? undefined,
        sdpMLineIndex: init.sdpMLineIndex ?? undefined,
      }),
    onRemoteTrack: attachRemoteTrack,
  });

  const getSignalingUrl = useCallback(async () => {
    if (!roomId) {
      throw new Error("missing roomId");
    }
    try {
      const url = await resolveSignalingWsUrl(roomId);
      dispatchFailedRef.current = false;
      return url;
    } catch (e) {
      dispatchFailedRef.current = true;
      setConnectionStatus("无法联系调度服务，正在重试...");
      throw e;
    }
  }, [roomId]);

  // 消息处理函数
  const handleMessage = useCallback(
    async (msg: ServerMessage) => {
      switch (msg.type) {
        case "joined":
          joinedOnceRef.current = true;
          setConnected(true, msg.roomId, msg.userId);
          setConnectionStatus("已连接");
          setPeers(
            msg.peers.map((p) => ({
              userId: p.userId,
              userName: p.userName,
              muted: p.muted,
            })),
          );
          // 初始化 PeerConnection 并预取麦克风权限，
          // 服务端的 offer 会紧接着到达
          init(msg.iceServers);
          getMonitor().start(setAudioLevels);
          enableMic()
            .then((stream) => {
              if (stream) monitorRef.current?.addStream(msg.userId, stream);
            })
            .catch((err) => console.error("Microphone init failed:", err));
          break;

        case "offer":
          try {
            await handleServerOffer(msg.sdp);
          } catch (err) {
            console.error("Failed to handle server offer:", err);
          }
          break;

        case "candidate":
          try {
            await addIceCandidate({
              candidate: msg.candidate,
              sdpMid: msg.sdpMid,
              sdpMLineIndex: msg.sdpMLineIndex,
            });
          } catch (err) {
            console.error("Failed to add ICE candidate:", err);
          }
          break;

        case "peerJoined":
          addPeer({
            userId: msg.userId,
            userName: msg.userName,
            muted: false,
          });
          break;

        case "peerLeft":
          removePeer(msg.userId);
          monitorRef.current?.removeStream(msg.userId);
          setRemoteStreams((prev) => {
            const next = new Map(prev);
            next.delete(msg.userId);
            return next;
          });
          break;

        case "peerMuted":
          setPeerMuted(msg.userId, msg.muted);
          break;

        case "error":
          if (msg.code === "UNAUTHORIZED") {
            clearAuth();
            disconnectRef.current();
            resetStore();
            navigate("/");
            break;
          }
          alert(`进入房间失败: ${msg.message}`);
          resetStore();
          navigate("/");
          break;

        default:
          break;
      }
    },
    [
      navigate,
      clearAuth,
      setConnected,
      setPeers,
      addPeer,
      removePeer,
      setPeerMuted,
      resetStore,
      init,
      enableMic,
      handleServerOffer,
      addIceCandidate,
      getMonitor,
    ],
  );

  const handleSignalingStateChange = useCallback((state: ConnectionState) => {
    const dispatchRetryStatus = "无法联系调度服务，正在重试...";
    if (state === "connected") {
      setConnectionStatus("已连接");
      // 断线重连成功：重建 WebRTC 并重新进房（服务端会分配新 userId）
      if (joinedOnceRef.current) {
        rejoin();
      }
    } else if (state === "reconnecting") {
      setConnectionStatus(
        dispatchFailedRef.current ? dispatchRetryStatus : "连接断开，正在重连...",
      );
    } else if (state === "connecting") {
      if (dispatchFailedRef.current) {
        setConnectionStatus(dispatchRetryStatus);
      } else {
        setConnectionStatus(joinedOnceRef.current ? "正在重连..." : "正在连接服务器...");
      }
    } else if (state === "disconnected") {
      setConnectionStatus("已断开连接");
    } else if (state === "error") {
      setConnectionStatus("无法联系调度服务");
    }
  }, [rejoin]);

  const { send, disconnect } = useSignaling(
    getSignalingUrl,
    (msg) => void handleMessage(msg),
    handleSignalingStateChange,
  );

  sendRef.current = send;
  disconnectRef.current = disconnect;

  // 进入页面发送 join 请求
  useEffect(() => {
    if (!userName || !roomId || !token) {
      navigate("/");
      return;
    }

    send({
      type: "join",
      roomId,
      userName,
      password,
      token,
    });

    return () => {
      send({ type: "leave" });
      disconnect();
      resetStore();
    };
  }, [userName, roomId, password, token, navigate, send, disconnect, resetStore]);

  // 挂断退出
  const handleLeave = () => {
    send({ type: "leave" });
    disconnect();
    resetStore();
    navigate("/");
  };

  // 切换静音
  const handleToggleMute = () => {
    const nextMuted = !isMuted;
    toggleMuteStore();
    setMuted(nextMuted);
    send({ type: nextMuted ? "mute" : "unmute" });
  };

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      {/* Header */}
      <header className="bg-gray-800 px-6 py-4 flex items-center justify-between border-b border-gray-700">
        <div className="flex items-center gap-3">
          <span className="text-2xl">🌊</span>
          <div>
            <h1 className="text-white font-semibold text-lg flex items-center gap-2">
              房间 {roomId}
              {connected ? (
                <span className="w-2.5 h-2.5 rounded-full bg-green-500 inline-block" title="在线" />
              ) : (
                <span className="w-2.5 h-2.5 rounded-full bg-yellow-500 inline-block animate-pulse" title="连接中" />
              )}
            </h1>
            <p className="text-gray-400 text-xs">
              {peers.length + 1} 人在房间 · {connectionStatus}
            </p>
          </div>
        </div>
        <button
          onClick={handleLeave}
          className="text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 px-3 py-1.5 rounded-md transition-colors cursor-pointer"
        >
          返回大厅
        </button>
      </header>

      {/* Peer Grid */}
      <main className="flex-1 p-6 flex flex-col justify-center">
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4 max-w-4xl w-full mx-auto">
          {/* 自己 */}
          <UserCard
            userName={userName ?? "我"}
            isMuted={isMuted}
            isSelf={true}
            level={audioLevels.get(userId ?? "") ?? 0}
          />

          {/* 房间内的其他小伙伴 */}
          {peers.map((peer) => (
            <UserCard
              key={peer.userId}
              userName={peer.userName}
              isMuted={peer.muted}
              isSelf={false}
              level={audioLevels.get(peer.userId) ?? 0}
            />
          ))}
        </div>
      </main>

      {/* 远端音频：SFU 以发送者 userId 命名 MediaStream */}
      {[...remoteStreams.entries()].map(([userId, stream]) => (
        <audio
          key={userId}
          autoPlay
          ref={(el) => {
            if (el) {
              el.srcObject = stream;
              el.play().catch(() => {});
            }
          }}
        />
      ))}

      {/* Controls */}
      <Controls
        isMuted={isMuted}
        onToggleMute={handleToggleMute}
        onLeave={handleLeave}
      />
    </div>
  );
}
