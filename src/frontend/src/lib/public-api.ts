import axios from "axios";

import type { SceneContent } from "@/store/scene-store";

// Public challenges must not attach login credentials or run the login 401 handler.
export const publicApi = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "/api/v1",
  headers: { "Content-Type": "application/json" },
});

export type PublicScene = { title: string; content: SceneContent };

export async function readPublicScene(token: string, password = "", displayName = ""): Promise<PublicScene> {
  const response = await publicApi.post<PublicScene>(
    `/public/scenes/${encodeURIComponent(token)}`,
    { ...(password ? { password } : {}), ...(displayName ? { displayName } : {}) },
  );
  return response.data;
}
