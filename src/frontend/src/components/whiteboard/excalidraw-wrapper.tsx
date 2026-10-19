import { useEffect, useRef, useState, type ComponentType } from 'react';
import '@excalidraw/excalidraw/index.css';
import type { AssetReference, NativeElement, SceneContent } from '@collab/contracts';
import type { ExcalidrawImperativeAPI } from '@excalidraw/excalidraw/types';
import { CollabProvider, useCollab } from '@/collab/provider';
import { bindNativePresence } from '@/collab/native-presence';
import { CollaborationToolbar } from './collaboration-toolbar';
import { offlineStore, scopeKey, type PendingBlob, type ImageDraft } from '@/collab/offline-store';
import { uploadBlob, type UploadGrant } from '@/collab/assets';
import { advanceLocalBaseline, mergeSceneChanges } from '@/collab/scene-diff';
import { Button } from '@/components/ui/button';
import { navigate } from '@/lib/router';
import { normalizeSceneContent } from '@/lib/scene-content';
import useSceneStore, { syncPresentationOrder } from '@/store/scene-store';
import useAuthStore from '@/store/auth-store';
import usePresentationStore from '@/store/presentation-store';
import { PresentationControls } from './presentation-controls';
import { SlideNavigator } from './slide-navigator';
import { AICommandPanel } from './ai-command-panel';

