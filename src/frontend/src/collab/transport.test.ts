import 'fake-indexeddb/auto';
import { it, expect, vi } from 'vitest';
import * as Y from 'yjs';
import { decodeFrame, encodeFrame } from '@collab/contracts/protocol';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { OfflineStore } from './offline-store';
import { CollabTransport, type WireSocket } from './transport';
import { rectangle, scene } from './fixtures';
class Socket extends EventTarget {
  readyState = 1; binaryType: BinaryType = 'arraybuffer'; sent: (string | Uint8Array)[] = [];
  send(data: string | Uint8Array) { this.sent.push(data); }
  close() { this.readyState = 3; }
  receive(data: unknown) { this.dispatchEvent(new MessageEvent('message', { data: JSON.stringify(data) })); }
}
const tick = () => new Promise(resolve => setTimeout(resolve, 30));
const scope = { principalId: 'alice', workspaceId: 'workspace', sceneId: 'scene', documentId: 'epoch' };
function synced(socket: Socket, server: ExcalidrawAdapter, access = 'edit', documentId = 'epoch') {
  socket.receive({ type: 'ready', requestId: 'auth', payload: { ...scope, documentId, access, sequence: '1', stateVector: '' } });
  const bytes = Y.encodeStateAsUpdate(server.doc);
  socket.receive({ type: 'sync_chunk', requestId: 'auth', payload: { documentId, index: 0, count: 1, totalBytes: bytes.length, data: btoa(String.fromCharCode(...bytes)) } });
}
it('collab reload replays retained updates only after synchronization and matching ACK clears them', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const author = new ExcalidrawAdapter(); author.applyRemote(Y.encodeStateAsUpdate(server.doc));
  const vector = Y.encodeStateVector(author.doc); await author.applyAgentElements([rectangle('offline')]);
  await store.enqueuePending(scope, 'durable-request', Y.encodeStateAsUpdate(author.doc, vector));
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); await tick();
  expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  synced(socket, server); await tick();
  expect(adapter.content().elements.map(e => e.id)).toContain('offline');
  const frames = socket.sent.filter((item): item is Uint8Array => typeof item !== 'string'); expect(frames).toHaveLength(1);
  expect(decodeFrame(frames[0]).updateId).toBe('durable-request');
  socket.receive({ type: 'ack', requestId: 'other', payload: { requestId: 'other', sequence: '2', stateVector: '' } }); await tick();
  expect(await store.pending(scope)).toHaveLength(1);
  socket.receive({ type: 'ack', requestId: 'durable-request', payload: { requestId: 'durable-request', sequence: '2', stateVector: '' } }); await tick();
  expect(await store.pending(scope)).toHaveLength(0); transport.stop();
});
it.each(['view', 'wrong-generation'])('collab %s admission retains recovery without replay writes', async mode => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  await store.enqueuePending(scope, 'pending', Y.encodeStateAsUpdate(server.doc));
  const socket = new Socket(); const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter: new ExcalidrawAdapter(), store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server, mode === 'view' ? 'view' : 'edit', mode === 'wrong-generation' ? 'other' : 'epoch'); await tick();
  expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  expect(await store.pending(scope)).toHaveLength(1); transport.stop();
});

it('collab awareness strips native pointer tool metadata before protocol transmission', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(); const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter: new ExcalidrawAdapter(), store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  const native = { x: 1, y: 2, tool: 'pointer' }; transport.sendAwareness({ cursor: native });
  const frame = decodeFrame(socket.sent.filter((item): item is Uint8Array => typeof item !== 'string').at(-1)!);
  expect(JSON.parse(new TextDecoder().decode(frame.payload))).toEqual({ cursor: { x: 1, y: 2 } }); transport.stop();
});

