import type { SceneContent } from '@/store/scene-store';
const writers = new Map<string, (content: SceneContent) => void>();
const conflicts = new Map<string, (content: SceneContent) => void>();
const key = (workspaceId: string, sceneId: string) => JSON.stringify([workspaceId, sceneId]);
export function registerContentWriter(workspaceId: string, sceneId: string, writer: (content: SceneContent) => void) {
  const id = key(workspaceId, sceneId); writers.set(id, writer);
  return () => { if (writers.get(id) === writer) writers.delete(id); };
}
export function writeCollaborativeContent(workspaceId: string, sceneId: string, content: SceneContent) {
  const writer = writers.get(key(workspaceId, sceneId)); if (!writer) return false; writer(content); return true;
}
export function registerLegacyConflict(workspaceId: string, sceneId: string, callback: (content: SceneContent) => void) {
  const id = key(workspaceId, sceneId); conflicts.set(id, callback); return () => { conflicts.delete(id); };
}
export function legacyConflict(workspaceId: string, sceneId: string, content: SceneContent) { conflicts.get(key(workspaceId, sceneId))?.(content); }
