import type { SceneContent } from "@/store/scene-store";

export function normalizeSceneContent(content: SceneContent): SceneContent {
  const collaborators = content.appState?.collaborators;
  if (collaborators instanceof Map) return content;

  return {
    ...content,
    appState: { ...content.appState, collaborators: new Map() } as SceneContent["appState"],
  };
}