it('collab revoked editor retains pending recovery and stale connection ACK cannot clear it', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  const desired = adapter.content(); desired.elements[0].x = 52; adapter.applyLocalScene(desired); await transport.whenDurable();
  socket.receive({ type: 'error', requestId: 'server', payload: { code: 'FORBIDDEN', retryable: false } }); await tick(); await transport.flush();
  expect(transport.state.canEdit).toBe(false); expect(transport.state.status).toBe('revoked');
  expect(await store.pending(scope)).toHaveLength(1); expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  const row = (await store.pending(scope))[0]; transport.stop();
  socket.receive({ type: 'ack', requestId: row.requestId, payload: { requestId: row.requestId, sequence: '2' } }); await tick();
  expect(await store.pending(scope)).toHaveLength(1);
  expect((await transport.exportRecovery()).scopes[0].pending).toHaveLength(1);
});
it('collab account switch reads only its own principal scope', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  await store.enqueuePending(scope, 'alice-private', Y.encodeStateAsUpdate(server.doc));
  const socket = new Socket(); const transport = new CollabTransport({ sceneRef: scope, principalId: 'bob', adapter: new ExcalidrawAdapter(), store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open'));
  socket.receive({ type: 'ready', requestId: 'auth', payload: { ...scope, principalId: 'bob', access: 'edit', sequence: '1' } });
  const bytes = Y.encodeStateAsUpdate(server.doc); socket.receive({ type: 'sync_chunk', requestId: 'auth', payload: { documentId: scope.documentId, index: 0, count: 1, totalBytes: bytes.length, data: btoa(String.fromCharCode(...bytes)) } }); await tick();
  expect(transport.state.status).toBe('synced'); expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  expect(await store.pending(scope)).toHaveLength(1); transport.stop();
});

it('collab buffers committed updates interleaved with snapshot chunks and never echoes them', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open'));
  const snapshot = Y.encodeStateAsUpdate(server.doc), vector = Y.encodeStateVector(server.doc); await server.applyAgentElements([rectangle('interleaved')]);
  socket.receive({ type: 'ready', requestId: 'auth', payload: { ...scope, access: 'edit', sequence: '1' } });
  const bytes = encodeFrame({ channel: 3, updateId: 'interleaved', payload: Y.encodeStateAsUpdate(server.doc, vector) }); socket.dispatchEvent(new MessageEvent('message', { data: bytes.buffer }));
  socket.receive({ type: 'sync_chunk', requestId: 'auth', payload: { documentId: scope.documentId, index: 0, count: 1, totalBytes: snapshot.length, data: btoa(String.fromCharCode(...snapshot)) } }); await tick();
  expect(adapter.content().elements.map(e => e.id)).toContain('interleaved');
  expect(transport.state.status).toBe('synced'); expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0); transport.stop();
});

it.each([403, 503])('collab authority status %s clears credentials only for definitive revocation', async status => {
  const store = new OfflineStore(crypto.randomUUID()); let lost = 0;
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter: new ExcalidrawAdapter(), store, grant: async () => { throw { response: { status } }; }, url: 'ws://local', onAccessLost: () => { lost++; } });
  await transport.start(); expect(lost).toBe(status === 403 ? 1 : 0); transport.stop();
});

it('collab overlapping scheduled and explicit flush owns one frame and no ACK timeout after acknowledgement', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
  let release!: () => void; const gate = new Promise<void>(resolve => { release = resolve; });
  const nextBatch = store.nextBatch.bind(store); let entered!: () => void; const entering = new Promise<void>(resolve => { entered = resolve; });
  vi.spyOn(store, 'nextBatch').mockImplementation(async (...args) => { entered(); await gate; return nextBatch(...args); });
  try {
    const desired = adapter.content(); desired.elements[0].x = 80; adapter.applyLocalScene(desired); await transport.whenDurable();
    const explicit = transport.flush(); await entering;
    await vi.advanceTimersByTimeAsync(600); release(); await explicit;
    // IndexedDB callbacks use real immediates; wait until both contenders finish.
    await vi.waitFor(() => expect(store.nextBatch).toHaveBeenCalled(), { interval: 1 });
    await new Promise<void>(resolve => { const tx = indexedDB.open(crypto.randomUUID()); tx.onsuccess = () => { tx.result.close(); resolve(); }; });
    const row = (await store.pending(scope))[0];
    socket.receive({ type: 'ack', requestId: row.requestId, payload: { requestId: row.requestId, sequence: '2' } });
    await vi.waitFor(async () => expect(await store.pending(scope)).toHaveLength(0), { interval: 1 });
    await vi.advanceTimersByTimeAsync(16000);
    expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(1);
    expect(socket.readyState).toBe(1);
  } finally { transport.stop(); vi.useRealTimers(); vi.restoreAllMocks(); }
});

it('collab stopping transport immediately revokes edit eligibility', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(), transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter: new ExcalidrawAdapter(), store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  expect(transport.state.canEdit).toBe(true); transport.stop(); expect(transport.state.canEdit).toBe(false);
});

it('collab occupied send slot still waits for caller-local durability', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  let releaseBatch!: () => void, releaseDurable!: () => void;
  const batchGate = new Promise<void>(resolve => { releaseBatch = resolve; });
  const durableGate = new Promise<void>(resolve => { releaseDurable = resolve; });
  const next = store.nextBatch.bind(store), enqueue = store.enqueuePending.bind(store);
  vi.spyOn(store, 'nextBatch').mockImplementation(async (...args) => { await batchGate; return next(...args); });
  const first = transport.flush(); await tick();
  vi.spyOn(store, 'enqueuePending').mockImplementation(async (...args) => { await durableGate; return enqueue(...args); });
  const desired = adapter.content(); desired.elements[0].x = 99; adapter.applyLocalScene(desired);
  let returned = false; const second = transport.flush().then(() => { returned = true; });
  try { await tick(); expect(returned).toBe(false); releaseDurable(); await second; expect(await store.pending(scope)).toHaveLength(1); }
  finally { releaseDurable(); releaseBatch(); await first; transport.stop(); vi.restoreAllMocks(); }
});
