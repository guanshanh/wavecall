import { create } from "zustand";

export interface PeerState {
  userId: string;
  userName: string;
  muted: boolean;
}

interface RoomState {
  // Connection state
  connected: boolean;
  roomId: string | null;
  userId: string | null;

  // Self state
  isMuted: boolean;

  // Peers
  peers: PeerState[];

  // Actions
  setConnected: (connected: boolean, roomId: string, userId: string) => void;
  setPeers: (peers: PeerState[]) => void;
  addPeer: (peer: PeerState) => void;
  removePeer: (userId: string) => void;
  setPeerMuted: (userId: string, muted: boolean) => void;
  toggleMute: () => void;
  reset: () => void;
}

export const useRoomStore = create<RoomState>((set) => ({
  connected: false,
  roomId: null,
  userId: null,
  isMuted: false,
  peers: [],

  setConnected: (connected, roomId, userId) =>
    set({ connected, roomId, userId }),

  setPeers: (peers) => set({ peers }),

  addPeer: (peer) =>
    set((state) => ({
      peers: [...state.peers, peer],
    })),

  removePeer: (userId) =>
    set((state) => ({
      peers: state.peers.filter((p) => p.userId !== userId),
    })),

  setPeerMuted: (userId, muted) =>
    set((state) => ({
      peers: state.peers.map((p) =>
        p.userId === userId ? { ...p, muted } : p,
      ),
    })),

  toggleMute: () => set((state) => ({ isMuted: !state.isMuted })),

  reset: () =>
    set({
      connected: false,
      roomId: null,
      userId: null,
      isMuted: false,
      peers: [],
    }),
}));
