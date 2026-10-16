import * as Y from 'yjs';
import type { SceneContent } from '@collab/contracts';
export type PendingScope = { principalId: string; workspaceId: string; sceneId: string; documentId: string };
export type PendingUpdate = { scope: PendingScope; requestId: string; update: Uint8Array; created: number; sent?: boolean };
export type ImageDraft = { base: SceneContent; desired: SceneContent; revision?: number };
type DraftRecord = { scope: PendingScope; content?: SceneContent; imageDraft?: ImageDraft; revision?: number };
export type PendingBlob = { scope: PendingScope; id: string; blob: Blob; mimeType: string; created: number; draft?: ImageDraft };
export const scopeKey = (s: PendingScope) => JSON.stringify([s.principalId, s.workspaceId, s.sceneId, s.documentId]);
const key = (scope: PendingScope, id: string) => JSON.stringify([scopeKey(scope), id]);
const request = <T>(req: IDBRequest<T>) => new Promise<T>((resolve, reject) => { req.onsuccess = () => resolve(req.result); req.onerror = () => reject(req.error); });
export class OfflineStore {
  private database?: Promise<IDBDatabase>;
  constructor(private name = 'devtool-collab-v2') {}
  private open() {
    return this.database ??= new Promise<IDBDatabase>((resolve, reject) => {
      const opening = indexedDB.open(this.name, 2);
      opening.onupgradeneeded = () => { for (const name of ['pending', 'snapshots', 'blobs', 'drafts']) if (!opening.result.objectStoreNames.contains(name)) opening.result.createObjectStore(name); };
      opening.onsuccess = () => resolve(opening.result); opening.onerror = () => reject(opening.error);
    });
  }
  private async transaction<T>(name: string | string[], mode: IDBTransactionMode, work: (store: IDBObjectStore) => Promise<T>): Promise<T> {
    const db = await this.open(), tx = db.transaction(name, mode);
    const done = new Promise<void>((resolve, reject) => { tx.oncomplete = () => resolve(); tx.onabort = tx.onerror = () => reject(tx.error ?? new Error('Local storage failed')); });
    try { const value = await work(tx.objectStore(Array.isArray(name) ? name[0] : name)); await done; return value; }
    catch (error) { try { tx.abort(); } catch { /* Transaction may already have failed. */ } await done.catch(() => undefined); throw error; }
  }
  async enqueuePending(scope: PendingScope, requestId: string, update: Uint8Array, snapshot?: Uint8Array): Promise<void> {
    await this.transaction(snapshot ? ['pending', 'snapshots'] : 'pending', 'readwrite', async store => {
      const id = key(scope, requestId), old = await request(store.get(id)) as PendingUpdate | undefined;
      if (old && (old.update.length !== update.length || old.update.some((b, i) => b !== update[i]))) throw new Error('Request identity conflict');
      const rows = await request(store.getAll()) as PendingUpdate[];
      const created = Math.max(Date.now(), ...rows.filter(row => scopeKey(row.scope) === scopeKey(scope)).map(row => row.created + 1));
      if (!old) await request(store.put({ scope, requestId, update: update.slice(), created, sent: false }, id));
      if (snapshot) await request(store.transaction.objectStore('snapshots').put({ scope, update: snapshot, updated: Date.now() }, scopeKey(scope)));
    });
  }
  async ackPending(scope: PendingScope, requestId: string): Promise<void> { await this.transaction('pending', 'readwrite', async store => { await request(store.delete(key(scope, requestId))); }); }
  async pending(scope: PendingScope): Promise<PendingUpdate[]> {
    return this.transaction('pending', 'readonly', async store => (await request(store.getAll()) as PendingUpdate[]).filter(item => scopeKey(item.scope) === scopeKey(scope)).sort((a, b) => a.created - b.created));
  }
  // Merge only updates never sent. Once sent, request identity and bytes are immutable.
  async nextBatch(scope: PendingScope, maxBytes: number): Promise<PendingUpdate | undefined> {
    return this.transaction('pending', 'readwrite', async store => {
      const rows = (await request(store.getAll()) as PendingUpdate[]).filter(item => scopeKey(item.scope) === scopeKey(scope)).sort((a, b) => a.created - b.created);
      if (!rows.length) return;
      let first = rows[0];
      if (!first.sent) {
        const selected = [first]; let bytes = first.update;
        for (const row of rows.slice(1)) { if (row.sent) break; const merged = Y.mergeUpdates([bytes, row.update]); if (merged.length > maxBytes) break; bytes = merged; selected.push(row); }
        if (selected.length > 1) {
          first = { ...first, requestId: crypto.randomUUID(), update: bytes };
          for (const row of selected) await request(store.delete(key(scope, row.requestId)));
        }
      }
      first = { ...first, sent: true }; await request(store.put(first, key(scope, first.requestId))); return first;
    });
  }
  async saveSnapshot(scope: PendingScope, update: Uint8Array): Promise<void> { await this.transaction('snapshots', 'readwrite', async store => { await request(store.put({ scope, update, updated: Date.now() }, scopeKey(scope))); }); }
  async snapshots(principalId: string, workspaceId: string, sceneId: string): Promise<{ scope: PendingScope; update: Uint8Array; updated: number }[]> {
    return this.transaction('snapshots', 'readonly', async store => (await request(store.getAll())).filter(row => row.scope.principalId === principalId && row.scope.workspaceId === workspaceId && row.scope.sceneId === sceneId).sort((a, b) => b.updated - a.updated));
  }
  async scopes(principalId: string, workspaceId: string, sceneId: string): Promise<PendingScope[]> {
    const rows = (await Promise.all(['pending', 'snapshots', 'blobs', 'drafts'].map(name => this.transaction(name, 'readonly', store => request(store.getAll()))))).flat();
    const scopes = rows.map(row => row.scope as PendingScope).filter(scope => scope.principalId === principalId && scope.workspaceId === workspaceId && scope.sceneId === sceneId);
    return scopes.filter((scope, i) => scopes.findIndex(other => scopeKey(other) === scopeKey(scope)) === i);
  }
  async saveDraft(scope: PendingScope, content: SceneContent): Promise<void> { await this.transaction('drafts', 'readwrite', async store => { const old = await request(store.get(scopeKey(scope))) as DraftRecord | undefined; await request(store.put({ ...old, scope, content }, scopeKey(scope))); }); }
  async draft(scope: PendingScope): Promise<SceneContent | undefined> { return this.transaction('drafts', 'readonly', async store => (await request(store.get(scopeKey(scope))))?.content); }
  async removeDraft(scope: PendingScope): Promise<void> { await this.transaction('drafts', 'readwrite', async store => { const old = await request(store.get(scopeKey(scope))) as DraftRecord | undefined; if (old?.revision !== undefined) await request(store.put({ scope, revision: old.revision }, scopeKey(scope))); else await request(store.delete(scopeKey(scope))); }); }
  // Reserve callback order before slow Blob conversion, across every mount of this scope.
  async reserveImageDraft(scope: PendingScope, draft: ImageDraft): Promise<number> {
    return this.transaction(['drafts', 'blobs'], 'readwrite', async store => {
      const previous = await request(store.get(scopeKey(scope))) as DraftRecord | undefined;
      const revision = (previous?.revision ?? 0) + 1, latest = { ...draft, revision };
      await request(store.put({ scope, content: latest.desired, imageDraft: latest, revision }, scopeKey(scope)));
      const blobs = store.transaction.objectStore('blobs');
      for (const row of await request(blobs.getAll()) as PendingBlob[]) if (scopeKey(row.scope) === scopeKey(scope)) await request(blobs.put({ ...row, draft: latest }, key(scope, row.id)));
      return revision;
    });
  }
  async putImageBlob(item: PendingBlob, revision: number): Promise<void> {
    await this.transaction(['drafts', 'blobs'], 'readwrite', async store => {
      const latest = await request(store.get(scopeKey(item.scope))) as DraftRecord | undefined;
      if (latest?.revision === undefined || latest.revision < revision) throw new Error('Image recovery revision unavailable');
      // A successful commit retains its revision as a tombstone. Late bytes cannot resurrect it.
      if (!latest.imageDraft) return;
      const blobs = store.transaction.objectStore('blobs'), id = key(item.scope, item.id);
      const existing = await request(blobs.get(id)) as PendingBlob | undefined;
      await request(blobs.put({ ...(existing ?? item), draft: latest.imageDraft }, id));
    });
  }
  async isImageDraftCurrent(scope: PendingScope, revision: number): Promise<boolean> {
    return this.transaction('drafts', 'readonly', async store => {
      const latest = await request(store.get(scopeKey(scope))) as DraftRecord | undefined;
      return latest?.revision === revision && !!latest.imageDraft;
    });
  }
  async completeImageDraft(scope: PendingScope, revision: number, ids: string[]): Promise<boolean> {
    return this.transaction(['drafts', 'blobs'], 'readwrite', async store => {
      const latest = await request(store.get(scopeKey(scope))) as DraftRecord | undefined;
      if (latest?.revision !== revision) return false;
      await request(store.put({ scope, revision }, scopeKey(scope)));
      const blobs = store.transaction.objectStore('blobs');
      for (const id of ids) await request(blobs.delete(key(scope, id)));
      return true;
    });
  }
  async putBlob(item: PendingBlob): Promise<void> { await this.transaction('blobs', 'readwrite', async store => { await request(store.put(item, key(item.scope, item.id))); }); }
  async blobs(scope: PendingScope): Promise<PendingBlob[]> { return this.transaction('blobs', 'readonly', async store => (await request(store.getAll()) as PendingBlob[]).filter(row => scopeKey(row.scope) === scopeKey(scope))); }
  async removeBlob(scope: PendingScope, id: string): Promise<void> { await this.transaction('blobs', 'readwrite', async store => { await request(store.delete(key(scope, id))); }); }
}
export const offlineStore = new OfflineStore();
export const enqueuePending = (scope: PendingScope, requestId: string, update: Uint8Array) => offlineStore.enqueuePending(scope, requestId, update);
export const ackPending = (scope: PendingScope, requestId: string) => offlineStore.ackPending(scope, requestId);
