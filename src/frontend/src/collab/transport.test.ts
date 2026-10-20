import 'fake-indexeddb/auto';
import { it, expect, vi } from 'vitest';
import * as Y from 'yjs';
import { decodeFrame, encodeFrame } from '@collab/contracts/protocol';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { OfflineStore } from './offline-store';
import { CollabTransport, type Control, type WireSocket } from './transport';
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
it('collab legacy pending bytes fail closed after synchronization and remain recoverable', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const author = new ExcalidrawAdapter(); author.applyRemote(Y.encodeStateAsUpdate(server.doc));
  const vector = Y.encodeStateVector(author.doc); await author.applyAgentElements([rectangle('offline')]);
  await store.enqueuePending(scope, 'durable-request', Y.encodeStateAsUpdate(author.doc, vector));
  const socket = new Socket(), adapter = new ExcalidrawAdapter();
  const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter, store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  await transport.start(); socket.dispatchEvent(new Event('open')); await tick();
  expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  synced(socket, server); await tick();
  expect(transport.state.sequence).toBe('1');
  expect(adapter.content().elements.map(e => e.id)).toContain('offline');
  await vi.waitFor(() => expect(transport.state.status).toBe('degraded'));
  expect(socket.sent.filter(item => typeof item !== 'string')).toHaveLength(0);
  expect(await store.pending(scope)).toHaveLength(1);transport.stop();
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

it('uses presentation_auth for waiting clients and surfaces only the admitted credential',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),socket=new Socket(),admitted:unknown[]=[];
 const transport=new CollabTransport({sceneRef:scope,principalId:'presentation:guest',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'pending-token',expiresAt:new Date(Date.now()+300000).toISOString()}),url:'ws://local',authType:'presentation_auth',onPresentationAdmitted:value=>admitted.push(value),socket:()=>socket as unknown as WireSocket});
 await transport.start();socket.dispatchEvent(new Event('open'));await tick();expect(JSON.parse(socket.sent.find((value):value is string=>typeof value==='string')!)).toMatchObject({type:'presentation_auth',payload:{token:'pending-token'}});
 socket.receive({type:'presentation_admitted',requestId:'admit',payload:{sessionId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',token:'audience-token',expiresAt:Math.floor(Date.now()/1000)+600}});await tick();expect(admitted).toHaveLength(1);expect(socket.sent.filter(value=>typeof value==='string')).toHaveLength(1);transport.stop();
});

it('surfaces a rejected presentation_auth as terminal instead of reconnecting joining state',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),socket=new Socket(),controls:Control[]=[];
 const transport=new CollabTransport({sceneRef:scope,principalId:'presentation:guest',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'expired-pending',expiresAt:new Date(Date.now()+300000).toISOString()}),url:'ws://local',authType:'presentation_auth',socket:()=>socket as unknown as WireSocket});transport.subscribeControl(value=>controls.push(value));
 await transport.start();socket.dispatchEvent(new Event('open'));socket.receive({type:'error',requestId:'auth',payload:{code:'PRESENTATION_ENDED',message:'Presentation has ended',retryable:false}});await tick();
 expect(controls.at(-1)).toMatchObject({type:'error',payload:{code:'PRESENTATION_ENDED'}});expect(transport.state.status).toBe('revoked');transport.stop();
});

it('aborts a pending presentation start by closing the exact active socket',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),socket=new Socket(),transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'host-token',expiresAt:new Date(Date.now()+300000).toISOString()}),url:'ws://local',socket:()=>socket as unknown as WireSocket});
 const close=vi.spyOn(socket,'close');await transport.start();socket.dispatchEvent(new Event('open'));transport.abortPresentationStart();
 expect(close).toHaveBeenCalledOnce();expect(socket.readyState).toBe(3);transport.stop();
});

it('forwards cursor-only feature invalidations without disturbing document admission', async () => {
  const store = new OfflineStore(crypto.randomUUID()), server = new ExcalidrawAdapter(); server.applyLocalScene(scene());
  const socket = new Socket(); const transport = new CollabTransport({ sceneRef: scope, principalId: 'alice', adapter: new ExcalidrawAdapter(), store, grant: async () => ({ token: 'token', expiresAt: new Date(Date.now() + 600000).toISOString() }), url: 'ws://local', socket: () => socket as unknown as WireSocket });
  const invalidations: unknown[] = []; transport.subscribeControl(control => { if (control.type === 'feature_changed') invalidations.push(control.payload); });
  await transport.start(); socket.dispatchEvent(new Event('open')); synced(socket, server); await tick();
  socket.receive({ type: 'feature_changed', requestId: 'server', payload: { eventCursor: '42' } }); await tick();
  expect(invalidations).toEqual([{ eventCursor: '42' }]); expect(transport.state.status).toBe('synced');
  transport.stop();
});

