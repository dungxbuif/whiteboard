# ADR 0011: Native MCP with scoped server-managed keys

Status: accepted for collaboration v2; implementation validation in D1 report.

Go is the authority for bearer keys, current owner membership and scene scope. Keys use 32 random bytes and SHA-256 storage; audit records only key IDs. Key changes bump workspace ACL under the same workspace lock as writes. Agent principals have no user identity and can never return Owner=true. No agent key authenticates on login-only routes.

The adapter uses official `@modelcontextprotocol/sdk` 1.32.1 (registry verified 2026-10-16) and zod 4.6.7, with project TypeScript 7 and Vitest 4.1.12. SDK source/license: https://github.com/modelcontextprotocol/typescript-sdk (MIT). Official transport specification: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports. Streamable HTTP authenticates every request, validates Origin and binds sessions to principal. Stdio emits only MCP frames on stdout and calls the remote Go API; diagnostics use stderr. PAT compatibility is explicit; OAuth discovery/PKCE is deferred.

Dedicated `/api/v1/agent` API endpoints provide authentication, tenant-filtered discovery, bounded projection, empty-scene creation and scoped room credential exchange. MCP never forwards credentials to caller-selected URLs. Room tokens retain ten-minute expiry and agent key identity; Go re-resolves current key, owner, targets and action scope for every authority decision. Scene create requires `scene:create`; room edit requires `scene:write`, while read requires `scene:read`.

Verification requires actual PostgreSQL random schemas and official SDK HTTP/stdio client lifecycle tests; no local database in the stdio process. Public and user credential semantics remain unchanged. Human guide and public Markdown mirror are the documentation surfaces; controller owns central API catalog integration.

D1 materialized reads expose contentVersion, not the durable collaboration sequence. D3 must integrate the authenticated current durable read for live mutation preconditions. The adapter returns a single text projection and enforces the encoded result budget, avoiding a second structured copy that would exceed 256 KiB.


## Review ruling: durable audit provenance (2026-10-16)

Every tool call receives a random correlation UUID. Go persists authoritative operation metadata in `agent_operation_audit`: server-derived key/principal, key workspace, requested workspace/scene, tool, exact scope, outcome, latency and allowlisted failure class. Successful creation and its audit commit in the same workspace transaction; audit failure rolls creation back. Denied/invalid operation events persist after rollback. Correlation is accepted only as a UUID in `X-Agent-Call-ID` and never confers authority.

The shared MCP `runAuditedTool` boundary appends and fsyncs metadata-only adapter events before attempting work and after completion. It covers local validation, denial and unreachable-upstream failures. No bearer, token hash, document, title, full arguments or exception text is logged. Fields have explicit `adapter_observed` provenance and `prior_verified`/`unknown` identity source; client-controlled stdio records are not authoritative. HTTP production requires an explicit absolute persistent `MCP_AUDIT_PATH`; stdio may use a protected per-client default state journal. Each deployment instance uses its own journal path. Files are mode 0600, bounded by rotation/retention; operations must collect journals for full-call visibility.

Journal write failure before operation fails closed. Completion-journal failure after an operation returns `AUDIT_COMPLETION_UNCERTAIN` with correlation ID; clients must inspect authoritative audit/state and never blindly retry creation. Prior start and Go operation records remain the evidence. D3 reuses the wrapper and Go audit service, retaining its own durable mutation receipts. Idempotent D3 mutations may retry the exact stable request ID and bytes even after uncertain completion; create_scene has no such deduplication contract. RunAuditedOperation is a top-level trusted service boundary: its callback performs authorization and uses the supplied transaction context; it must not be wrapped in an external transaction. Failure auditing uses a separate bounded connection after rollback.

Verification: real PostgreSQL operation success/denial and atomic create/audit rollback, audit metadata secrecy, reopened durable local journal, rotation/permissions, validation/upstream failure and failed completion semantics. Actual HTTP upstream 413 must preserve `PROJECTION_TOO_LARGE`. Bounded source formatting is a separate nonfunctional review cleanup.
