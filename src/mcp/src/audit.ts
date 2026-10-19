import {
  closeSync, constants, existsSync, fchmodSync, fstatSync, fsyncSync,
  mkdirSync, openSync, renameSync, unlinkSync, writeSync,
} from 'node:fs';
import { dirname, isAbsolute } from 'node:path';
import { randomUUID } from 'node:crypto';
import { ZodError } from 'zod';
import { AccessError } from './auth.js';
import type { GoClient } from './go-client.js';

const scopes: Record<string, string> = {
  list_workspaces: 'workspace:read', list_scenes: 'scene:read', get_scene: 'scene:read',
  create_scene: 'scene:create', generate_diagram: 'scene:write', patch_elements: 'scene:write',
  delete_elements: 'scene:write', get_scene_activity: 'scene:read', export_scene: 'scene:export',
  list_diagram_types: '', get_diagram_template: '', validate_diagram: '', compile_mermaid: '',
};
const failureClasses = new Set([
  'RENDER_LIMIT','RENDER_TIMEOUT','RENDER_BUSY','RENDER_FAILED','INVALID_ASSET','MISSING_ASSET','EXPORT_CAPACITY','ARTIFACT_NOT_FOUND','INVALID_FORMAT','INVALID_PROJECTION','UNSUPPORTED_NATIVE_DIAGRAM',
  'INVALID_INPUT', 'PROJECTION_TOO_LARGE', 'UNAUTHENTICATED', 'ACCESS_DENIED', 'NOT_FOUND',
  'UNKNOWN_TOOL', 'AGENT_API_ERROR', 'INVALID_API_RESPONSE', 'UPSTREAM_UNAVAILABLE',
  'CONFLICT', 'REQUEST_ID_REUSED', 'STALE_AUTHORITY', 'RATE_LIMITED', 'UNSUPPORTED_NATIVE_DIAGRAM', 'AUDIT_COMPLETION_UNCERTAIN', 'INVALID_UPDATE', 'COLLAB_DISABLED', 'PROMOTION_REQUIRED', 'SNAPSHOT_HISTORY_LIMIT',
  'SESSION_NOT_FOUND', 'SESSION_PRINCIPAL_MISMATCH', 'INITIALIZE_REQUIRED', 'SESSION_LIMIT', 'INVALID_SESSION',
]);
function uuid(value: unknown): string | null {
  return typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value)
    ? value : null;
}
export type AdapterAuditEvent = {
  correlationId: string; principalId: string | null; keyId: string | null;
  workspaceId: string | null; sceneId: string | null; tool: string; scope: string;
  outcome: string; latencyMs: number; failureClass: string;
  provenance: 'adapter_observed'; identitySource: 'prior_verified' | 'unknown'; timestamp: string;
};

// One process owns each configured path. Rotation retains at most 3 MiB by
// default (active + two archives). Collect these files before retention expires.
export class AuditJournal {
  private readonly maxBytes: number;
  private readonly archives: number;

  constructor(readonly path: string, options: { maxBytes?: number; archives?: number } = {}) {
    if (!isAbsolute(path)) throw new Error('Audit path must be absolute');
    this.maxBytes = options.maxBytes ?? 1024 * 1024;
    this.archives = options.archives ?? 2;
    if (!Number.isInteger(this.maxBytes) || this.maxBytes < 1024 || this.maxBytes > 16 * 1024 * 1024 ||
        !Number.isInteger(this.archives) || this.archives < 1 || this.archives > 10) {
      throw new Error('Invalid audit retention bounds');
    }
    mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
    const fd = this.open();
    try { fsyncSync(fd); } finally { closeSync(fd); }
    this.syncDirectory();
  }

  private open(): number {
    const fd = openSync(this.path, constants.O_WRONLY | constants.O_APPEND | constants.O_CREAT | constants.O_NOFOLLOW, 0o600);
    if (!fstatSync(fd).isFile()) { closeSync(fd); throw new Error('Audit path must be a regular file'); }
    fchmodSync(fd, 0o600);
    return fd;
  }

  private syncDirectory(): void {
    const fd = openSync(dirname(this.path), constants.O_RDONLY);
    try { fsyncSync(fd); } finally { closeSync(fd); }
  }