it('advances the current sequence after a remote committed update',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),server=new ExcalidrawAdapter();server.applyLocalScene(scene());const socket=new Socket(),transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>socket as unknown as WireSocket});
 await transport.start();socket.dispatchEvent(new Event('open'));synced(socket,server);await tick();
 socket.receive({type:'committed',requestId:'server',payload:{sequence:'2'}});await tick();expect(transport.state.sequence).toBe('2');
 socket.receive({type:'committed',requestId:'server',payload:{sequence:'1'}});await tick();expect(transport.state.sequence).toBe('2');transport.stop();
});
it('never moves the durable sequence backwards on a late acknowledgement',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),server=new ExcalidrawAdapter();server.applyLocalScene(scene());const socket=new Socket(),adapter=new ExcalidrawAdapter();
 const transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter,store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>socket as unknown as WireSocket});
 await transport.start();socket.dispatchEvent(new Event('open'));synced(socket,server);await tick();adapter.applyLocalScene(scene([rectangle('local')]));await transport.whenDurable();await transport.flush();const frame=socket.sent.find((item):item is Uint8Array=>typeof item!=='string')!;const id=decodeFrame(frame).updateId;
 socket.receive({type:'committed',requestId:'server',payload:{sequence:'9'}});socket.receive({type:'ack',requestId:id,payload:{requestId:id,sequence:'2',stateVector:''}});await tick();expect(transport.state.sequence).toBe('9');transport.stop();
});
it('rebases a never-sent offline image edit causally after a restore sync',async()=>{
 const file={id:'file',assetId:'00000000-0000-4000-8000-000000000001',mimeType:'image/png',created:1},image={...rectangle('image'),type:'image' as const,fileId:'file',status:'saved' as const,scale:[1,1] as [number,number],crop:null};
 const base={...scene([image]),files:{file}},server=new ExcalidrawAdapter();server.applyLocalScene(base);const sockets=[new Socket(),new Socket()],store=new OfflineStore(crypto.randomUUID()),adapter=new ExcalidrawAdapter();
 const transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter,store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>sockets.shift() as unknown as WireSocket});
 await transport.start();const first=(transport as any).socket as Socket;first.dispatchEvent(new Event('open'));synced(first,server);await tick();window.dispatchEvent(new Event('offline'));let release!:()=>void;const gate=new Promise<void>(resolve=>release=resolve),enqueue=store.enqueuePending.bind(store);vi.spyOn(store,'enqueuePending').mockImplementation(async(...args)=>{await gate;return enqueue(...args);});const desired=adapter.content();desired.elements[0].x=88;adapter.applyLocalScene(desired);const restored=server.content();restored.elements[0].isDeleted=true;server.applyLocalScene(restored);
 window.dispatchEvent(new Event('online'));await tick();const second=(transport as any).socket as Socket;second.dispatchEvent(new Event('open'));synced(second,server);await tick();expect(second.sent.some(value=>typeof value!=='string')).toBe(false);release();await vi.waitFor(()=>expect(second.sent.some(value=>typeof value!=='string')).toBe(true));const frame=second.sent.find((value):value is Uint8Array=>typeof value!=='string')!;server.applyRemote(decodeFrame(frame).payload);expect(server.content().elements[0]).toMatchObject({x:88,isDeleted:false});transport.stop();vi.restoreAllMocks();
});
it('preserves sequential edit then revert intent across restore reconnect',async()=>{
 const firstElement=rectangle('first'),secondElement=rectangle('second'),base={...scene([firstElement,secondElement]),appState:{gridSize:20},files:{}};const server=new ExcalidrawAdapter();server.applyLocalScene(base);const sockets=[new Socket(),new Socket()],store=new OfflineStore(crypto.randomUUID()),adapter=new ExcalidrawAdapter(),transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter,store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>sockets.shift() as unknown as WireSocket});
 await transport.start();const first=(transport as any).socket as Socket;first.dispatchEvent(new Event('open'));synced(first,server);await tick();window.dispatchEvent(new Event('offline'));const intermediate=structuredClone(base);intermediate.elements[0].x=10;intermediate.elements.reverse();intermediate.appState={gridSize:40};intermediate.files={temporary:{id:'temporary',assetId:'00000000-0000-4000-8000-000000000001',mimeType:'image/png',created:1}};adapter.applyLocalScene(intermediate);await transport.whenDurable();adapter.applyLocalScene(base);await transport.whenDurable();expect(await store.pending(scope)).toHaveLength(2);
 const restored=server.content();restored.appState={...restored.appState,viewBackgroundColor:'#eeeeee'};server.applyLocalScene(restored);window.dispatchEvent(new Event('online'));await tick();const second=(transport as any).socket as Socket;second.dispatchEvent(new Event('open'));synced(second,server);await vi.waitFor(()=>expect(second.sent.some(value=>typeof value!=='string')).toBe(true));server.applyRemote(decodeFrame(second.sent.find((value):value is Uint8Array=>typeof value!=='string')!).payload);expect(server.content().elements.map(element=>({id:element.id,x:element.x}))).toEqual(base.elements.map(element=>({id:element.id,x:element.x})));expect(server.content().appState.gridSize).toBe(20);expect(server.content().files).not.toHaveProperty('temporary');transport.stop();
});
it('rebases safe rows but fails closed when a sent row is ambiguous after reconnect',async()=>{
 const server=new ExcalidrawAdapter();server.applyLocalScene(scene([rectangle('base')]));const sockets=[new Socket(),new Socket()],store=new OfflineStore(crypto.randomUUID()),adapter=new ExcalidrawAdapter(),transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter,store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>sockets.shift() as unknown as WireSocket});
 await transport.start();const first=(transport as any).socket as Socket;first.dispatchEvent(new Event('open'));synced(first,server);await tick();window.dispatchEvent(new Event('offline'));const changed=adapter.content();changed.elements[0].x=10;adapter.applyLocalScene(changed);await transport.whenDurable();const original=(await store.pending(scope))[0].requestId;await store.nextBatch(scope,1_000_000);const later=adapter.content();later.elements[0].y=20;adapter.applyLocalScene(later);await transport.whenDurable();server.applyLocalScene(scene([]));window.dispatchEvent(new Event('online'));await tick();const second=(transport as any).socket as Socket;second.dispatchEvent(new Event('open'));synced(second,server);await vi.waitFor(()=>expect(transport.state.status).toBe('degraded'));expect(second.sent.some(value=>typeof value!=='string')).toBe(false);const retained=await store.pending(scope);expect(retained.some(row=>row.requestId===original&&row.sent)).toBe(true);expect(retained.some(row=>row.requestId!==original&&row.intent&&!row.sent)).toBe(true);transport.stop();
});
it('never sends raw offline bytes when the atomic causal replacement loses a race',async()=>{
 const server=new ExcalidrawAdapter();server.applyLocalScene(scene());const sockets=[new Socket(),new Socket()],store=new OfflineStore(crypto.randomUUID()),adapter=new ExcalidrawAdapter(),transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter,store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>sockets.shift() as unknown as WireSocket});
 await transport.start();const first=(transport as any).socket as Socket;first.dispatchEvent(new Event('open'));synced(first,server);await tick();window.dispatchEvent(new Event('offline'));adapter.applyLocalScene(scene([rectangle('offline')]));await transport.whenDurable();vi.spyOn(store,'replaceUnsent').mockResolvedValue(false);window.dispatchEvent(new Event('online'));await tick();const second=(transport as any).socket as Socket;second.dispatchEvent(new Event('open'));synced(second,server);await vi.waitFor(()=>expect(transport.state.status).toBe('degraded'));expect(second.sent.some(value=>typeof value!=='string')).toBe(false);expect(await store.pending(scope)).toHaveLength(1);transport.stop();vi.restoreAllMocks();
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

