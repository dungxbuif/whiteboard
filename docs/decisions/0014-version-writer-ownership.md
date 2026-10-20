# ADR 0014: Scene version writer ownership

Status: accepted for V1 implementation.

Node remains the sole writer of durable scene content and product checkpoints.
Go owns authentication, feature policy, scene authorization, version reads and
label/pin updates. Manual create passes through Go, which signs
`version:create` with the exact workspace, scene, request ID, expected sequence
and canonical payload hash. Private Node ingress verifies the grant and exposes
no public version route.

Go copies the authenticated login credential expiry into the user principal
before capability issuance. If the private Node bridge is absent, manual create
returns typed retryable `503 UNAVAILABLE`; list/get remain available from
PostgreSQL. Node treats the workspace feature policy as current authority: after
locking the workspace it rechecks `versions_enabled` in the same transaction.
A grant issued before disable cannot cross that boundary.

Node captures the current Yjs materialization under its existing writer
transaction. The row stores document UUID, source sequence, canonical JSON bytes
and SHA-256. Every image must resolve to an active asset in the same workspace
and scene. Version row, asset pins, safe event and outbox commit together.
Content fields and request binding are immutable; label and pinned metadata use
a separate revision and emit a cursor-only event in the same Go transaction.

The runtime owner runs a bounded checkpointer. The first edit starts a
five-minute dirty interval, later edits coalesce, and the database prevents
checkpoints less than one minute apart. R1 owns deletion above 100 automatic
versions and must respect asset pins. Manual and pinned versions remain
indefinite by default. Compaction does not modify product versions.
Disabled workspaces are excluded from durable backlog scans; the default-false
policy creates no automatic versions or events.

Verification uses disposable PostgreSQL for exact sequence/hash, immutable
retry, atomic asset pin failure and compaction survival. Go tests cover feature
binding, guest denial, metadata revision conflicts and private bridge binding.
