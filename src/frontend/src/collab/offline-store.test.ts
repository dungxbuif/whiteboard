import 'fake-indexeddb/auto';
import { it, expect } from 'vitest';
import { OfflineStore, type PendingScope } from './offline-store';
const scope: PendingScope = { principalId: 'alice', workspaceId: 'workspace', sceneId: 'scene', documentId: 'epoch' };
it('collab persists bytes across reload and clears only a matching scoped acknowledgement', async () => {
  const name = crypto.randomUUID(); const first = new OfflineStore(name);
  await first.enqueuePending(scope, 'request', new Uint8Array([1, 2, 3]));
  const reloaded = new OfflineStore(name);
  expect(Array.from((await reloaded.pending(scope))[0].update)).toEqual([1, 2, 3]);
  for (const wrong of [{ ...scope, principalId: 'bob' }, { ...scope, workspaceId: 'other' }, { ...scope, sceneId: 'other' }, { ...scope, documentId: 'new-generation' }]) await reloaded.ackPending(wrong, 'request');
  await reloaded.ackPending(scope, 'unknown'); expect(await reloaded.pending(scope)).toHaveLength(1);
  expect(await reloaded.pending({ ...scope, principalId: 'bob' })).toEqual([]);
  await reloaded.ackPending(scope, 'request'); expect(await reloaded.pending(scope)).toEqual([]);
});
it('collab request identity cannot be reused with changed bytes', async () => {
  const store = new OfflineStore(crypto.randomUUID());
  await store.enqueuePending(scope, 'request', new Uint8Array([1]));
  await expect(store.enqueuePending(scope, 'request', new Uint8Array([2]))).rejects.toThrow(/conflict/i);
});

it('upgrades a v2 database without discarding an unreconstructable pending row',async()=>{
  const name=crypto.randomUUID(),opening=indexedDB.open(name,2);await new Promise<void>((resolve,reject)=>{opening.onupgradeneeded=()=>{for(const store of ['pending','snapshots','blobs','drafts'])opening.result.createObjectStore(store);};opening.onsuccess=()=>resolve();opening.onerror=()=>reject(opening.error);});const db=opening.result;const tx=db.transaction('pending','readwrite');tx.objectStore('pending').put({scope,requestId:'legacy',update:new Uint8Array([1]),created:1,sent:false},JSON.stringify([JSON.stringify([scope.principalId,scope.workspaceId,scope.sceneId,scope.documentId]),'legacy']));await new Promise<void>((resolve,reject)=>{tx.oncomplete=()=>resolve();tx.onerror=()=>reject(tx.error);});db.close();
  const store=new OfflineStore(name);const row=(await store.pending(scope))[0];expect(row.requestId).toBe('legacy');expect(row.intent).toBeUndefined();const probe=indexedDB.open(name);const upgraded=await new Promise<IDBDatabase>((resolve,reject)=>{probe.onsuccess=()=>resolve(probe.result);probe.onerror=()=>reject(probe.error);});expect(upgraded.version).toBe(3);upgraded.close();
});

it('atomically replaces every never-sent row in creation order rather than IndexedDB key order',async()=>{
  const store=new OfflineStore(crypto.randomUUID());await store.enqueuePending(scope,'z-request',new Uint8Array([1]));await store.enqueuePending(scope,'a-request',new Uint8Array([2]));const rows=await store.pending(scope);
  expect(rows.map(row=>row.requestId)).toEqual(['z-request','a-request']);
  expect(await store.replaceUnsent(scope,rows,{scope,requestId:'rebased',update:new Uint8Array([3]),created:rows[0].created})).toBe(true);
  expect((await store.pending(scope)).map(row=>row.requestId)).toEqual(['rebased']);
});

it('collab checkpoint and pending bytes commit together, and batch retries preserve sent bytes', async () => {
  const Y = await import('yjs'); const doc = new Y.Doc(); const store = new OfflineStore(crypto.randomUUID());
  doc.getMap('appState').set('theme', 'dark'); const one = Y.encodeStateAsUpdate(doc);
  await store.enqueuePending(scope, 'one', one, one);
  const vector = Y.encodeStateVector(doc); doc.getMap('appState').set('gridSize', 20); const two = Y.encodeStateAsUpdate(doc, vector);
  await store.enqueuePending(scope, 'two', two, Y.encodeStateAsUpdate(doc));
  const cached = await store.snapshots(scope.principalId, scope.workspaceId, scope.sceneId); expect(cached).toHaveLength(1);
  const batch = await store.nextBatch(scope, 1000000); expect(await store.pending(scope)).toHaveLength(1);
  expect(batch?.sent).toBe(true); expect(Array.from((await store.nextBatch(scope, 1000000))!.update)).toEqual(Array.from(batch!.update));
  expect((await store.nextBatch(scope, 1000000))!.requestId).toBe(batch!.requestId);
});

it('collab image revisions survive store reopen and reject stale cleanup or resurrection', async () => {
  const name = crypto.randomUUID(), first = new OfflineStore(name), reopened = new OfflineStore(name);
  const base = { elements: [], appState: {}, files: {} };
  const old = { base, desired: { ...base, appState: { gridSize: 10 } } }, latest = { base, desired: { ...base, appState: { gridSize: 20 } } };
  const one = await first.reserveImageDraft(scope, old), two = await reopened.reserveImageDraft(scope, latest);
  const item = { scope, id: 'file', blob: new Blob(['bytes']), mimeType: 'image/png', created: 1, draft: old };
  await first.putImageBlob(item, one);
  expect((await reopened.blobs(scope))[0].draft?.desired).toEqual(latest.desired);
  expect(await first.completeImageDraft(scope, one, ['file'])).toBe(false);
  expect(await reopened.blobs(scope)).toHaveLength(1);
  expect(await reopened.completeImageDraft(scope, two, ['file'])).toBe(true);
  await first.putImageBlob(item, one);
  expect(await reopened.blobs(scope)).toHaveLength(0);
  expect(await reopened.draft(scope)).toBeUndefined();
  expect(await first.reserveImageDraft(scope, latest)).toBeGreaterThan(two);
});
