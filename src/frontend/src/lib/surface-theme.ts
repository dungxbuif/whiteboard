import { createContext, useContext } from "react";

export type SurfaceTheme = "light" | "dark";
export const SurfaceThemeContext = createContext<SurfaceTheme | null>(null);
export function useSurfaceTheme() {
  const theme = useContext(SurfaceThemeContext);
  return theme ? { className: "dashboard-theme", "data-theme": theme } : {};
}