  private rotate(): void {
    const oldest = `${this.path}.${this.archives}`;
    if (existsSync(oldest)) unlinkSync(oldest);
    for (let i = this.archives - 1; i >= 1; i--) {
      if (existsSync(`${this.path}.${i}`)) renameSync(`${this.path}.${i}`, `${this.path}.${i + 1}`);
    }
    renameSync(this.path, `${this.path}.1`);
    this.syncDirectory();
  }

  record(event: AdapterAuditEvent): void {
    const bytes = Buffer.from(JSON.stringify(event) + '\n');
    if (bytes.byteLength > this.maxBytes) throw new Error('Audit event exceeds record bound');
    let fd = this.open();
    try {
      if (fstatSync(fd).size + bytes.byteLength > this.maxBytes) {
        closeSync(fd);
        fd = -1;
        this.rotate();
        fd = this.open();
        this.syncDirectory();
      }
      let written = 0;
      while (written < bytes.byteLength) written += writeSync(fd, bytes, written);
      fsyncSync(fd);
    } finally {
      if (fd !== -1) closeSync(fd);
    }
  }
}

function classified(error: unknown): { code: string; status: number; outcome: string } {
  const code = error instanceof ZodError ? 'INVALID_INPUT'
    : error instanceof AccessError && failureClasses.has(error.code) ? error.code : 'UPSTREAM_UNAVAILABLE';
  const outcome = ['UNAUTHENTICATED', 'ACCESS_DENIED', 'NOT_FOUND'].includes(code) ? 'denied'
    : ['INVALID_INPUT', 'UNKNOWN_TOOL', 'PROJECTION_TOO_LARGE'].includes(code) ? 'invalid' : 'failure';
  return { code, status: error instanceof AccessError ? error.status : 503, outcome };
}

// Shared D1/D3 boundary. Correlation and observations confer no authority.
// A failure to persist the start event prevents the callback from running.
export async function runAuditedTool<T extends Record<string, unknown>>(
  client: GoClient, journal: AuditJournal, name: string, args: Record<string, unknown>,
  operation: (client: GoClient) => Promise<T>,
): Promise<T> {
  const correlationId = randomUUID();
  const started = performance.now();
  const key = uuid(client.identity?.keyId);
  const keyId = key && client.identity?.principalId === `agent:${key}` ? key : null;
  const tool = Object.hasOwn(scopes, name) ? name : 'unknown_tool';
  const event: AdapterAuditEvent = {
    correlationId, principalId: keyId ? `agent:${keyId}` : null, keyId,
    workspaceId: uuid(args.workspaceId ?? (args.sceneRef as Record<string,unknown>|undefined)?.workspaceId), sceneId: uuid(args.sceneId ?? (args.sceneRef as Record<string,unknown>|undefined)?.sceneId), tool,
    scope: scopes[tool] || '', outcome: 'started', latencyMs: 0, failureClass: '',
    provenance: 'adapter_observed', identitySource: keyId ? 'prior_verified' : 'unknown',
    timestamp: new Date().toISOString(),
  };
  try { journal.record(event); } catch { throw new AccessError(503, 'AUDIT_UNAVAILABLE', correlationId); }

  let result: T | undefined;
  let failure: ReturnType<typeof classified> | undefined;
  try { result = await operation(client.forCall(correlationId)); } catch (error) { failure = classified(error); }
  event.latencyMs = Math.min(Math.max(Math.round(performance.now() - started), 0), 86400000);
  event.timestamp = new Date().toISOString();
  event.outcome = failure?.outcome || 'success';
  event.failureClass = failure?.code || '';
  if (tool === 'create_scene' && result) {
    // Read only a UUID from the known result envelope; never retain its title/content.
    const content = result.content as { text?: string }[] | undefined;
    if (content?.[0]?.text) {
      try { event.sceneId = uuid(JSON.parse(content[0].text).id); } catch { /* no scene ID to record */ }
    }
  }
  try { journal.record(event); } catch {
    // A start record and possibly committed Go operation exist. Never retry a
    // non-idempotent create just because its completion observation is missing.
    throw new AccessError(503, 'AUDIT_COMPLETION_UNCERTAIN', correlationId);
  }
  if (failure) throw new AccessError(failure.status, failure.code, correlationId);
  return { ...result!, _meta: { ...(result!._meta as object || {}), agentCallId: correlationId } };
}
