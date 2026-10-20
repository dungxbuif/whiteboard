import { createContext, useContext, useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from 'react';
import type { NativeElement, SceneContent, SceneRef } from '@collab/contracts';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { CollabTransport, type RoomGrant } from './transport';
import { offlineStore } from './offline-store';
import { createLocalUndo } from './undo';
import { Presence } from './presence';
import { Voice } from './voice';
import { ScreenShare } from './screen-share';
import useAuthStore from '@/store/auth-store';
import { registerContentWriter, registerLegacyConflict } from './writers';
import { bindFeatureInvalidation, FeatureInvalidationCoordinator, type FeatureRefetch } from './feature-invalidation';
import api from '@/lib/api';
import { publicApi } from '@/lib/public-api';
import { renewPublicRoom, type SceneSession } from '@/lib/collaboration-api';
import useSceneStore, { type Scene } from '@/store/scene-store';
export type CollabPrincipal = { kind: 'user'; id: string } | { kind: 'public'; id: string; session: SceneSession; onSessionChange: (session: SceneSession) => void };
export function collabURL() {
  const configured = import.meta.env.VITE_COLLAB_URL as string | undefined;
  if (configured) return configured;
  const url = new URL('/collab', window.location.href); url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'; return url.href;
}
function useController(sceneRef: SceneRef, principal: CollabPrincipal) {
  const principalRef = useRef(principal); principalRef.current = principal;
  const [legacy, setLegacy] = useState<Scene | null>(null);
  const recovery = useRef<unknown>(null);
  const controller = useMemo(() => {
    const adapter = new ExcalidrawAdapter(); const undo = createLocalUndo(adapter);
    const grant = async (updateSession = true): Promise<RoomGrant> => {
        const current = principalRef.current;
        if (current.kind === 'public') {
          const result = await renewPublicRoom(current.session);
          if (updateSession && result.sceneSession) { principalRef.current = { ...current, session: result.sceneSession }; current.onSessionChange(result.sceneSession); }
          return result;
        }
        return (await api.post<RoomGrant>(`/scenes/${sceneRef.sceneId}/room-token`, {}, { headers: { 'X-Workspace-ID': sceneRef.workspaceId } })).data;
    };
    const transport = new CollabTransport({ sceneRef, principalId: principal.id, adapter, store: offlineStore, url: collabURL(), grant,
      onAccessLost: () => { const current = principalRef.current; if (current.kind === 'public') { const session = { ...current.session, token: '' }; principalRef.current = { ...current, session }; current.onSessionChange(session); } },
      onLegacy: async () => {
        if (principalRef.current.kind !== 'user') return false;
        const fresh = (await api.get<Scene>(`/scenes/${sceneRef.sceneId}/`, { headers: { 'X-Workspace-ID': sceneRef.workspaceId } })).data;
        if (fresh.writerMode !== 'v1') return false;
        setLegacy(fresh); return true;
      },
    });
    const features = new FeatureInvalidationCoordinator();
    const presence = new Presence(transport, () => principalRef.current.kind === 'public' ? principalRef.current.session.displayName : useAuthStore.getState().user?.name ?? 'Participant');
    const mediaConfig = async () => {
      const room = await grant(false);
      return (await publicApi.get<RTCConfiguration & { expiresAt: string }>('/realtime/config', { headers: { Authorization: `Bearer ${room.token}`, 'X-Workspace-ID': sceneRef.workspaceId } })).data;
    };
    const voice = new Voice(transport, presence, { config: mediaConfig });
    const screenShare = new ScreenShare(transport, presence, { config: mediaConfig });
    return { adapter, transport, undo, presence, voice, screenShare, features };
  }, [sceneRef.workspaceId, sceneRef.sceneId, principal.id]);
  const state = useSyncExternalStore(controller.transport.subscribe, controller.transport.getSnapshot);
  const [content, setContent] = useState<SceneContent>(() => controller.adapter.content());
  useEffect(() => {
    const { adapter, transport } = controller;
    const update = () => {
      const content = adapter.content(); setContent(content);
      if (transport.scope && principalRef.current.kind === 'user') useSceneStore.setState(state => ({ activeScene: state.activeScene?.id === sceneRef.sceneId ? { ...state.activeScene, content, writerMode: 'v2' } : state.activeScene }));
    };
    adapter.doc.on('update', update); controller.presence.start(); controller.voice.start(); controller.screenShare.start(); void transport.start();
    const unbindFeatures = bindFeatureInvalidation(controller.features, transport, {
      browser: window,
      document,
      isVisible: () => document.visibilityState === 'visible',
      route: () => `${window.location.pathname}${window.location.search}`,
      onError: error => console.error('Authorized feature refetch failed', error),
    });
    const removeConflict = registerLegacyConflict(sceneRef.workspaceId, sceneRef.sceneId, draft => { recovery.current = draft; setLegacy(null); transport.retry(); });
    return () => { unbindFeatures(); adapter.doc.off('update', update); controller.screenShare.stop(); controller.voice.stop(); controller.presence.stop(); transport.stop(); removeConflict(); };
  }, [controller, sceneRef.workspaceId, sceneRef.sceneId]);
  useEffect(() => {
    if (state.status === 'legacy') return;
    return registerContentWriter(sceneRef.workspaceId, sceneRef.sceneId, input => {
      if (!controller.transport.state.canEdit) throw new Error('Scene is read-only until collaboration admission');
      controller.undo.apply(input as SceneContent);
    });
  }, [controller, sceneRef.workspaceId, sceneRef.sceneId, state.status === 'legacy']);
  return { ...controller, state, content, legacy, principal: principalRef, sceneRef,
    applyLocalScene(content: SceneContent) { if (!controller.transport.state.canEdit) throw new Error('Scene is read-only'); controller.undo.apply(content); },
    async applyAgentElements(elements: NativeElement[]) { if (!controller.transport.state.canEdit) throw new Error('Scene is read-only'); await controller.adapter.applyAgentElements(elements); await controller.transport.whenDurable(); },
    async exportRecovery() { return { ...await controller.transport.exportRecovery(), legacyDraft: recovery.current }; },
    registerFeatureRefetch(key: string, refetch: FeatureRefetch) { return controller.features.register(key, refetch); },
    refreshFeatures() { return controller.features.manualRefresh(); },
    async post<T>(path: string, body: unknown): Promise<T> { const current = principalRef.current; return (await (current.kind === 'public' ? publicApi : api).post<T>(path, body, { headers: { 'X-Workspace-ID': sceneRef.workspaceId, ...(current.kind === 'public' ? { Authorization: `Bearer ${current.session.token}` } : {}) } })).data; },
  };
}
type Controller = ReturnType<typeof useController>;
const Context = createContext<Controller | null>(null);
export function CollabProvider({ sceneRef, principal, children }: { sceneRef: SceneRef; principal: CollabPrincipal; children: ReactNode }) {
  const value = useController(sceneRef, principal); return <Context.Provider value={value}>{children}</Context.Provider>;
}
export function useCollab() { const value = useContext(Context); if (!value) throw new Error('CollabProvider is required'); return value; }
export function useAuthorizedFeatureRefetch(key: string, refetch: FeatureRefetch) {
  const { features } = useCollab();
  useEffect(() => {
    const registration = features.register(key, refetch);
    void registration.ready.catch(error => console.error('Authorized feature refetch failed', error));
    return registration.unregister;
  }, [features, key, refetch]);
}
