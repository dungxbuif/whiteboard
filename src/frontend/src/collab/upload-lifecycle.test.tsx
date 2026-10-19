import 'fake-indexeddb/auto';
import { afterEach, expect, it, vi } from 'vitest';
import { act, render, waitFor } from '@testing-library/react';
import * as Y from 'yjs';
import type { SceneContent } from '@collab/contracts';
import { CollabTransport, type WireSocket } from './transport';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { offlineStore } from './offline-store';
import { rectangle, scene } from './fixtures';
import { uploadBlob } from './assets';
import { createLocalUndo } from './undo';
import useSceneStore from '@/store/scene-store';
import api from '@/lib/api';
import type { useCollab } from './provider';

const mock = vi.hoisted(() => ({ controller: undefined as unknown }));
vi.mock('./provider', () => ({ useCollab: () => mock.controller, CollabProvider: () => null }));
vi.mock('./assets', () => ({ uploadBlob: vi.fn() }));
vi.mock('@excalidraw/excalidraw', () => ({ Excalidraw: () => null, restoreElements: (elements: unknown) => elements, CaptureUpdateAction: { NEVER: 'never' } }));
vi.mock('@/components/whiteboard/ai-command-panel', () => ({ AICommandPanel: () => null }));
import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
class Socket extends EventTarget {
  readyState = 1; binaryType: BinaryType = 'arraybuffer';
  send() {} close() { this.readyState = 3; }
  receive(value: unknown) { this.dispatchEvent(new MessageEvent('message', { data: JSON.stringify(value) })); }
}
const tick = () => new Promise(resolve => setTimeout(resolve, 20));
const transports: CollabTransport[] = [];
afterEach(() => { for (const transport of transports.splice(0)) transport.stop(); useSceneStore.getState().reset(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

it.each(['v2', 'legacy'])('collab %s pending PUT completion after unmount retains draft/blobs for a fresh controller', async mode => {
  // FileReader presentation is irrelevant to this lifecycle proof; PUT itself is held below.
  vi.stubGlobal('FileReader', class { result = 'data:image/png;base64,Ynl0ZXM='; onload?: () => void; readAsDataURL() { queueMicrotask(() => this.onload?.()); } });
  const ref = { workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID() }, principalId = 'alice';
  const scope = { ...ref, principalId, documentId: mode === 'legacy' ? 'legacy-v1' : 'epoch' };
  const base = scene();
  const desired = { ...base, elements: [...base.elements, { ...rectangle('image'), type: 'image', fileId: 'file', status: 'saved', scale: [1, 1], crop: null }] } as SceneContent;
  await offlineStore.putBlob({ scope, id: 'file', blob: new Blob(['bytes'], { type: 'image/png' }), mimeType: 'image/png', created: 1, draft: { base, desired } });
  if (mode === 'legacy') await offlineStore.saveDraft(scope, desired);
  const activeScene = { id: ref.sceneId, workspaceId: ref.workspaceId, content: base, writerMode: mode === 'legacy' ? 'v1' : 'v2', access: 'edit' };
  useSceneStore.setState({ activeScene: activeScene as never });
  vi.spyOn(api, 'patch').mockImplementation(async (_path, body) => ({ data: { ...activeScene, content: (body as { content: SceneContent }).content } }));
  async function controller() {
    const adapter = new ExcalidrawAdapter(), socket = new Socket();
    const transport = new CollabTransport({ sceneRef: ref, principalId, adapter, store: offlineStore, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://test', socket: () => socket as unknown as WireSocket });
    transports.push(transport); await transport.start(); socket.dispatchEvent(new Event('open'));
    const server = new ExcalidrawAdapter(); server.applyLocalScene(base); const bytes = Y.encodeStateAsUpdate(server.doc);
    socket.receive({ type: 'ready', requestId: 'auth', payload: { ...ref, principalId, documentId: 'epoch', access: 'edit' } });
    socket.receive({ type: 'sync_chunk', requestId: 'auth', payload: { documentId: 'epoch', index: 0, count: 1, totalBytes: bytes.length, data: btoa(String.fromCharCode(...bytes)) } });
    await tick();
    if (mode === 'legacy') transport.state = { ...transport.state, status: 'legacy', canEdit: false };
    mock.controller = { adapter, transport, state: transport.state, content: adapter.content(), legacy: mode === 'legacy' ? activeScene : null, sceneRef: ref, undo: createLocalUndo(adapter.doc), principal: { current: { kind: 'user', id: principalId } }, applyLocalScene: (content: SceneContent) => adapter.applyLocalScene(content), post: vi.fn() } as unknown as ReturnType<typeof useCollab>;
    return { adapter, transport };
  }
  const first = await controller();
  let finish!: () => void;
  vi.mocked(uploadBlob).mockImplementationOnce(() => new Promise(resolve => { finish = () => resolve({ id: 'file', assetId: 'stored', mimeType: 'image/png', created: 1 }); }));
  const canvas = render(<CollaborativeCanvas title="test" />);
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  canvas.unmount(); first.transport.stop();
  await act(async () => { finish(); await tick(); });
  expect(await offlineStore.blobs(scope)).toHaveLength(1);
  expect((await offlineStore.blobs(scope))[0].draft?.desired.elements.some(element => element.id === 'image')).toBe(true);
  expect(first.adapter.content().elements.some(element => element.id === 'image')).toBe(false);
  expect(await offlineStore.pending(scope)).toHaveLength(0);
  if (mode === 'legacy') expect(await offlineStore.draft(scope)).toEqual(desired);
  vi.mocked(uploadBlob).mockResolvedValue({ id: 'file', assetId: 'stored', mimeType: 'image/png', created: 1 });
  const reopened = await controller(); const nextCanvas = render(<CollaborativeCanvas title="reopened" />);
  await waitFor(async () => expect(await offlineStore.blobs(scope)).toHaveLength(0));
  if (mode === 'v2') {
    expect(reopened.adapter.content().elements.some(element => element.id === 'image')).toBe(true);
    expect((await offlineStore.pending(scope)).length).toBeGreaterThan(0);
  } else { expect(useSceneStore.getState().saveStatus).toBe('saved'); expect(await offlineStore.draft(scope)).toBeUndefined(); }
  nextCanvas.unmount();
});