type ExcalidrawModule = typeof import('@excalidraw/excalidraw');
type CanvasFile = { id: string; dataURL?: string; mimeType: string; created: number; assetId?: string };
const appKeys = ['viewBackgroundColor', 'gridSize', 'gridStep', 'gridModeEnabled', 'theme'] as const;
const json = <T,>(value: T): T => JSON.parse(JSON.stringify(value));
const signature = (content: SceneContent) => JSON.stringify(content);
function canonical(elements: readonly unknown[], appState: Record<string, unknown>, files: Record<string, AssetReference>, presentationOrder: string[]): SceneContent {
  return json({ elements: [...elements] as NativeElement[], appState: Object.fromEntries(appKeys.filter(key => appState[key] !== undefined).map(key => [key, appState[key]])), files, presentationOrder });
}
function download(value: unknown) {
  const url = URL.createObjectURL(new Blob([JSON.stringify(value)], { type: 'application/json' }));
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = 'whiteboard-recovery.json'; anchor.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export function ExcalidrawWrapper() {
  const scene = useSceneStore(state => state.activeScene), user = useAuthStore(state => state.user);
  if (!scene || !user) return null;
  return <CollabProvider sceneRef={{ workspaceId: scene.workspaceId, sceneId: scene.id }} principal={{ kind: 'user', id: user.id }}><CollaborativeCanvas title={scene.title} onBack={() => { void useSceneStore.getState().closeScene().then(() => navigate('/')); }} /></CollabProvider>;
}
export function CollaborativeCanvas({ title, onBack, displayName, onRejoin, initialContent }: { initialContent?: SceneContent; title: string; onBack?: () => void; displayName?: string; onRejoin?: () => void }) {
  const saveStatus = useSceneStore(state => state.saveStatus);
  const activeScene = useSceneStore(state => state.activeScene);
  const collab = useCollab(); const { adapter, transport, state, content, legacy, sceneRef, undo } = collab;
  const [module, setModule] = useState<ExcalidrawModule | null>(null);
  const [api, setAPI] = useState<ExcalidrawImperativeAPI | null>(null);
  const [uploadError, setUploadError] = useState(''); const [uploads, setUploads] = useState(0);
  const files = useRef<Record<string, CanvasFile>>({});
  const refs = useRef<Record<string, AssetReference>>({});
  const pending = useRef<ImageDraft | null>(null);
  const preparing = useRef(new WeakMap<ImageDraft, Promise<void>>());
  const appliedImage = useRef<{ scope: string; owner: number; transport: typeof transport; base: SceneContent; durable: Promise<void> } | null>(null);
  const lastNative = useRef<SceneContent | null>(null);
  const localProjection = useRef<string | null>(null);
  const lifetime = useRef(0);
  const rendering = useRef(false), uploading = useRef(false), mounted = useRef(true);
  const { isPresenting, activeSlideIndex, enter, exit, setActiveSlide } = usePresentationStore();
  const canEdit = state.canEdit || (state.status === 'legacy' && legacy?.access === 'edit');
  const renderedContent = legacy && state.status === 'legacy' ? (activeScene?.id === sceneRef.sceneId ? activeScene.content : legacy.content) as SceneContent : !transport.scope && initialContent ? initialContent : content;
  const slideIds = syncPresentationOrder(renderedContent.elements, renderedContent.presentationOrder ?? []);
  const legacyScope = { ...sceneRef, principalId: collab.principal.current.id, documentId: 'legacy-v1' };
  const assetScope = state.status === 'legacy' ? legacyScope : transport.scope;
  const currentRef = useRef(collab); currentRef.current = collab;
  useEffect(() => { const owner = ++lifetime.current; mounted.current = true; void import('@excalidraw/excalidraw').then(value => { if (mounted.current && lifetime.current === owner) setModule(value); }); return () => { mounted.current = false; lifetime.current++; }; }, [transport]);
  useEffect(() => {
    if (!api || !module) return;
    if (localProjection.current === signature(renderedContent)) return;
    const display = pending.current ? mergeSceneChanges(pending.current.base, pending.current.desired, renderedContent) : renderedContent;
    refs.current = { ...refs.current, ...renderedContent.files };
    const nativeElements = module.restoreElements(display.elements as never, null);
    lastNative.current = canonical(nativeElements, { ...api.getAppState(), ...display.appState }, refs.current, display.presentationOrder ?? []);
    rendering.current = true;
    api.updateScene({ elements: nativeElements as never, appState: display.appState as never, captureUpdate: module.CaptureUpdateAction.NEVER });
    api.history.clear();
    requestAnimationFrame(() => { rendering.current = false; });
  }, [api, module, renderedContent]);
  useEffect(() => {
    if (!api || !assetScope) return;
    let current = true;
    const missing = Object.keys(renderedContent.files).filter(id => !files.current[id]?.dataURL);
    if (missing.length) {
      const guest = collab.principal.current.kind === 'public';
      void collab.post<{ urls: Record<string, { viewUrl: string; mimeType: string }> }>(guest ? `/public/scenes/${sceneRef.sceneId}/assets/presign-view` : '/assets/presign-view', guest ? { fileIds: missing } : { sceneId: sceneRef.sceneId, fileIds: missing }).then(result => {
        if (!current) return;
        for (const id of missing) if (result.urls[id]) files.current[id] = { ...renderedContent.files[id], dataURL: result.urls[id].viewUrl };
        api.addFiles(Object.values(files.current).filter(file => file.dataURL) as never);
      }).catch(() => { if (current) setUploadError('Some images could not load. Retry when connected.'); });
    }
    return () => { current = false; };
  }, [api, renderedContent.files, state.status]);
  useEffect(() => {
    if (api && module && collab.presence) return bindNativePresence(collab.presence, api, module.CaptureUpdateAction.NEVER);
  }, [api, module, collab.presence]);
  async function retryUploads() {
    const current = currentRef.current, owner = lifetime.current;
    const isLegacy = current.transport.state.status === 'legacy' && current.legacy?.access === 'edit';
    const scope = isLegacy ? legacyScope : current.transport.scope;
    const allowed = () => {
      const latest = currentRef.current, activeScene = useSceneStore.getState().activeScene;
      if (!scope || !mounted.current || owner !== lifetime.current || latest.transport !== current.transport || !current.transport.isActive() || latest.principal.current.id !== scope.principalId || latest.sceneRef.workspaceId !== scope.workspaceId || latest.sceneRef.sceneId !== scope.sceneId) return false;
      return isLegacy
        ? current.transport.state.status === 'legacy' && latest.legacy?.access === 'edit' && activeScene?.writerMode === 'v1' && activeScene.id === scope.sceneId && activeScene.workspaceId === scope.workspaceId
        : current.transport.state.canEdit && !!current.transport.scope && scopeKey(current.transport.scope) === scopeKey(scope);
    };
    if (!scope || uploading.current || !allowed()) return;
    uploading.current = true; let repeat = false;
    try {
      const draft = pending.current;
      if (draft) await preparing.current.get(draft);
      if (!allowed()) return;
      if (pending.current !== draft) { repeat = true; return; }
      if (draft?.pendingBaseline) throw new Error('Image preparation was interrupted. Export recovery before reopening or clearing local data; automatic image replay is paused.');
      const rows = await offlineStore.blobs(scope); if (!allowed()) return; setUploads(rows.length);
      // Upgrade retained pre-revision recovery before any upload/cleanup.
      if (pending.current && pending.current.revision === undefined) { pending.current.revision = await offlineStore.reserveImageDraft(scope, pending.current); if (!allowed()) return; }
      const conflict = () => new Error('A newer image draft exists in this scene. Recovery retained; export recovery and reopen the scene.');
      const currentDraft = async () => {
        if (!allowed()) return false;
        if (pending.current !== draft) { repeat = true; return false; }
        const matches = !draft || (draft.revision !== undefined && await offlineStore.isImageDraftCurrent(scope, draft.revision));
        if (!allowed()) return false;
        if (pending.current !== draft) { repeat = true; return false; }
        if (!matches) throw conflict();
        return true;
      };
      if (!await currentDraft()) return;
      for (const item of rows) {
        if (!allowed()) return;
        const guest = current.principal.current.kind === 'public';
        const reference = await uploadBlob(item, assets => current.post<{ urls: Record<string, UploadGrant> }>(guest ? `/public/scenes/${sceneRef.sceneId}/assets/presign-upload` : '/assets/presign-upload', guest ? { assets } : { sceneId: sceneRef.sceneId, assets }), fetch, allowed);
        if (!await currentDraft()) return;
        refs.current[item.id] = reference;
      }
      if (!await currentDraft()) return;
      if (draft) {
        const desired = { ...draft.desired, files: { ...draft.desired.files, ...refs.current } };
        if (isLegacy) {
          await offlineStore.saveDraft(scope, desired);
          if (!await currentDraft()) return;
          useSceneStore.getState().scheduleSave(sceneRef.workspaceId, sceneRef.sceneId, desired);
          await useSceneStore.getState().flushSave();
          if (!allowed()) return;
          if (useSceneStore.getState().saveStatus !== 'saved') throw new Error('Legacy save is not confirmed; recovery retained.');
        } else {
          const before = current.adapter.content();
          current.applyLocalScene(mergeSceneChanges(draft.base, desired, before));
          const durable = current.transport.whenDurable(); void durable.catch(() => undefined);
          appliedImage.current = { scope: scopeKey(scope), owner, transport: current.transport, base: advanceLocalBaseline(draft.base, desired, before), durable };
          await current.transport.flush();
        }
        if (!allowed()) return;
        if (pending.current !== draft) { repeat = true; return; }
        const completed = draft.revision !== undefined && await offlineStore.completeImageDraft(scope, draft.revision, rows.map(item => item.id));
        if (!allowed()) return;
        if (pending.current !== draft) { repeat = true; return; }
        if (!completed) throw conflict();
        pending.current = null; appliedImage.current = null;
      }
      setUploads(0); setUploadError('');
    } catch (error) { if (allowed()) setUploadError(error instanceof Error ? error.message : 'Upload failed. Bytes retained.'); }
    finally { uploading.current = false; if (repeat && allowed()) void retryUploads(); }
  }
  useEffect(() => {
    if (!assetScope || !canEdit) return;
    let current = true;
    void offlineStore.blobs(assetScope).then(async rows => {
      if (!current || !rows.length) return;
      if (!pending.current) pending.current = rows.find(row => row.draft)?.draft ?? null;
      for (const row of rows) if (!files.current[row.id]?.dataURL) {
        const dataURL = await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result)); reader.onerror = reject; reader.readAsDataURL(row.blob); });
        if (!current) return;
        files.current[row.id] = { id: row.id, mimeType: row.mimeType, created: row.created, dataURL };
      }
      if (api) api.addFiles(Object.values(files.current).filter(file => file.dataURL) as never);
      if (current) void retryUploads();
    }).catch(() => setUploadError('Local image recovery could not be read. Export recovery before clearing browser data.'));
    return () => { current = false; };
  }, [canEdit, api, state.status, transport]);
  useEffect(() => {
    const keydown = (event: KeyboardEvent) => {
      if (event.target instanceof HTMLElement && (event.target.isContentEditable || ['INPUT', 'TEXTAREA'].includes(event.target.tagName))) return;
      if ((event.ctrlKey || event.metaKey) && ['z', 'y'].includes(event.key.toLowerCase()) && state.status !== 'legacy') {
        event.preventDefault(); event.stopImmediatePropagation();
        if (state.canEdit) { if (event.shiftKey || event.key.toLowerCase() === 'y') undo.redo(); else undo.undo(); }
      }
      if (isPresenting && ['Escape', 'ArrowRight', 'ArrowDown', ' ', 'ArrowLeft', 'ArrowUp'].includes(event.key)) {
        event.preventDefault(); if (event.key === 'Escape') exit(); else setActiveSlide(activeSlideIndex + (['ArrowLeft', 'ArrowUp'].includes(event.key) ? -1 : 1), slideIds.length);
      }
    };
    window.addEventListener('keydown', keydown, true); return () => window.removeEventListener('keydown', keydown, true);
  }, [state.canEdit, state.status, undo, isPresenting, activeSlideIndex, slideIds.length, exit, setActiveSlide]);
  useEffect(() => {
    setActiveSlide(activeSlideIndex, slideIds.length);
    if (isPresenting && api) { const frame = renderedContent.elements.find(e => e.id === slideIds[activeSlideIndex]); if (frame) api.scrollToContent(frame as never, { fitToViewport: true, viewportZoomFactor: 0.8, animate: true, duration: 300 }); }
  }, [isPresenting, activeSlideIndex, JSON.stringify(slideIds), api]);
  async function onChange(elements: readonly unknown[], appState: Record<string, unknown>, incomingFiles: Record<string, CanvasFile>) {
    const owner = lifetime.current;
    const stillCurrent = () => mounted.current && lifetime.current === owner && currentRef.current.transport === transport && transport.isActive();
    if (!stillCurrent()) return;
    files.current = { ...files.current, ...incomingFiles };
    const desired = canonical(elements, appState, refs.current, syncPresentationOrder([...elements], renderedContent.presentationOrder ?? []));
    const firstCallback = lastNative.current === null;
    // A recovered image draft is the native baseline on a same-scope reopen,
    // even before the imperative canvas API has reported its first snapshot.
    // Comparing the first gesture with bare durable content would treat the
    // recovered image ID as an unrelated colliding add and discard the move.
    const recoveredBaseline = pending.current
      ? mergeSceneChanges(pending.current.base, pending.current.desired, renderedContent)
      : renderedContent;
    const before = lastNative.current ?? recoveredBaseline; lastNative.current = desired;
    // Native mount supplies default appState. Establish its baseline without persisting defaults.
    if (firstCallback && JSON.stringify(desired.elements) === JSON.stringify(renderedContent.elements)) return;
    if (rendering.current || !canEdit || signature(before) === signature(desired)) return;

    api?.history.clear();
    const missing = Object.values(files.current).filter(file => !refs.current[file.id] && file.dataURL?.startsWith('data:') && desired.elements.some(e => e.type === 'image' && e.fileId === file.id));
    try {
      if (missing.length || pending.current) {
        const prior = appliedImage.current;
        const applied = pending.current && prior && assetScope && prior.scope === scopeKey(assetScope) && prior.owner === owner && prior.transport === transport ? prior : null;
        // A native callback includes fields that were merely rendered from a
        // remote update. Accumulate only this callback's before→desired delta
        // into the pending local intent so a later retry cannot replay those
        // observed remote fields over a newer collaborator change.
        const localIntent = pending.current ? mergeSceneChanges(before, desired, pending.current.desired) : desired;
        const draft: ImageDraft = { base: pending.current?.base ?? renderedContent, desired: localIntent, ...((applied || pending.current?.pendingBaseline) ? { pendingBaseline: true } : {}) };
        pending.current = draft;
        if (!assetScope) return;
        const preparation = (async () => {
          draft.revision = await offlineStore.reserveImageDraft(assetScope, draft);
          for (const file of missing) {
            const blob = await (await fetch(file.dataURL!)).blob();
            const item: PendingBlob = { scope: assetScope, id: file.id, mimeType: file.mimeType, created: file.created, blob, draft };
            await offlineStore.putImageBlob(item, draft.revision);
          }
          if (applied) {
            await applied.durable;
            if (!stillCurrent()) return;
            if (!await offlineStore.rebaseImageDraft(assetScope, draft.revision, applied.base)) throw new Error('A newer image draft exists. Recovery retained; export recovery and reopen the scene.');
            draft.base = applied.base; delete draft.pendingBaseline;
          }
        })();
        preparing.current.set(draft, preparation);
        await preparation;
        if (stillCurrent()) { setUploads(missing.length); void retryUploads(); }
      } else if (state.status === 'legacy') {
        await offlineStore.saveDraft(legacyScope, desired);
        if (!stillCurrent()) return;
        useSceneStore.getState().scheduleSave(sceneRef.workspaceId, sceneRef.sceneId, desired);
        localProjection.current = signature(desired);
      } else { collab.applyLocalScene(mergeSceneChanges(before, desired, adapter.content())); localProjection.current = signature(adapter.content()); }
    } catch (error) { setUploadError(error instanceof Error ? error.message : 'Changes retained in canvas; export recovery.'); }
  }
  const labels = { connecting: 'Connecting · Read-only until admitted', synced: 'Saved', pending: `${state.pending} pending updates`, offline: 'Offline · Changes stay on this device', readonly: 'Read-only', revoked: 'Access revoked · Recovery retained', degraded: 'Recovery required', legacy: ({ idle: 'Saved', saved: 'Saved', pending: 'Waiting to save', saving: 'Saving', error: 'Save failed' } as const)[saveStatus] };
  const Excalidraw = module?.Excalidraw as ComponentType<Record<string, unknown>> | undefined;
  return <main className="grid h-screen min-w-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden bg-background">
    <header className="flex min-h-14 min-w-0 flex-wrap items-center justify-between gap-3 border-b bg-card px-4 py-2">
      <div className="flex min-w-0 items-center gap-3">{onBack && <Button aria-label="Back to dashboard" variant="ghost" onClick={onBack}>Back</Button>}<h1 className="truncate font-semibold">{title}</h1>{displayName && <span>{displayName}</span>}</div>
      <div className="flex flex-wrap items-center gap-2">{state.status !== 'legacy' && <CollaborationToolbar />}<Button onClick={() => isPresenting ? exit() : enter()}>{isPresenting ? 'Exit presentation' : 'Present'}</Button>{state.status !== 'legacy' && <><Button disabled={!state.canEdit || !undo.canUndo()} onClick={() => undo.undo()}>Undo my edit</Button><Button disabled={!state.canEdit || !undo.canRedo()} onClick={() => undo.redo()}>Redo my edit</Button></>}{state.status === 'revoked' && onRejoin && <Button onClick={() => void collab.exportRecovery().then(value => { download({ ...value, pendingCanvas: pending.current }); onRejoin(); })}>Export recovery and rejoin</Button>}<span role="status">{uploads ? `${uploads} image uploads pending` : collab.principal.current.kind === 'public' && collab.principal.current.session.access === 'view' && state.status !== 'revoked' ? state.status === 'degraded' ? 'Read-only · Recovery required' : 'Read-only' : labels[state.status]}</span>{(state.recovery || uploadError || ['offline', 'revoked', 'degraded'].includes(state.status)) && <Button onClick={() => void collab.exportRecovery().then(value => download({ ...value, pendingCanvas: pending.current }))}>Export recovery</Button>}{(uploadError || ['offline', 'degraded', 'revoked'].includes(state.status)) && <Button onClick={() => { setUploadError(''); transport.retry(); void retryUploads(); }}>Retry connection / uploads</Button>}</div>
      {(uploadError || state.error) && <p role="alert" className="w-full text-sm text-destructive">{uploadError || state.error}</p>}
    </header>
    <section className="relative min-h-0" aria-label="Excalidraw canvas">
      {Excalidraw ? <Excalidraw excalidrawAPI={setAPI} initialData={normalizeSceneContent({ ...renderedContent, files: Object.fromEntries(Object.entries(renderedContent.files).filter(([, file]) => 'dataURL' in file && typeof file.dataURL === 'string' && file.dataURL.length > 0)) })} name={title} isCollaborating={!!state.connectionId} viewModeEnabled={!canEdit || isPresenting} onChange={onChange} onPointerUpdate={(value: { pointer: { x: number; y: number } }) => transport.sendAwareness({ cursor: value.pointer })} /> : <p>Loading whiteboard...</p>}
      {isPresenting && <><SlideNavigator activeIndex={activeSlideIndex} onMove={(index, direction) => { if (!canEdit) return; const next = [...slideIds], swap = index + direction; [next[index], next[swap]] = [next[swap], next[index]]; if (state.status === 'legacy') useSceneStore.getState().updatePresentationOrder(sceneRef.workspaceId, sceneRef.sceneId, next); else adapter.reorderPresentation(next); setActiveSlide(swap, next.length); }} onSelect={index => setActiveSlide(index, slideIds.length)} slideIds={slideIds} /><PresentationControls activeIndex={activeSlideIndex} onExit={exit} onNext={() => setActiveSlide(activeSlideIndex + 1, slideIds.length)} onPrevious={() => setActiveSlide(activeSlideIndex - 1, slideIds.length)} slideCount={slideIds.length} /></>}
      {!isPresenting && canEdit && <AICommandPanel scene={renderedContent} onApplyDiagram={generated => { if (state.status === 'legacy') { const desired = { ...renderedContent, elements: [...renderedContent.elements, ...generated.elements] } as SceneContent; void offlineStore.saveDraft(legacyScope, desired).then(() => { if (mounted.current && currentRef.current.transport === transport && transport.isActive() && transport.state.status === 'legacy') useSceneStore.getState().scheduleSave(sceneRef.workspaceId, sceneRef.sceneId, desired); }).catch(error => setUploadError(String(error))); } else void collab.applyAgentElements(generated.elements as NativeElement[]).catch(error => setUploadError(String(error))); }} />}
    </section>
  </main>;
}
