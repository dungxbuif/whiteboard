import 'fake-indexeddb/auto';
import { afterEach, expect, it, vi } from 'vitest';
import { act, render, waitFor, cleanup, fireEvent } from '@testing-library/react';
import type { SceneContent } from '@collab/contracts';
import { offlineStore } from './offline-store';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { createLocalUndo } from './undo';
import { rectangle } from './fixtures';
const mock = vi.hoisted(() => ({ controller: undefined as unknown, changes: new Map<string, (elements: unknown[], state: object, files: object) => Promise<void>>() }));
// Pin each rendered canvas to its own controller, as two provider roots/tabs do.
vi.mock('./provider', async () => { const { useState } = await import('react'); return { useCollab: () => useState(() => mock.controller)[0], CollabProvider: () => null }; });
vi.mock('@excalidraw/excalidraw', () => ({ Excalidraw: (props: { name: string; onChange: typeof mock.changes extends Map<string, infer T> ? T : never }) => { mock.changes.set(props.name, props.onChange); return null; }, restoreElements: (elements: unknown) => elements, CaptureUpdateAction: { NEVER: 'never' } }));
vi.mock('@/components/whiteboard/ai-command-panel', () => ({ AICommandPanel: () => null }));
import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
afterEach(() => { cleanup(); mock.changes.clear(); vi.resetAllMocks(); vi.unstubAllGlobals(); });
function deferred() { let resolve!: () => void; const promise = new Promise<void>(done => { resolve = done; }); return { promise, resolve }; }

it.each(['during-put', 'during-flush'])('collab active same-scope conflict stops stale uploads and keeps newest recovery: %s', async phase => {
  const { Blob: ByteBlob } = await vi.importActual<{ Blob: typeof Blob }>('node:buffer');
  const scope = { principalId: 'alice', workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID(), documentId: 'epoch' };
  const put = deferred(), flush = deferred(), reachedFlush = deferred();
  vi.stubGlobal('FileReader', class { result = 'data:image/png;base64,Ynl0ZXM='; onload?: () => void; readAsDataURL() { queueMicrotask(() => this.onload?.()); } });
  let attempts = 0, firstActive = true;
  vi.stubGlobal('fetch', vi.fn(async (url: string) => {
    if (url.startsWith('data:')) return { blob: async () => new ByteBlob(['retained-bytes'], { type: 'image/png' }) };
    if (url === 'https://assets.test/second') return new Promise(() => {});
    expect(url).toBe('https://assets.test/first');
    attempts++;
    // Bound the RED reproduction; production must stop without this lifetime escape.
    if (attempts >= 3) firstActive = false;
    await put.promise;
    return { ok: true };
  }));
  function controller(first: boolean) {
    const adapter = new ExcalidrawAdapter();
    const transport = { scope, state: { status: 'synced', canEdit: true, pending: 0, recovery: false }, isActive: () => !first || firstActive, flush: async () => { if (first) { reachedFlush.resolve(); await flush.promise; } }, sendAwareness: () => {}, retry: () => {} };
    const applyLocalScene = vi.fn((content: SceneContent) => adapter.applyLocalScene(content));
    mock.controller = { adapter, transport, state: transport.state, content: adapter.content(), legacy: null, sceneRef: scope, undo: createLocalUndo(adapter.doc), principal: { current: { kind: 'user', id: scope.principalId } }, applyLocalScene, post: async () => ({ urls: { file: { assetId: 'stored', uploadUrl: `https://assets.test/${first ? 'first' : 'second'}`, headers: { 'Content-Type': 'image/png' } } } }) };
    return { adapter, applyLocalScene };
  }
  async function change(name: string, x: number) {
    await mock.changes.get(name)!([{ ...rectangle('image'), type: 'image', fileId: 'file', status: 'saved', scale: [1, 1], crop: null, x }, ...(x === 270 ? [{ ...rectangle('next-local-edit'), x: 270 }] : [])], {}, { file: { id: 'file', mimeType: 'image/png', created: 1, dataURL: 'data:image/png;base64,Ynl0ZXM=' } });
  }
  const first = controller(true), firstCanvas = render(<CollaborativeCanvas title="first" />);
  await waitFor(() => expect(mock.changes.has('first')).toBe(true));
  // Mount the second active controller before the first draft exists, so it starts clean.
  controller(false); render(<CollaborativeCanvas title="second" />);
  await waitFor(() => expect(mock.changes.has('second')).toBe(true));
  await act(async () => { await change('first', 90); });
  await waitFor(() => expect(attempts).toBe(1));
  if (phase === 'during-flush') await act(async () => { put.resolve(); await reachedFlush.promise; });
  await act(async () => { await change('second', 180); });
  expect((await offlineStore.blobs(scope))[0].draft?.desired.elements[0].x).toBe(180);
  await act(async () => { put.resolve(); flush.resolve(); });
  await waitFor(() => expect(attempts >= 3 || firstCanvas.container.querySelector('[role="alert"]')).toBeTruthy());
  expect(attempts).toBe(1);
  expect(firstActive).toBe(true);
  expect(first.applyLocalScene).toHaveBeenCalledTimes(phase === 'during-put' ? 0 : 1);
  expect(firstCanvas.container.querySelector('[role="alert"]')?.textContent).toMatch(/recovery/i);
  const retained = (await offlineStore.blobs(scope))[0];
  expect(retained.draft?.desired.elements[0].x).toBe(180);
  expect(await retained.blob.text()).toBe('retained-bytes');
  // Explicit retry must also reject the unchanged stale revision without another PUT.
  fireEvent.click(firstCanvas.getAllByText('Retry connection / uploads')[0]);
  await waitFor(() => expect(firstCanvas.container.querySelector('[role="alert"]')).toBeTruthy());
  expect(attempts).toBe(1);
  // The conflict does not disable the canvas: a deliberate newer local edit can commit.
  await act(async () => { await change('first', 270); });
  await waitFor(async () => { expect(first.adapter.content().elements.find(element => element.id === 'next-local-edit')?.x).toBe(270); expect(await offlineStore.blobs(scope)).toHaveLength(0); });
  expect(attempts).toBe(2);
});
