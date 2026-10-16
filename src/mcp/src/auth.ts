import type { IncomingMessage } from 'node:http';

export class AccessError extends Error {
  constructor(public readonly status: number, public readonly code: string, public readonly correlationId?: string) {
    super(code);
  }
}

export function bearer(req: IncomingMessage): string {
  const raw = req.headers.authorization;
  if (!raw?.startsWith('Bearer ') || raw.length > 512 || raw.slice(7).includes(' ') || raw.length === 7) {
    throw new AccessError(401, 'UNAUTHENTICATED');
  }
  return raw.slice(7);
}

export function checkOrigin(req: IncomingMessage, allowedOrigins: readonly string[]): void {
  const origin = req.headers.origin;
  if (origin && !allowedOrigins.includes(origin)) throw new AccessError(403, 'ORIGIN_DENIED');
}

export function apiBase(raw: string): URL {
  const url = new URL(raw);
  if (url.username || url.password || url.search || url.hash || url.pathname !== '/') {
    throw new Error('WHITEBOARD_API_URL must be an origin');
  }
  if (url.protocol !== 'https:' && !(url.protocol === 'http:' && ['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname))) {
    throw new Error('Remote API requires HTTPS');
  }
  return url;
}
