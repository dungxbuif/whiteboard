import { create } from "zustand";

type DashboardState = {
  activeWorkspace: string;
  setActiveWorkspace: (workspace: string) => void;
};

export const useDashboardStore = create<DashboardState>((set) => ({
  activeWorkspace: "Design Ops",
  setActiveWorkspace: (workspace) => set({ activeWorkspace: workspace }),
}));
