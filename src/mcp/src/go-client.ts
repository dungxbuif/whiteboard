import { timingSafeEqual } from 'node:crypto';
import { AccessError, apiBase } from './auth.js';

export type AgentIdentity = { principalId: string; keyId: string };

export class GoClient {
  private readonly base: URL;
  private verifiedIdentity?: AgentIdentity;

  constructor(url: string, private readonly key: string, private readonly correlationId?: string) {
    this.base = apiBase(url);
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
    const client = new GoClient(this.base.origin, this.key, correlationId);
    client.verifiedIdentity = this.verifiedIdentity;
    return client;
  }

  async request(path: string, method = 'GET', body?: unknown): Promise<Record<string, unknown>> {
    if (!path.startsWith('/api/v1/agent/')) throw new Error('Invalid agent API path');
    const response = await fetch(new URL(path, this.base), {
      method,
      headers: {
        Authorization: `Bearer ${this.key}`,
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(this.correlationId ? { 'X-Agent-Call-ID': this.correlationId } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      redirect: 'error',
      signal: AbortSignal.timeout(10000),
    });
    if (!response.ok) {
      await response.body?.cancel();
      const codes: Record<number, string> = {
        401: 'UNAUTHENTICATED', 403: 'ACCESS_DENIED', 404: 'NOT_FOUND',
        413: 'PROJECTION_TOO_LARGE',
      };
      throw new AccessError(response.status, codes[response.status] || 'AGENT_API_ERROR');
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
        if (size > 256 * 1024) throw new AccessError(413, 'PROJECTION_TOO_LARGE');
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
