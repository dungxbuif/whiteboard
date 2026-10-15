import { create } from "zustand";

import api from "@/lib/api";
import type { SceneContent } from "@/store/scene-store";

type AIKind = "chat" | "diagram" | "code";

interface AIState {
  isLoading: boolean;
  error: string | null;
  result: { kind: AIKind; content: unknown; model?: string } | null;
  runChat: (prompt: string, imageDataUrl?: string) => Promise<void>;
  generateDiagram: (prompt: string, imageDataUrl?: string) => Promise<SceneContent | null>;
  toCode: (prompt: string, scene: SceneContent) => Promise<void>;
  reset: () => void;
}

const useAIStore = create<AIState>((set) => ({
  isLoading: false,
  error: null,
  result: null,
  runChat: async (prompt, imageDataUrl) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.post("/ai/chat", { prompt, imageDataUrl });
      set({ result: response.data, isLoading: false });
    } catch (error) {
      set({ error: "AI chat failed. Check the server configuration.", isLoading: false });
      throw error;
    }
  },
  generateDiagram: async (prompt, imageDataUrl) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.post("/ai/generate-diagram", { prompt, imageDataUrl });
      set({ result: response.data, isLoading: false });
      return response.data.content as SceneContent;
    } catch (error) {
      set({ error: "Diagram generation failed. Check the server configuration.", isLoading: false });
      throw error;
    }
  },
  toCode: async (prompt, scene) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.post("/ai/to-code", { prompt, scene });
      set({ result: response.data, isLoading: false });
    } catch (error) {
      set({ error: "Code generation failed. Check the server configuration.", isLoading: false });
      throw error;
    }
  },
  reset: () => set({ isLoading: false, error: null, result: null }),
}));

export default useAIStore;
