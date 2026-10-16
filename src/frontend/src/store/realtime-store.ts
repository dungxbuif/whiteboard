import { create } from "zustand";
import io from "socket.io-client";

export type RealtimeStatus = "idle" | "connecting" | "connected" | "disconnected" | "error";

export interface PeerPresence {
  userId: string;
  status?: "online" | "offline";
}

export interface RemoteCursor {
  userId: string;
  x: number;
  y: number;
  name?: string;
}

export interface RemoteSceneUpdate {
  elements: unknown[];
  appState: Record<string, unknown>;
  files: Record<string, unknown>;
  presentationOrder?: string[];
}

interface RealtimeState {
  status: RealtimeStatus;
  peers: Record<string, PeerPresence>;
  cursors: Record<string, RemoteCursor>;
  latestSceneUpdate: RemoteSceneUpdate | null;
  voiceMuted: boolean;
  connect: (workspaceId: string, sceneId: string) => void;
  disconnect: () => void;
  publish: (event: string, payload: Record<string, unknown>) => void;
  toggleVoiceMute: () => Promise<void>;
  clearLatestSceneUpdate: () => void;
  reset: () => void;
}

let socket: SocketIOClient.Socket | null = null;
let activeRoom: { workspaceId: string; sceneId: string } | null = null;
let localStream: MediaStream | null = null;
const peerConnections = new Map<string, RTCPeerConnection>();
let rtcConfiguration: RTCConfiguration = {};

const peerConnectionFor = (peerId: string, publish: (event: string, payload: Record<string, unknown>) => void) => {
  const existing = peerConnections.get(peerId);
  if (existing) return existing;
  const peer = new RTCPeerConnection(rtcConfiguration);
  localStream?.getTracks().forEach((track) => peer.addTrack(track, localStream as MediaStream));
  peer.onicecandidate = (event) => {
    if (event.candidate) publish("voice:ice", { to: peerId, candidate: event.candidate });
  };
  peer.ontrack = (event) => {
    const audio = new Audio();
    audio.autoplay = true;
    audio.srcObject = event.streams[0];
    void audio.play().catch(() => undefined);
  };
  peerConnections.set(peerId, peer);
  return peer;
};

const socketURL = () => {
  const apiURL = import.meta.env.VITE_API_URL || window.location.origin + "/api/v1";
  return apiURL.replace(/\/api\/v1\/?$/, "");
};

const useRealtimeStore = create<RealtimeState>((set, get) => ({
  status: "idle",
  peers: {},
  cursors: {},
  latestSceneUpdate: null,
  voiceMuted: false,

  connect: (workspaceId, sceneId) => {
    if (activeRoom?.workspaceId === workspaceId && activeRoom.sceneId === sceneId && socket) return;
    get().disconnect();
    activeRoom = { workspaceId, sceneId };
    set({ status: "connecting", peers: {}, cursors: {} });
    socket = io(socketURL(), {
      path: "/socket.io/",
      transports: ["websocket", "polling"],
      query: { token: localStorage.getItem("auth_token") || "" },
      autoConnect: true,
    });
    socket.on("connect", () => {
      set({ status: "connected" });
      socket?.emit("scene:join", { workspaceId, sceneId });
    });
    socket.on("disconnect", () => set({ status: "disconnected" }));
    socket.on("connect_error", () => set({ status: "error" }));
    socket.on("presence:update", (message: { data?: PeerPresence }) => {
      const peer = message?.data;
      if (!peer?.userId) return;
      set((state) => {
        if (peer.status === "offline") {
          const peers = { ...state.peers };
          delete peers[peer.userId];
          return { peers };
        }
        return { peers: { ...state.peers, [peer.userId]: { ...peer, status: "online" } } };
      });
    });
    socket.on("cursor:move", (message: { data?: RemoteCursor }) => {
      const cursor = message?.data;
      if (!cursor?.userId) return;
      set((state) => ({ cursors: { ...state.cursors, [cursor.userId]: cursor } }));
    });
    // Scene content is owned by the admitted Yjs transport; legacy sockets carry presence/voice only.
    socket.on("voice:offer", async (message: { data?: { from?: string; to?: string; sdp?: RTCSessionDescriptionInit } }) => {
      const data = message?.data;
      if (!data?.from || data.to && data.to !== socket?.id || !data.sdp) return;
      const peer = peerConnectionFor(data.from, get().publish);
      await peer.setRemoteDescription(new RTCSessionDescription(data.sdp));
      const answer = await peer.createAnswer();
      await peer.setLocalDescription(answer);
      get().publish("voice:answer", { to: data.from, from: socket?.id, sdp: answer });
    });
    socket.on("voice:answer", async (message: { data?: { from?: string; to?: string; sdp?: RTCSessionDescriptionInit } }) => {
      const data = message?.data;
      if (!data?.from || data.to && data.to !== socket?.id || !data.sdp) return;
      const peer = peerConnections.get(data.from);
      if (peer) await peer.setRemoteDescription(new RTCSessionDescription(data.sdp));
    });
    socket.on("voice:ice", async (message: { data?: { from?: string; to?: string; candidate?: RTCIceCandidateInit } }) => {
      const data = message?.data;
      if (!data?.from || data.to && data.to !== socket?.id || !data.candidate) return;
      const peer = peerConnectionFor(data.from, get().publish);
      await peer.addIceCandidate(new RTCIceCandidate(data.candidate));
    });
  },

  disconnect: () => {
    socket?.disconnect();
    socket = null;
    activeRoom = null;
    peerConnections.forEach((peer) => peer.close());
    peerConnections.clear();
    localStream?.getTracks().forEach((track) => track.stop());
    localStream = null;
    set({ status: "idle", peers: {}, cursors: {}, latestSceneUpdate: null });
  },

  publish: (event, payload) => {
    if (event === "scene:update") return;
    if (!socket || !activeRoom) return;
    socket.emit(event, { ...activeRoom, data: payload });
  },

  toggleVoiceMute: async () => {
    if (!localStream) {
      const apiBase = import.meta.env.VITE_API_URL || "/api/v1";
      try {
        const configResponse = await fetch(`${apiBase}/realtime/config`, {
          headers: { Authorization: `Bearer ${localStorage.getItem("auth_token") || ""}` },
        });
        if (configResponse.ok) rtcConfiguration = await configResponse.json() as RTCConfiguration;
      } catch {
        // Voice still works through the browser's direct ICE candidates.
      }
      localStream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const state = get();
      Object.keys(state.peers).forEach(async (peerId) => {
        if (peerId === socket?.id) return;
        const peer = peerConnectionFor(peerId, state.publish);
        const offer = await peer.createOffer();
        await peer.setLocalDescription(offer);
        state.publish("voice:offer", { to: peerId, from: socket?.id, sdp: offer });
      });
      set({ voiceMuted: false });
      return;
    }
    const nextMuted = !get().voiceMuted;
    localStream.getAudioTracks().forEach((track) => { track.enabled = !nextMuted; });
    set({ voiceMuted: nextMuted });
  },
  clearLatestSceneUpdate: () => set({ latestSceneUpdate: null }),
  reset: () => {
    get().disconnect();
    set({ voiceMuted: false, latestSceneUpdate: null });
  },
}));

export default useRealtimeStore;
