# ADR 0011: Native MCP with scoped server-managed keys

Status: accepted for collaboration v2; implementation validation in D1 report.

Go is the authority for bearer keys, current owner membership and scene scope. Keys use 32 random bytes and SHA-256 storage; audit records only key IDs. Key changes bump workspace ACL under the same workspace lock as writes. Agent principals have no user identity and can never return Owner=true. No agent key authenticates on login-only routes.

The adapter uses official `@modelcontextprotocol/sdk` 1.32.1 (registry verified 2026-10-16) and zod 4.6.7, with project TypeScript 7 and Vitest 4.1.12. SDK source/license: https://github.com/modelcontextprotocol/typescript-sdk (MIT). Official transport specification: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports. Streamable HTTP authenticates every request, validates Origin and binds sessions to principal. Stdio emits only MCP frames on stdout and calls the remote Go API; diagnostics use stderr. PAT compatibility is explicit; OAuth discovery/PKCE is deferred.

Dedicated `/api/v1/agent` API endpoints provide authentication, tenant-filtered discovery, bounded projection, empty-scene creation and scoped room credential exchange. MCP never forwards credentials to caller-selected URLs. Room tokens retain ten-minute expiry and agent key identity; Go re-resolves current key, owner, targets and action scope for every authority decision. Scene create requires `scene:create`; room edit requires `scene:write`, while read requires `scene:read`.

Verification requires actual PostgreSQL random schemas and official SDK HTTP/stdio client lifecycle tests; no local database in the stdio process. Public and user credential semantics remain unchanged. Human guide and public Markdown mirror are the documentation surfaces; controller owns central API catalog integration.

D1 materialized reads expose contentVersion, not the durable collaboration sequence. D3 must integrate the authenticated current durable read for live mutation preconditions. The adapter returns a single text projection and enforces the encoded result budget, avoiding a second structured copy that would exceed 256 KiB.