it('collab correlates ephemeral failures without degrading document admission or losing the connection ID', async () => {
 const store=new OfflineStore(crypto.randomUUID()),server=new ExcalidrawAdapter(),socket=new Socket();server.applyLocalScene(scene());
 const transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>socket as unknown as WireSocket});
 const controls:unknown[]=[];transport.subscribeControl(c=>controls.push(c));
 try {
  await transport.start();socket.dispatchEvent(new Event('open'));synced(socket,server);await tick();
  const id=transport.sendControl('voice_join',{});expect(id).toEqual(expect.any(String));
  socket.receive({type:'error',requestId:id,payload:{code:'VOICE_TARGET',retryable:false}});await tick();
  expect(transport.state.status).toBe('synced');expect(transport.state.canEdit).toBe(true);expect(controls).toContainEqual(expect.objectContaining({type:'error',requestId:id}));
  socket.receive({type:'error',requestId:'server',payload:{code:'PRESENCE_RATE_LIMITED',retryable:false}});await tick();expect(transport.state.status).toBe('synced');
  socket.receive({type:'error',requestId:transport.sendControl('voice_join',{}),payload:{code:'FORBIDDEN'}});await tick();expect(transport.state.status).toBe('revoked');
 } finally {transport.stop();}
});

it('coalesces cursor and final selection within the awareness budget and drops queued state on disconnect',async()=>{
 const store=new OfflineStore(crypto.randomUUID()),server=new ExcalidrawAdapter();server.applyLocalScene(scene());const socket=new Socket();const transport=new CollabTransport({sceneRef:scope,principalId:'alice',adapter:new ExcalidrawAdapter(),store,grant:async()=>({token:'token',expiresAt:new Date(Date.now()+600000).toISOString()}),url:'ws://local',socket:()=>socket as unknown as WireSocket});
 await transport.start();socket.dispatchEvent(new Event('open'));synced(socket,server);await tick();
 transport.sendAwareness({cursor:{x:1,y:2}});transport.sendAwareness({selectedIds:['selected']});transport.sendAwareness({cursor:{x:3,y:4}});await new Promise(r=>setTimeout(r,60));
 const frames=socket.sent.filter((v):v is Uint8Array=>typeof v!=='string').map(decodeFrame).filter(f=>f.channel===2);expect(frames).toHaveLength(2);expect(JSON.parse(new TextDecoder().decode(frames[1].payload))).toEqual({selectedIds:['selected'],cursor:{x:3,y:4}});
 transport.sendAwareness({selectedIds:[]});transport.stop();const count=socket.sent.length;await new Promise(r=>setTimeout(r,60));expect(socket.sent).toHaveLength(count);
});
