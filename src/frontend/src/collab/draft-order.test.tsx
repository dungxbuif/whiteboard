import 'fake-indexeddb/auto';
import { afterEach, expect, it, vi } from 'vitest';
import { act, render, waitFor, cleanup } from '@testing-library/react';
import type { SceneContent } from '@collab/contracts';
import { offlineStore } from './offline-store';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { createLocalUndo } from './undo';
import { rectangle } from './fixtures';
import { uploadBlob } from './assets';
const mock = vi.hoisted(() => ({ controller: undefined as unknown, change: undefined as unknown }));
vi.mock('./provider', () => ({ useCollab: () => mock.controller, CollabProvider: () => null }));
vi.mock('./assets', () => ({ uploadBlob: vi.fn() }));
vi.mock('@excalidraw/excalidraw', () => ({ Excalidraw: (props: { onChange: unknown }) => { mock.change = props.onChange; return null; }, restoreElements: (elements: unknown) => elements, CaptureUpdateAction: { NEVER: 'never' } }));
vi.mock('@/components/whiteboard/ai-command-panel', () => ({ AICommandPanel: () => null }));
import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
afterEach(() => { cleanup(); vi.resetAllMocks(); vi.unstubAllGlobals(); });

it.each(['initial-only', 'closed', 'same-scope-reopen', 'different-principal'])('collab ordered recovery preserves latest draft and exact late bytes: %s', async scenario => {
  // Node's standards-compatible Blob supports fake-indexeddb's structuredClone;
  // jsdom's presentation Blob does not. Native FileReader rendering is stubbed.
  const { Blob: ByteBlob } = await vi.importActual<{ Blob: typeof Blob }>('node:buffer');
  vi.stubGlobal('FileReader', class { result = 'data:image/png;base64,Ynl0ZXM='; onload?: () => void; readAsDataURL() { queueMicrotask(() => this.onload?.()); } });
  const scope = { principalId: 'alice', workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID(), documentId: 'epoch' };
  const other = scenario === 'different-principal' ? { ...scope, principalId: 'bob' } : scope;
  let releaseOld!: () => void, conversions = 0;
  const oldGate = new Promise<void>(resolve => { releaseOld = resolve; });
  vi.stubGlobal('fetch', vi.fn(async (url: string) => ({ blob: async () => { if (++conversions === 1) await oldGate; return new ByteBlob([url.includes('other') ? 'other-bytes' : 'original-bytes'], { type: 'image/png' }); } })));
  vi.mocked(uploadBlob).mockImplementation(() => new Promise(() => {}));
  function controller(target = scope) {
    const adapter = new ExcalidrawAdapter(), transport = { scope: target, state: { status: 'synced', canEdit: true, pending: 0, recovery: false }, isActive: () => true, flush: async () => {}, sendAwareness: () => {} };
    mock.controller = { adapter, transport, state: transport.state, content: adapter.content(), legacy: null, sceneRef: target, undo: createLocalUndo(adapter.doc), principal: { current: { kind: 'user', id: target.principalId } }, applyLocalScene: (content: SceneContent) => adapter.applyLocalScene(content), post: vi.fn() };
    return adapter;
  }
  function change(x: number, bytes = 'original') {
    const callback = mock.change as (elements: unknown[], appState: object, files: object) => Promise<void>;
    return callback([{ ...rectangle('image'), type: 'image', fileId: 'file', status: 'saved', scale: [1, 1], crop: null, x }], {}, { file: { id: 'file', mimeType: 'image/png', created: 1, dataURL: 'data:image/png;base64,' + bytes } });
  }
  controller(); const first = render(<CollaborativeCanvas title="first" />);
  await waitFor(() => expect(mock.change).toBeTypeOf('function'));
  let old!: Promise<void>;
  await act(async () => { old = change(0); });
  if (scenario !== 'initial-only') {
    await act(async () => { await change(90); });
    expect((await offlineStore.blobs(scope))[0].draft?.desired.elements[0].x).toBe(90);
  }
  first.unmount();
  if (scenario === 'same-scope-reopen' || scenario === 'different-principal') {
    mock.change = undefined;
    controller(other); const second = render(<CollaborativeCanvas title="second" />);
    await waitFor(() => expect(mock.change).toBeTypeOf('function'));
    await act(async () => { await change(180, scenario === 'different-principal' ? 'other' : 'original'); });
    await waitFor(async () => expect((await offlineStore.blobs(other))[0].draft?.desired.elements[0].x).toBe(180));
    second.unmount();
  }
  await act(async () => { releaseOld(); await old; });
  const expected = scenario === 'same-scope-reopen' ? 180 : scenario === 'initial-only' ? 0 : 90;
  const retained = (await offlineStore.blobs(scope))[0];
  expect(retained.draft?.desired.elements[0].x).toBe(expected);
  expect(await retained.blob.text()).toBe('original-bytes');
  if (scenario === 'different-principal') { const isolated = (await offlineStore.blobs(other))[0]; expect(isolated.draft?.desired.elements[0].x).toBe(180); expect(await isolated.blob.text()).toBe('other-bytes'); }
  vi.mocked(uploadBlob).mockResolvedValue({ id: 'file', assetId: 'stored', mimeType: 'image/png', created: 1 });
  mock.change = undefined; const reopened = controller(); render(<CollaborativeCanvas title="reopened" />);
  await waitFor(() => expect(reopened.content().elements[0]?.x).toBe(expected));
});
