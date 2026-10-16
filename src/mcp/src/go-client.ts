import { timingSafeEqual } from 'node:crypto';
import { AccessError, apiBase } from './auth.js';

export type AgentIdentity = { principalId: string; keyId: string };

export class GoClient {
  private readonly base: URL;
  private readonly publicBase?: URL;
  private verifiedIdentity?: AgentIdentity;

  constructor(url: string, private readonly key: string, private readonly correlationId?: string, publicURL?: string) {
    this.base = apiBase(url);
    this.publicBase = publicURL === undefined ? undefined : apiBase(publicURL);
  }

  get identity(): AgentIdentity | undefined {
    return this.verifiedIdentity;
  }

  matchesCredential(key: string): boolean {
    const candidate = Buffer.from(key);
    const original = Buffer.from(this.key);
    return candidate.length === original.length && timingSafeEqual(candidate, original);
  }

  forCall(correlationId: string): GoClient {
    const client = new GoClient(this.base.origin, this.key, correlationId, this.publicBase?.origin);
    client.verifiedIdentity = this.verifiedIdentity;
    return client;
  }

  private frontendBase():URL {
    if(!this.publicBase)throw new Error('WHITEBOARD_PUBLIC_URL required for public links and remote exports');
    return this.publicBase;
  }

  sceneURL(ref:{workspaceId:string;sceneId:string}):string {return new URL(`/w/${ref.workspaceId}/scenes/${ref.sceneId}`,this.frontendBase()).href;}

  artifactURL(id:string):string {return new URL(`/mcp/artifacts/${id}`,this.frontendBase()).href;}
  async remoteExport(input:unknown):Promise<Record<string,unknown>> {return this.requestJSON('/mcp/exports','POST',input,256*1024,true);}
  async exportRequest(path:string):Promise<Record<string,unknown>> {
    if(!/\/export-(projection$|assets\/)/.test(path))throw new Error('Invalid export path');
    return this.requestJSON(path,'GET',undefined,26*1024*1024);
  }
  async request(path: string, method = 'GET', body?: unknown): Promise<Record<string, unknown>> {return this.requestJSON(path,method,body,256*1024);}
  private async requestJSON(path:string,method:string,body:unknown,maxBytes:number,remoteExport=false):Promise<Record<string,unknown>> {
    if (!(path.startsWith('/api/v1/agent/') || remoteExport&&path==='/mcp/exports')) throw new Error('Invalid agent API path');
    const response = await fetch(new URL(path, remoteExport ? this.frontendBase() : this.base), {
      method,
      headers: {
        Authorization: `Bearer ${this.key}`,
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(this.correlationId ? { 'X-Agent-Call-ID': this.correlationId } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      redirect: 'error',
      signal: AbortSignal.timeout(remoteExport?30000:10000),
    });
    if (!response.ok) {
      let upstreamCode:string|undefined;
      if(response.body){
        const reader=response.body.getReader();let size=0;const chunks:Uint8Array[]=[];
        try{for(;;){const part=await reader.read();if(part.done)break;size+=part.value.length;if(size>4096)break;chunks.push(part.value);}if(size<=4096){try{upstreamCode=JSON.parse(Buffer.concat(chunks).toString('utf8')).code;}catch{}}}finally{await reader.cancel();}
      }
      const safeCodes=new Set(['CONFLICT','REQUEST_ID_REUSED','STALE_AUTHORITY','RATE_LIMITED','AUDIT_COMPLETION_UNCERTAIN','INVALID_UPDATE','COLLAB_DISABLED','PROMOTION_REQUIRED','SNAPSHOT_HISTORY_LIMIT']);
      const codes: Record<number, string> = {
        401: 'UNAUTHENTICATED', 403: 'ACCESS_DENIED', 404: 'NOT_FOUND',
        400: 'INVALID_INPUT', 409: 'CONFLICT', 413: 'PROJECTION_TOO_LARGE', 429: 'RATE_LIMITED', 503: 'UPSTREAM_UNAVAILABLE',
      };
      throw new AccessError(response.status, upstreamCode&&safeCodes.has(upstreamCode)?upstreamCode:codes[response.status] || 'AGENT_API_ERROR');
    }
    if (!response.body) throw new AccessError(502, 'INVALID_API_RESPONSE');
    const reader = response.body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    try {
      for (;;) {
        const part = await reader.read();
        if (part.done) break;
        size += part.value.byteLength;
        if (size > maxBytes) throw new AccessError(413, 'PROJECTION_TOO_LARGE');
        chunks.push(part.value);
      }
    } finally {
      await reader.cancel();
    }
    let result: unknown;
    try {
      result = JSON.parse(Buffer.concat(chunks).toString('utf8'));
    } catch {
      throw new AccessError(502, 'INVALID_API_RESPONSE');
    }
    if (!result || typeof result !== 'object' || Array.isArray(result)) {
      throw new AccessError(502, 'INVALID_API_RESPONSE');
    }
    return result as Record<string, unknown>;
  }

  async authenticate(): Promise<AgentIdentity> {
    const result = await this.request('/api/v1/agent/auth');
    if (typeof result.principalId !== 'string' || typeof result.keyId !== 'string') {
      throw new AccessError(502, 'INVALID_API_RESPONSE');
    }
    this.verifiedIdentity = { principalId: result.principalId, keyId: result.keyId };
    return this.verifiedIdentity;
  }

  async roomCredential(workspaceId: string, sceneId: string): Promise<Record<string, unknown>> {
    return this.request(
      `/api/v1/agent/workspaces/${encodeURIComponent(workspaceId)}/scenes/${encodeURIComponent(sceneId)}/room-token`,
      'POST',
    );
  }
}
