import 'fake-indexeddb/auto';
import * as Y from 'yjs';
import { afterEach, expect, it, vi } from 'vitest';
import { act, render, waitFor, cleanup } from '@testing-library/react';
import type { SceneContent } from '@collab/contracts';
import { offlineStore } from './offline-store';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { CollabTransport } from './transport';
import { createLocalUndo } from './undo';
import { rectangle } from './fixtures';
const mock = vi.hoisted(() => ({ controller: undefined as unknown, change: undefined as unknown }));
vi.mock('./provider', () => ({ useCollab: () => mock.controller, CollabProvider: () => null }));
vi.mock('@excalidraw/excalidraw', () => ({ Excalidraw: (props: { onChange: unknown }) => { mock.change = props.onChange; return null; }, restoreElements: (elements: unknown) => elements, CaptureUpdateAction: { NEVER: 'never' } }));
vi.mock('@/components/whiteboard/ai-command-panel', () => ({ AICommandPanel: () => null }));
import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
afterEach(() => { cleanup(); mock.change = undefined; vi.resetAllMocks(); vi.unstubAllGlobals(); });
function deferred() { let resolve!: () => void, reject!: (error: Error) => void; const promise = new Promise<void>((done, fail) => { resolve = done; reject = fail; }); return { promise, resolve, reject }; }

