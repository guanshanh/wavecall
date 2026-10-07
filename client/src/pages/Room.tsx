import { useParams, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useCallback, useState } from "react";
import { useRoomStore } from "../stores/roomStore";
import { useSignaling } from "../hooks/useSignaling";
import type { ServerMessage } from "../types/protocol";
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

  // WebSocket 服务器地址
  const wsUrl = `ws://${window.location.hostname || "localhost"}:8080/ws`;

  // 消息处理函数
  const handleMessage = useCallback(
    (msg: ServerMessage) => {
      switch (msg.type) {
        case "joined":
          setConnected(true, msg.roomId, msg.userId);
          setConnectionStatus("已连接");
          setPeers(
            msg.peers.map((p) => ({
              userId: p.userId,
              userName: p.userName,
              muted: p.muted,
              speaking: false,
            })),
          );
          break;

        case "peerJoined":
          addPeer({
            userId: msg.userId,
            userName: msg.userName,
            muted: false,
            speaking: false,
          });
          break;

        case "peerLeft":
          removePeer(msg.userId);
          break;

        case "peerMuted":
          setPeerMuted(msg.userId, msg.muted);
          break;

        case "error":
          alert(`进入房间失败: ${msg.message}`);
          resetStore();
          navigate("/");
          break;

        default:
          break;
      }
    },
    [navigate, setConnected, setPeers, addPeer, removePeer, setPeerMuted, resetStore],
  );

  const { send, disconnect } = useSignaling(
    wsUrl,
    handleMessage,
    (state) => {
      if (state === "error") {
        setConnectionStatus("连接异常，请检查服务端是否启动");
      } else if (state === "disconnected") {
        setConnectionStatus("已断开连接");
      }
    },
  );

  // 进入页面发送 join 请求
  useEffect(() => {
    if (!userName || !roomId) {
      navigate("/");
      return;
    }

    send({
      type: "join",
      roomId,
      userName,
      password,
    });

    return () => {
      send({ type: "leave" });
      disconnect();
      resetStore();
    };
  }, [userName, roomId, password, navigate, send, disconnect, resetStore]);

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
            speaking={false}
          />

          {/* 房间内的其他小伙伴 */}
          {peers.map((peer) => (
            <UserCard
              key={peer.userId}
              userName={peer.userName}
              isMuted={peer.muted}
              isSelf={false}
              speaking={peer.speaking}
            />
          ))}
        </div>
      </main>

      {/* Controls */}
      <Controls
        isMuted={isMuted}
        onToggleMute={handleToggleMute}
        onLeave={handleLeave}
      />
    </div>
  );
}
