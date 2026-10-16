import type { Transport } from '@modelcontextprotocol/sdk/shared/transport.js';
import { CallToolRequestSchema, ErrorCode } from '@modelcontextprotocol/sdk/types.js';
import { AuditJournal, runAuditedTool } from './audit.js';
import { AccessError } from './auth.js';
import type { GoClient } from './go-client.js';

// Install before Server.connect starts the underlying transport. Only rejected
// envelopes are handled here; valid requests retain the handler's one audit pair.
export function auditToolEnvelopes(transport: Transport, client: GoClient, journal: AuditJournal): Transport {
  const boundary: Transport = {
    start: () => transport.start(),
    send: (message, options) => transport.send(message, options),
    close: () => transport.close(),
    get sessionId() { return transport.sessionId; },
    setProtocolVersion: version => transport.setProtocolVersion?.(version),
    get onclose() { return transport.onclose; },
    set onclose(value) { transport.onclose = value; },
    get onerror() { return transport.onerror; },
    set onerror(value) { transport.onerror = value; },
  };

  transport.onmessage = (message, extra) => {
    if (!('method' in message) || message.method !== 'tools/call' || !('id' in message) ||
        CallToolRequestSchema.safeParse(message).success) {
      boundary.onmessage?.(message, extra);
      return;
    }

    const params = message.params as { name?: unknown; arguments?: unknown } | undefined;
    const name = typeof params?.name === 'string' ? params.name : 'unknown_tool';
    const args = params?.arguments;
    const metadata = args && typeof args === 'object' && !Array.isArray(args)
      ? args as Record<string, unknown> : {};
    // Reject before SDK validation can bypass the handler. The wrapper retains
    // only allowlisted tool/UUID metadata and never the SDK's detailed exception.
    void runAuditedTool(client, journal, name, metadata, async () => {
      throw new AccessError(400, 'INVALID_INPUT');
    }).catch(async error => {
      const code = error instanceof AccessError ? error.code : 'UPSTREAM_UNAVAILABLE';
      await transport.send({
        jsonrpc: '2.0', id: message.id,
        error: {
          code: code === 'INVALID_INPUT' ? ErrorCode.InvalidParams : ErrorCode.InternalError,
          message: code,
          ...(error instanceof AccessError && error.correlationId
            ? { data: { agentCallId: error.correlationId } } : {}),
        },
      }, { relatedRequestId: message.id });
    }).catch(() => {
      boundary.onerror?.(new Error('MCP rejected-call response unavailable'));
    });
  };
  return boundary;
}
