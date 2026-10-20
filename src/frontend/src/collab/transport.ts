import * as Y from 'yjs';
import { encodeRecoveryBytes, encodeRecoveryBlob } from './recovery';
import { decodeFrame, encodeFrame, MAX_FRAME_BYTES, parseFeatureChanged, parsePresentationAdmitted, type SceneContent, type SceneRef } from '@collab/contracts';
import { ExcalidrawAdapter, REMOTE_ORIGIN, REPLAY_ORIGIN } from './excalidraw-adapter';
import { type OfflineStore, type PendingScope, scopeKey } from './offline-store';
export type RoomGrant = { token: string; expiresAt: string };
export type WireSocket = Pick<WebSocket, 'readyState' | 'send' | 'close' | 'addEventListener' | 'binaryType'>;
export type CollabStatus = 'connecting' | 'synced' | 'pending' | 'offline' | 'readonly' | 'revoked' | 'degraded' | 'legacy';
export type TransportState = { status: CollabStatus; canEdit: boolean; pending: number; recovery: boolean; error?: string; connectionId?: string; sequence?: string };
export type Control = { type: string; requestId: string; payload: Record<string, unknown> };
export type TransportOptions = { sceneRef: SceneRef; principalId: string; adapter: ExcalidrawAdapter; store: OfflineStore; grant: () => Promise<RoomGrant>; url: string; authType?:'auth'|'presentation_auth'; socket?: (url: string) => WireSocket; onAccessLost?: () => void; onLegacy?: (code: string) => Promise<boolean>; onPresentationAdmitted?:(grant:{sessionId:string;token:string;expiresAt:number})=>void };
const MAX_HISTORY_BYTES = 64 * 1024 * 1024;
const SEND_INTERVAL_MS = 600;
const REFRESH_EARLY_MS = 60000;
const MAX_RECONNECT_MS = 30000;
const decode64 = (text: string) => Uint8Array.from(atob(text), c => c.charCodeAt(0));
export class CollabTransport {
  state: TransportState = { status: 'connecting', canEdit: false, pending: 0, recovery: false };
  scope?: PendingScope;
  private socket?: WireSocket;
  private generation = 0;
  private stopped = true;
  private admitted = false;
  private synced = false;
  private access = 'view';
  private attempts = 0;
  private lastAwareness = 0;
  private awarenessTimer?: ReturnType<typeof setTimeout>;
  private pendingAwareness?: {cursor?: {x:number;y:number}; selectedIds?:string[]};
  private clearAwareness() { clearTimeout(this.awarenessTimer); this.awarenessTimer=undefined; this.pendingAwareness=undefined; this.lastAwareness=0; }
  private ephemeralRequests = new Map<string, number>();
  private retryTimer?: ReturnType<typeof setTimeout>;
  private refreshTimer?: ReturnType<typeof setTimeout>;
  private flushTimer?: ReturnType<typeof setTimeout>;
  private ackTimer?: ReturnType<typeof setTimeout>;
  private sent = new Set<string>();
  private flushing?: { generation: number };
  private listeners = new Set<() => void>();
  private controls = new Set<(control: Control) => void>();
  private chain: Promise<void> = Promise.resolve();
  private durable: Promise<void> = Promise.resolve();
  private durableError?: unknown;
  private incoming: Uint8Array[] = [];
  private incomingBytes = 0;
  private chunks?: { requestId: string; count: number; total: number; chunks: Uint8Array[] };
  private authoritative?:ExcalidrawAdapter;
  private observed?:SceneContent;
  constructor(readonly options: TransportOptions) {}
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  subscribeControl = (listener: (control: Control) => void) => { this.controls.add(listener); return () => { this.controls.delete(listener); }; };
  getSnapshot = () => this.state;
  isActive = () => !this.stopped;
  private set(update: Partial<TransportState>) { if (update.status && ['offline','revoked','degraded','legacy'].includes(update.status)) { update = {...update, connectionId: undefined}; this.ephemeralRequests.clear(); this.clearAwareness(); } this.state = { ...this.state, ...update }; this.listeners.forEach(listener => listener()); }
  private fail(error: unknown) { this.synced = false; this.set({ status: 'degraded', canEdit: false, recovery: true, error: error instanceof Error ? error.message : 'Collaboration unavailable' }); }
  private offline = () => { this.socket?.close(); this.synced = false; this.set({ status: 'offline', canEdit: this.admitted && this.access === 'edit' }); };
  private online = () => { if (!this.stopped) this.retry(); };
  async start() {
    window.addEventListener('offline', this.offline); window.addEventListener('online', this.online);
    this.stopped = false; this.options.adapter.doc.on('update', this.localUpdate);
    try {
      const cached = await this.options.store.snapshots(this.options.principalId, this.options.sceneRef.workspaceId, this.options.sceneRef.sceneId);
      if (this.stopped) return;
      if (cached[0]) { this.scope = cached[0].scope; Y.applyUpdate(this.options.adapter.doc, new Uint8Array(cached[0].update), REPLAY_ORIGIN); this.set({ recovery: true }); }
      await this.connect();
    } catch (error) { this.fail(error); }
  }
  private async connect() {
    if (this.stopped) return;
    if (!navigator.onLine) { this.set({ status: 'offline', canEdit: this.admitted && this.access === 'edit' }); return; }
    const generation = ++this.generation; this.flushing = undefined; this.synced = false; this.incoming = []; this.incomingBytes = 0; this.sent.clear(); clearTimeout(this.ackTimer);
    this.clearAwareness(); this.ephemeralRequests.clear(); this.set({ status: 'connecting', canEdit: false, connectionId: undefined });
    try {
      const grant = await this.options.grant(); if (this.stopped || generation !== this.generation) return;
      if (!Number.isFinite(Date.parse(grant.expiresAt))) throw new Error('Invalid room expiry');
      const socket = (this.options.socket ?? (url => new WebSocket(url)))(this.options.url);
      this.socket = socket; socket.binaryType = 'arraybuffer';
      socket.addEventListener('open', () => { if (this.current(generation)) { socket.send(JSON.stringify({ type: this.options.authType??'auth', requestId: crypto.randomUUID(), payload: { token: grant.token } })); if(this.options.authType!=='presentation_auth')this.scheduleRefresh(grant, generation); } });
      socket.addEventListener('message', event => {
        this.chain = this.chain.then(async () => { if (this.current(generation)) await this.receive((event as MessageEvent).data, generation); }).catch(error => { if (this.current(generation)) this.fail(error); });
      });
      socket.addEventListener('close', () => {
        if (!this.current(generation)) return;
        clearTimeout(this.refreshTimer); clearTimeout(this.ackTimer); this.synced = false;
        if (!['revoked', 'degraded', 'legacy'].includes(this.state.status)) { this.set({ status: 'offline', canEdit: this.admitted && this.access === 'edit' }); this.reconnect(); }
      });
      socket.addEventListener('error', () => { if (this.current(generation)) socket.close(); });
    } catch (error) { if (this.current(generation)) await this.authorityError(error); }
  }
  private current(generation: number) { return !this.stopped && this.generation === generation; }
  private reconnect() { clearTimeout(this.retryTimer); const delay = Math.min(MAX_RECONNECT_MS, 500 * 2 ** Math.min(this.attempts++, 6)) * (0.75 + Math.random() * 0.5); this.retryTimer = setTimeout(() => void this.connect(), delay); }
  private scheduleRefresh(grant: RoomGrant, generation: number) {
    clearTimeout(this.refreshTimer);
    const expiry = Date.parse(grant.expiresAt); if (!Number.isFinite(expiry)) throw new Error('Invalid room expiry');
    this.refreshTimer = setTimeout(() => { void this.options.grant().then(next => {
      if (!this.current(generation) || this.socket?.readyState !== 1) return;
      this.socket.send(JSON.stringify({ type: 'refresh', requestId: crypto.randomUUID(), payload: { token: next.token } })); this.scheduleRefresh(next, generation);
    }).catch(error => { if (this.current(generation)) void this.authorityError(error); }); }, Math.max(1000, expiry - Date.now() - REFRESH_EARLY_MS));
  }
  private async authorityError(error: unknown) {
    const response = (error as { response?: { status?: number; data?: { code?: string } } }).response;
    const code = response?.data?.code;
    if (await this.legacy(code)) return;
    if ([401, 403, 404, 410].includes(response?.status ?? 0)) { this.options.onAccessLost?.(); this.access = 'view'; this.synced = false; this.set({ status: 'revoked', canEdit: false, recovery: true, error: 'Access ended. Pending work is retained for recovery.' }); }
    else { this.set({ status: 'offline', canEdit: this.admitted && this.access === 'edit', error: 'Connection unavailable. Pending work stays on this device.' }); this.reconnect(); }
  }
  private async legacy(code?: string) {
    if (!this.admitted && code && ['COLLAB_DISABLED', 'PROMOTION_REQUIRED'].includes(code) && await this.options.onLegacy?.(code)) {
      this.set({ status: 'legacy', canEdit: false }); this.socket?.close(); return true;
    }
    return false;
  }
  private async receive(data: unknown, generation: number) {
    if (typeof data !== 'string') {
      if (!this.scope) throw new Error('Update before admission');
      const frame = decodeFrame(new Uint8Array(data as ArrayBuffer));
      if (frame.channel === 3) {
        if (!this.synced) { this.incomingBytes += frame.payload.length; if (this.incomingBytes > MAX_HISTORY_BYTES) throw new Error('Sync buffer limit; reconnect required'); this.incoming.push(frame.payload); }
        else { this.authoritative?.applyRemote(frame.payload);this.options.adapter.applyRemote(frame.payload); await this.cache(); }
      }
      return;
    }
    const control = JSON.parse(data) as Control, p = control.payload;
    if (!p || typeof control.type !== 'string') throw new Error('Invalid control');
    if (control.type === 'ready') {
      if (p.workspaceId !== this.options.sceneRef.workspaceId || p.sceneId !== this.options.sceneRef.sceneId || p.principalId !== this.options.principalId || typeof p.documentId !== 'string' || typeof p.sequence !== 'string' || !/^\d+$/.test(p.sequence) || !['view', 'edit'].includes(String(p.access))) throw new Error('Room scope mismatch');
      const scope = { ...this.options.sceneRef, principalId: this.options.principalId, documentId: p.documentId };
      if (this.scope && scopeKey(this.scope) !== scopeKey(scope)) { this.fail(new Error('Document generation changed. Export retained work before continuing.')); this.socket?.close(); return; }
      this.scope = scope; this.admitted = true; this.synced = false; this.chunks = undefined; this.access = String(p.access);
      this.set({ connectionId: typeof p.connectionId === 'string' ? p.connectionId : undefined, canEdit: false, sequence: p.sequence, status: p.access === 'edit' ? 'connecting' : 'readonly' }); return;
    }
    if (control.type === 'sync_chunk') {
      if (!this.scope || p.documentId !== this.scope.documentId) throw new Error('Snapshot generation mismatch');
      const { index, count, totalBytes } = p;
      if (!Number.isInteger(index) || !Number.isInteger(count) || !Number.isInteger(totalBytes) || Number(count) < 1 || Number(count) > 128 || Number(index) < 0 || Number(index) >= Number(count) || Number(totalBytes) > MAX_HISTORY_BYTES || Number(totalBytes) < 0 || typeof p.data !== 'string' || p.data.length > 700000) throw new Error('Invalid sync chunk');
      this.chunks ??= { requestId: control.requestId, count: Number(count), total: Number(totalBytes), chunks: [] };
      const group = this.chunks;
      if (group.requestId !== control.requestId || group.count !== count || group.total !== totalBytes || group.chunks.length !== index) throw new Error('Out-of-order sync chunk');
      group.chunks.push(decode64(p.data));
      if (group.chunks.length === group.count) {
        const size = group.chunks.reduce((sum, chunk) => sum + chunk.length, 0); if (size !== group.total) throw new Error('Snapshot length mismatch');
        const bytes = new Uint8Array(size); let offset = 0; for (const chunk of group.chunks) { bytes.set(chunk, offset); offset += chunk.length; }
        this.options.adapter.applyRemote(bytes);const incoming=this.incoming;
        for (const update of incoming) this.options.adapter.applyRemote(update); this.incoming = []; this.incomingBytes = 0;
        this.authoritative?.doc.destroy();this.authoritative=new ExcalidrawAdapter();this.authoritative.applyRemote(bytes);for(const update of incoming)this.authoritative.applyRemote(update);
        this.observed=this.options.adapter.content();
        await this.whenDurable();if(!this.current(generation))return;
        let pending = await this.options.store.pending(this.scope); if (!this.current(generation)) return;
        for (const row of pending) Y.applyUpdate(this.options.adapter.doc, new Uint8Array(row.update), REPLAY_ORIGIN);
        const safe=pending.filter(row=>!row.sent&&row.intent),unsafe=pending.filter(row=>row.sent||!row.intent);
        if(safe.length){
          const rebaseBase=this.authoritative.content(),causal:Uint8Array[]=[];for(const row of safe){const update=this.authoritative.reapplyIntent(row.intent!.base,row.intent!.desired);causal.push(update);this.options.adapter.applyRemote(update);}
          const replacement={scope:this.scope,requestId:crypto.randomUUID(),update:Y.mergeUpdates(causal),created:safe[0].created,intent:{base:rebaseBase,desired:this.authoritative.content()}};
          if(!await this.options.store.replaceUnsent(this.scope,safe,replacement))throw new Error('Pending edits changed during causal replay; reconnect required');
          pending=await this.options.store.pending(this.scope);const expected=new Set([...unsafe.map(row=>row.requestId),replacement.requestId]);if(pending.length!==expected.size||pending.some(row=>!expected.has(row.requestId)))throw new Error('Pending edits changed during causal replay; reconnect required');
        }
        if(unsafe.length)throw new Error('Ambiguous or legacy pending edits require recovery before reconnect');
        this.synced = true; this.attempts = 0; this.chunks = undefined;
        await this.cache(); if (!this.current(generation)) return;
        this.set({ status: this.access === 'edit' ? pending.length ? 'pending' : 'synced' : 'readonly', canEdit: this.access === 'edit', pending: pending.length, recovery: pending.length > 0, error: undefined });
        await this.flush();
      }
      return;
    }
    if (control.type === 'ack') {
      if (!this.scope || !this.synced || p.requestId !== control.requestId || !this.sent.has(control.requestId) || typeof p.sequence !== 'string' || !/^\d+$/.test(p.sequence)) return;
      const acknowledged=(await this.options.store.pending(this.scope)).find(row=>row.requestId===control.requestId);if(acknowledged)this.authoritative?.applyRemote(new Uint8Array(acknowledged.update));
      await this.options.store.ackPending(this.scope, control.requestId); this.sent.delete(control.requestId); clearTimeout(this.ackTimer); this.noteCommittedSequence(p.sequence);
      await this.updatePending(); this.scheduleFlush(); return;
    }
    const ephemeral = this.ephemeralRequests.delete(control.requestId);
    if (control.type === 'error') {
      if (await this.legacy(String(p.code))) return;
      if (this.options.authType==='presentation_auth'&&!this.admitted) { this.controls.forEach(listener=>listener(control));this.options.onAccessLost?.();this.access='view';this.synced=false;this.set({status:'revoked',canEdit:false,recovery:false,error:String(p.message??p.code)});this.socket?.close(); }
      else if (['FORBIDDEN', 'UNAUTHENTICATED'].includes(String(p.code))) { this.options.onAccessLost?.(); this.access = 'view'; this.synced = false; this.set({ status: 'revoked', canEdit: false, recovery: true, error: 'Access ended. Export pending work for recovery.' }); }
      else if (ephemeral || /^(VOICE_|INVALID_FOLLOW$|INVALID_PRESENCE$|PRESENCE_RATE_LIMITED$)/.test(String(p.code))) { this.controls.forEach(listener => listener(control)); }
      else if (p.retryable === true || p.code === 'RATE_LIMITED') { this.synced = false; this.set({ status: 'offline', error: String(p.code) }); this.socket?.close(); }
      else { this.fail(new Error(String(p.code))); }
      return;
    }
    if (control.type === 'committed') {
      if (typeof p.sequence !== 'string' || !/^\d+$/.test(p.sequence)) throw new Error('Invalid committed sequence');
      this.noteCommittedSequence(p.sequence); this.controls.forEach(listener => listener(control)); return;
    }
    if (control.type === 'resync_required') { this.access = 'view'; this.synced = false; this.set({ canEdit: false, connectionId: undefined, status: 'connecting' }); return; }
    if (control.type === 'feature_changed') { control.payload = parseFeatureChanged(p); this.controls.forEach(listener => listener(control)); return; }
    if (control.type === 'presentation_admitted') { const admitted=parsePresentationAdmitted(p);this.options.onPresentationAdmitted?.(admitted);control.payload=admitted;this.controls.forEach(listener=>listener(control));return; }
    this.controls.forEach(listener => listener(control));
  }
  private localUpdate = (update: Uint8Array, origin: unknown) => {
    const desired=this.options.adapter.content(),base=this.observed??desired;this.observed=desired;
    if (origin === REMOTE_ORIGIN || origin === REPLAY_ORIGIN || this.stopped) return;
    const scope = this.scope;
    if (!scope || !this.state.canEdit) { this.fail(new Error('Edit without current admission')); return; }
    const id = crypto.randomUUID(), snapshot = Y.encodeStateAsUpdate(this.options.adapter.doc); this.set({ status: this.synced ? 'pending' : 'offline', recovery: true });
    this.durable = this.durable.then(async () => {
      const intent=this.authoritative?{base,desired}:undefined;
      await this.options.store.enqueuePending(scope, id, update, snapshot,intent); if (snapshot.length > MAX_HISTORY_BYTES) throw new Error('SNAPSHOT_HISTORY_LIMIT: export recovery'); await this.updatePending(); this.scheduleFlush();
    }).catch(error => { this.durableError = error; this.fail(error); });
  };
  private async cache() { if (this.scope) { const bytes = Y.encodeStateAsUpdate(this.options.adapter.doc); if (bytes.length > MAX_HISTORY_BYTES) throw new Error('SNAPSHOT_HISTORY_LIMIT: export recovery'); await this.options.store.saveSnapshot(this.scope, bytes); } }
  private async updatePending() { if (!this.scope) return; const count = (await this.options.store.pending(this.scope)).length; this.set({ pending: count, recovery: count > 0, ...(this.synced && this.access === 'edit' ? { status: count ? 'pending' : 'synced' } : {}) }); }
  private scheduleFlush() { if (this.flushTimer) return; this.flushTimer = setTimeout(() => { this.flushTimer = undefined; void this.flush().catch(error => this.fail(error)); }, SEND_INTERVAL_MS); }
  async whenDurable() { await this.durable; if (this.durableError) throw this.durableError; }
  noteCommittedSequence(sequence: string) { if (!/^\d+$/.test(sequence)) throw new Error('Invalid committed sequence'); if (this.state.sequence !== undefined && BigInt(sequence) <= BigInt(this.state.sequence)) return; this.set({ sequence }); }
  async flush() {
    const generation = this.generation;
    if (this.stopped) return;
    if (this.flushing?.generation === generation) { await this.whenDurable(); return; }
    const reservation = { generation }; this.flushing = reservation;
    try {
      await this.whenDurable();
      const scope = this.scope;
      if (!this.current(generation) || !scope || !this.synced || this.access !== 'edit' || this.socket?.readyState !== 1 || this.sent.size) return;
      const row = await this.options.store.nextBatch(scope, MAX_FRAME_BYTES - 256);
      if (!row || this.flushing !== reservation || !this.current(generation) || !this.scope || scopeKey(scope) !== scopeKey(this.scope) || !this.synced || this.access !== 'edit' || this.socket?.readyState !== 1 || this.sent.size) return;
      const frame = encodeFrame({ channel: 3, updateId: row.requestId, payload: new Uint8Array(row.update) });
      this.sent.add(row.requestId); this.socket.send(new Uint8Array(frame));
      clearTimeout(this.ackTimer);
      this.ackTimer = setTimeout(() => { if (this.current(generation) && this.sent.has(row.requestId)) this.socket?.close(); }, 15000);
    } finally { if (this.flushing === reservation) this.flushing = undefined; }
  }
  sendAwareness(payload: { cursor?: { x: number; y: number }; selectedIds?: string[] }) {
    if (!this.synced || this.socket?.readyState !== 1) return;
    const clean = { ...(payload.cursor ? { cursor: { x: payload.cursor.x, y: payload.cursor.y } } : {}), ...(payload.selectedIds ? { selectedIds: payload.selectedIds.slice(0, 256) } : {}) };
    this.pendingAwareness={...this.pendingAwareness,...clean};
    if(this.awarenessTimer)return;
    const send=()=>{
      this.awarenessTimer=undefined; const value=this.pendingAwareness; this.pendingAwareness=undefined;
      if(!value||!this.synced||this.socket?.readyState!==1)return;
      this.lastAwareness=Date.now();
      this.socket.send(new Uint8Array(encodeFrame({channel:2,updateId:crypto.randomUUID(),payload:new Uint8Array(new TextEncoder().encode(JSON.stringify(value)))})));
    };
    const delay=50-(Date.now()-this.lastAwareness);
    if(delay>0)this.awarenessTimer=setTimeout(send,delay);else send();
  }
  sendControl(type: string, payload: Record<string, unknown>): string | undefined {
    if ((!this.synced && type !== 'presentation_leave') || this.socket?.readyState !== 1) return;
    const now = Date.now(); for (const [id, time] of this.ephemeralRequests) if (now - time > 30000) this.ephemeralRequests.delete(id);
    while (this.ephemeralRequests.size >= 256) this.ephemeralRequests.delete(this.ephemeralRequests.keys().next().value!);
    const requestId = crypto.randomUUID(); this.ephemeralRequests.set(requestId, now);
    this.socket.send(JSON.stringify({ type, requestId, payload })); return requestId;
  }
  retry = () => { if (!this.stopped) { this.socket?.close(); void this.connect(); } };
  async exportRecovery() {
    await this.durable;
    const snapshots = await this.options.store.snapshots(this.options.principalId, this.options.sceneRef.workspaceId, this.options.sceneRef.sceneId);
    const scopes = await this.options.store.scopes(this.options.principalId, this.options.sceneRef.workspaceId, this.options.sceneRef.sceneId);
    if (this.scope && !scopes.some(scope => scopeKey(scope) === scopeKey(this.scope!))) scopes.push(this.scope);
    return { format: 'devtool-collab-recovery-v1', encoding: 'base64', principalId: this.options.principalId, sceneRef: this.options.sceneRef, currentBase64: encodeRecoveryBytes(Y.encodeStateAsUpdate(this.options.adapter.doc)), scopes: await Promise.all(scopes.map(async scope => ({
      scope,
      snapshotBase64: snapshots.find(item => scopeKey(item.scope) === scopeKey(scope)) ? encodeRecoveryBytes(new Uint8Array(snapshots.find(item => scopeKey(item.scope) === scopeKey(scope))!.update)) : undefined,
      pending: (await this.options.store.pending(scope)).map(({ update, ...metadata }) => ({ ...metadata, updateBase64: encodeRecoveryBytes(new Uint8Array(update)) })),
      draft: await this.options.store.draft(scope),
      blobs: await Promise.all((await this.options.store.blobs(scope)).map(async ({ blob, ...metadata }) => ({ ...metadata, size: blob.size, blobBase64: await encodeRecoveryBlob(blob) }))),
    }))) };
  }
  stop() { window.removeEventListener('offline', this.offline); window.removeEventListener('online', this.online); this.stopped = true; this.generation++; this.flushing = undefined; this.synced = false; this.admitted = false; this.access = 'view'; this.set({ status: 'offline', canEdit: false }); this.options.adapter.doc.off('update', this.localUpdate);this.authoritative?.doc.destroy();this.authoritative=undefined;this.observed=undefined; clearTimeout(this.retryTimer); clearTimeout(this.refreshTimer); clearTimeout(this.flushTimer); clearTimeout(this.ackTimer); this.socket?.close(); }
}