it.each(['during-put', 'during-flush', 'completed-rebase-reload', 'durability-failure', 'lifetime-loss', 'newer-revision'])('retains movement of a newly inserted image while upload/save is pending: %s', async phase => {
  const { Blob: ByteBlob } = await vi.importActual<{ Blob: typeof Blob }>('node:buffer');
  const scope = { principalId: 'alice', workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID(), documentId: 'epoch' };
  const put = deferred(), flush = deferred(), reachedPut = deferred(), reachedFlush = deferred(), durable = deferred();
  const interrupted = ['durability-failure', 'lifetime-loss', 'newer-revision'].includes(phase);
  vi.stubGlobal('FileReader', class { result = 'data:image/png;base64,Ynl0ZXM='; onload?: () => void; readAsDataURL() { queueMicrotask(() => this.onload?.()); } });
  vi.stubGlobal('fetch', vi.fn(async (url: string) => {
    if (url.startsWith('data:')) return { blob: async () => new ByteBlob(['retained-bytes'], { type: 'image/png' }) };
    expect(url).toBe('https://assets.test/image'); reachedPut.resolve(); await put.promise; return { ok: true };
  }));
  const adapter = new ExcalidrawAdapter();
  const transport = { scope, state: { status: 'synced', canEdit: true, pending: 0, recovery: false }, isActive: () => true, flush: async () => { reachedFlush.resolve(); await flush.promise; }, whenDurable: () => interrupted ? durable.promise : Promise.resolve(), sendAwareness: () => {}, retry: () => {} };
  mock.controller = { adapter, transport, state: transport.state, content: adapter.content(), legacy: null, sceneRef: scope, undo: createLocalUndo(adapter.doc), principal: { current: { kind: 'user', id: scope.principalId } }, applyLocalScene: (content: SceneContent) => adapter.applyLocalScene(content), post: async () => ({ urls: { file: { assetId: 'stored', uploadUrl: 'https://assets.test/image', headers: { 'Content-Type': 'image/png' } } } }) };
  async function change(x: number) {
    await (mock.change as (elements: unknown[], state: object, files: object) => Promise<void>)([{ ...rectangle('image'), type: 'image', fileId: 'file', status: 'saved', scale: [1, 1], crop: null, x }], {}, { file: { id: 'file', mimeType: 'image/png', created: 1, dataURL: 'data:image/png;base64,Ynl0ZXM=' } });
  }
  const canvas = render(<CollaborativeCanvas title="image movement" />);
  await waitFor(() => expect(mock.change).toBeTypeOf('function'));
  await act(async () => { await change(90); await reachedPut.promise; });
  if (phase !== 'during-put') {
    await act(async () => { put.resolve(); await reachedFlush.promise; });
    expect(adapter.content().elements[0]?.x).toBe(90);
    // Remote style arriving after the first projection must survive the later local move.
    const remote = new ExcalidrawAdapter(); remote.applyRemote(Y.encodeStateAsUpdate(adapter.doc));
    remote.applyLocalScene({ ...remote.content(), elements: remote.content().elements.map(e => ({ ...e, strokeColor: '#00aa00' })) });
    adapter.applyRemote(Y.encodeStateAsUpdate(remote.doc));
  }
  if (interrupted) {
    let changePromise!: Promise<void>;
    await act(async () => { changePromise = change(270); });
    await waitFor(async () => expect((await offlineStore.blobs(scope))[0].draft).toMatchObject({ pendingBaseline: true, desired: { elements: [expect.objectContaining({ x: 270 })] } }));
    if (phase === 'newer-revision') {
      const other = (await offlineStore.blobs(scope))[0].draft!;
      await offlineStore.reserveImageDraft(scope, { base: other.base, desired: { ...other.desired, elements: other.desired.elements.map(e => ({ ...e, x: 540 })) } });
    }
    if (phase === 'lifetime-loss') canvas.unmount();
    await act(async () => { if (phase === 'durability-failure') durable.reject(new Error('durability unavailable')); else durable.resolve(); await changePromise; flush.resolve(); });
    const retained = (await offlineStore.blobs(scope))[0];
    expect(retained.draft?.desired.elements[0].x).toBe(phase === 'newer-revision' ? 540 : 270);
    expect(await retained.blob.text()).toBe('retained-bytes');
    expect(adapter.content().elements[0]?.x).toBe(90);
    canvas.unmount();
    if (phase !== 'newer-revision') {
      expect(retained.draft).toMatchObject({ pendingBaseline: true, base: { elements: [] } });
      const reopened = new ExcalidrawAdapter(); reopened.applyRemote(Y.encodeStateAsUpdate(adapter.doc));
      const freshTransport = { ...transport, whenDurable: async () => {}, flush: async () => {} };
      mock.controller = { ...(mock.controller as object), adapter: reopened, transport: freshTransport, content: reopened.content(), applyLocalScene: (content: SceneContent) => reopened.applyLocalScene(content) };
      const next = render(<CollaborativeCanvas title="reopened" />);
      await waitFor(() => expect(next.container.querySelector('[role="alert"]')?.textContent).toMatch(/recovery/i));
      expect(reopened.content().elements[0]?.x).toBe(90);
      expect((await offlineStore.blobs(scope))[0].draft?.desired.elements[0].x).toBe(270);
      const exporter = new CollabTransport({ adapter: reopened, store: offlineStore, sceneRef: scope, principalId: scope.principalId, url: 'ws://unused', grant: async () => ({ token: '', expiresAt: '' }) });
      const recovery = await exporter.exportRecovery();
      expect(recovery.scopes.find(entry => entry.scope.documentId === 'epoch')?.blobs[0].draft).toMatchObject({ pendingBaseline: true });
    }
    return;
  }
  await act(async () => { await change(270); });
  expect((await offlineStore.blobs(scope))[0].draft?.desired.elements[0].x).toBe(270);
  if (phase === 'completed-rebase-reload') {
    const saved = (await offlineStore.blobs(scope))[0].draft!;
    expect(saved.pendingBaseline).toBeUndefined(); expect(saved.base.elements[0].x).toBe(90);
    canvas.unmount();
    const reopened = new ExcalidrawAdapter(); reopened.applyRemote(Y.encodeStateAsUpdate(adapter.doc));
    mock.controller = { ...(mock.controller as object), adapter: reopened, content: reopened.content(), transport: { ...transport, flush: async () => {} }, applyLocalScene: (content: SceneContent) => reopened.applyLocalScene(content) };
    render(<CollaborativeCanvas title="normal replay" />);
    await waitFor(async () => expect(await offlineStore.blobs(scope)).toHaveLength(0));
    expect(reopened.content().elements[0]).toMatchObject({ x: 270, strokeColor: '#00aa00' });
    flush.resolve(); return;
  }
  await act(async () => { put.resolve(); flush.resolve(); });
  await waitFor(async () => expect(await offlineStore.blobs(scope)).toHaveLength(0));
  expect(adapter.content().elements[0]?.x).toBe(270);
  if (phase === 'during-flush') expect(adapter.content().elements[0]?.strokeColor).toBe('#00aa00');
});
