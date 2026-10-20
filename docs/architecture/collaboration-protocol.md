# Collaboration protocol (v2, deployed)

## Enterprise metadata invalidation

The existing authenticated socket carries a cursor-only
`feature_changed { eventCursor: string }` control after the runtime has checked
the recipient's current authority. The cursor is a positive decimal string and
the strict payload contains no other fields. It is an invalidation hint, so the
browser refetches an authorized activity or feature page and also refetches
after reconnect. No workspace event content is transported on the realtime
channel. Schema, projection and privacy rules are defined in
[Enterprise workspace events](enterprise-events.md).

The frontend `FeatureInvalidationCoordinator` is the production consumer. A
feature surface registers its authorized HTTP/query refetch function through
`useAuthorizedFeatureRefetch`; the coordinator never accepts event data as a
cache result. It tracks the highest decimal cursor only as an invalidation
watermark. Live, repeated, stale, and out-of-order cursors all cause a refetch,
and invalidations received during an in-flight request are coalesced into one
follow-up request using the highest observed cursor. If no feature surface is
mounted, the invalidation remains dirty until one registers.

Every admitted socket connection, including a reconnect after a possibly lost
control, invalidates registered queries. Window focus, restoration of a visible
document, route navigation, and the explicit `refreshFeatures()` action do the
same. Route and connection identities are deduplicated. Each registered loader
must call its normal authorized API, so the HTTP response is filtered using
current authority rather than authority implied by the socket notification.
These paths share `/collab`; the browser opens no workspace event socket.

Go owns access decisions; the custom Node service owns Yjs content. These
contracts describe the deployed VM100 runtime; the release checklist records
the remaining restart/TURN/denial/roles acceptance boundaries.

## B1 behavior and verification

`@collab/contracts` owns serializable Excalidraw native elements, scene content, mutation commands, revisions (decimal strings), and binary frames. `@collab/runtime` owns document construction, materialization, and isolated candidate validation. Diagram compilation depends on contracts, never on runtime.

Binary layout: version `1` (u8), channel (u8: sync=1, awareness=2, update=3), UTF-8 update-ID byte length (u16 big endian), payload byte length (u32 big endian), update-ID bytes, payload bytes. ID is 1–128 UTF-8 bytes. The complete frame is at most 1,048,576 bytes. Exact lengths, valid UTF-8, supported channel/version, and no trailing bytes are required. Frames contain no room identifier; authentication binds the room. JSON control frames use `{type,requestId,payload}` and remain the service layer's responsibility.

Scene content is `{elements,appState,files,presentationOrder?}`. All first three fields are required, with empty maps valid. Persisted app state contains only shared canvas styling (`viewBackgroundColor`, `gridSize`, `gridStep`, `gridModeEnabled`, `theme`). Camera, selection and credentials are rejected. Files are references (`id`, `mimeType`, `created`, `assetId`, optional `lastRetrieved`, `version`), never data URLs or arbitrary filesystem paths. Embedded executable iframe/embeddable/selection elements are unsupported. Strict native field schemas retain normal bindings, tombstones and native text/path geometry; links are HTTP(S), mailto or relative safe links. Unknown fields, unsafe metadata keys, invalid geometry/references and ID/type changes fail.

Yjs exact shape: named top-level `elements: Y.Map<Y.Map<field,value>>`, `order: Y.Array<string>`, `appState: Y.Map<field,value>`, `files: Y.Map<Y.Map<field,value>>`, `presentationOrder: Y.Array<string>`. Nested field values are JSON scalars/arrays/objects, never nested shared types. Every native element field is a separate map entry. Element deletion sets `isDeleted=true`; removing an existing element map is invalid. References may target tombstones. Materialization traverses order once (first occurrence wins), then missing IDs in lexical order. Dangling order entries fail; duplicate entries from concurrent moves are normalized deterministically. Presentation order is duplicate-free and references frame elements.

The exact 25 MiB (26,214,400 UTF-8 bytes) materialized JSON ceiling applies after validation. Untrusted updates are decoded only inside a worker with a 2,000 ms wall deadline, 256 MiB old-generation heap, 16 MiB young-generation heap, and 4 MiB stack. These are V8 limits, not a total-process RSS guarantee. Input update bytes are capped at 1 MiB before worker allocation; trusted serialized snapshot history is separately capped at 64 MiB to bound copying and external buffers. JSON nesting is limited to 32 levels. Native geometry and group/reference cycle checks are linear in document size.

`validateCandidate(current, update): Promise<ValidatedUpdate>` returns copied update, materialized scene and state vector but never changes `current`. The caller must serialize room writes, durably append the returned validated bytes, and only then apply/broadcast/ack. Its current snapshot must match the one validated. Sync-step-2 is a write whenever mutating. Raw Yjs updates also reject trailing bytes and unresolved dependency structs/delete sets. Existing IDs cannot be retyped, removed, or have their entire field maps replaced. Yjs documents use `gc:false`; native deletion sets `isDeleted=true`.

`SnapshotHistoryLimitError.code` is `SNAPSHOT_HISTORY_LIMIT`. This is a fail-closed operational recovery condition, separate from the 25 MiB visible document ceiling. The room must preserve durable and pending updates/receipts and stop accepting writes until an operator follows a tested B2/E3 recovery procedure. Ordinary update-log snapshot pruning does not necessarily reduce retained Yjs struct history. Never silently re-seed the document, change its epoch or drop another client's queued edits.

Tests prove bad versions/lengths/trailing bytes, exact frame/document
boundaries, bad nested fields and references, rejected data URLs, independent
field convergence, native tombstones/bindings, real worker deadline/heap
termination and live-state immutability. The B1 execution report records
red/green evidence. Real WebSocket, PostgreSQL durability, ACL, browser and
fault/recovery behavior now pass local real-stack acceptance; VM100 deployment
remains E3 work.

Materialization checks Yjs's underlying compartments as well as public values: all map roots and nested element/file field maps have no sequence structs; both array roots have no keyed structs and contain only native string-reference array content. Retained deleted/replaced nested maps and field values are checked too, so adding then deleting a hidden shared container cannot evade the boundary. These checks use the pinned Yjs `_start`/`_map` representation and exported content classes; dependency upgrades must rerun their regressions. Binary update-ID bytes must exactly match UTF-8 re-encoding after strict decoding; leading BOM bytes are rejected consistently with encoding, preventing empty or aliased IDs. Focused red/green regressions verify these cases without dependency changes.

## Workspace installation

Run `npm ci` from `src`, then `npm test --workspace ./collab -- document.test.ts`, `npm test --workspace ./packages/collab-contracts`, and frontend unit/build commands. Root `npm test` runs packages with a test script; frontend uses the separate `npm run test:unit --workspace ./frontend` script. New packages pin Yjs 13.6.33, lib0 0.2.119, Zod 4.6.7; shared tooling pins TypeScript 7.0.2, Vitest 4.1.12 and Vite 7.3.5. Excalidraw 0.18.1 is a type-only development dependency of contracts. No renderer code is imported by the server.

The root lock is authoritative for workspace installs. Frontend versions follow the reviewed root patched dependency graph; the older standalone lock remains temporarily for the existing frontend-only Docker context until E2 migrates the build context. `.npmrc` preserves legacy peer resolution because npm's automatic peer graph failed during migration. Shared root dev dependencies explicitly provide Vite, jsdom, React/ReactDOM and their types, avoiding inaccessible nested peers and duplicate hook implementations. After migrating an existing checkout, run `npm exec --workspace ./frontend -- tsc -b --force` once if stale TypeScript incremental caches retain old module resolution. Fresh-checkout builds need no cache repair.

## B2 durable service contract (deployed; final runtime canaries pending)

Public WebSocket endpoint `/collab` accepts no credentials in its URL. First text message within five seconds is `{type:"auth",requestId,payload:{token}}`. Subsequent refresh uses the same shape with `type:"refresh"`. Tokens bind one room; refresh cannot change principal or scope. Server replies `{type:"ready",requestId,payload:{workspaceId,sceneId,principalId,access,documentId,sequence,stateVector}}`. All JSON controls use `{type,requestId,payload}`; printable ASCII request IDs are 1–128 characters. Clients retain pending updates until a matching `ack` with payload `{requestId,sequence,stateVector}`; sequence is a decimal string. The authenticated connection is never rebound by binary data.

Channel 3 carries raw Yjs update bytes. Channel 1 follows lib0 varuint subtype plus varUint8Array payload: `0` = state-vector read request; `1` = client sync-step-2 write; `2` = update write (both use the same validation/authority/durability as channel 3). Trailing bytes fail. Server sync replies use JSON `{type:"sync_chunk",requestId,payload:{documentId,sequence,index,count,totalBytes,data}}`; `data` is base64, chunks decode to at most 512 KiB, ordered zero-based indices, concatenate all chunks before applying the Yjs update. Bound assembly to declared totalBytes <=64 MiB, consistent documentId/requestId/sequence/count, and exact ordered indices. documentId is the immutable initial-seed UUID, also the offline storage epoch; no automatic epoch switch occurs. This permits large existing documents without increasing the 1 MiB incoming frame ceiling. Server durable broadcasts are channel-3 frames carrying the original request ID and validated update. Channel 2 awareness is ephemeral JSON bytes and cannot write scene content.

Errors are `{type:"error",requestId,payload:{code,message,retryable}}`; codes include `UNAUTHENTICATED`, `FORBIDDEN`, `STALE_AUTHORITY`, `CONFLICT`, `REQUEST_ID_REUSED`, `INVALID_UPDATE`, `RATE_LIMITED`, `UNAVAILABLE`, `COLLAB_DISABLED`, `PROMOTION_REQUIRED`, and `SNAPSHOT_HISTORY_LIMIT`. `SNAPSHOT_HISTORY_LIMIT` sets `retryable:false` and requires operator recovery; clients preserve queues and offer export. `resync_required` has payload `{reason}` and closes the socket (1013); reconnect with a fresh room token and resend unacknowledged request IDs. ACL notifications invalidate every connection in the workspace; listener loss invalidates all connections and readiness until listening resumes. Room tokens are checked at admission, every read, every write and on expiry; no stale-room grace.

Private HTTP `POST /internal/scenes/:id/commands` runs on a separate loopback/private listener and requires `Authorization: Bearer <COLLAB_INTERNAL_SECRET>`. Body is `{grant,mutationBase64}`: grant is A4's signed `scene:command` decision token; decoded bytes are the original UTF-8 SceneMutation JSON, hashed before parsing. Mutation `{workspaceId,sceneId,requestId,expectedSequence,commands}` uses shared contracts. HTTP 200 returns the receipt; errors use `{code,message,requestId}` with 400/401/403/409/429/503 as appropriate. The public listener exposes only `/collab`, `/healthz`, `/readyz`.

A decision is requested from Go before any database transaction. Workspace row lock precedes scene/document lock; validate signature/byte hash/scope/version and clock expiry within that transaction, then validate the candidate, append update+receipt+audit atomically. Duplicate IDs are unique per scene and bind principal type/ID plus original payload hash; current authority is required even for replay. Failed or uncertain commits yield no ack or broadcast. Retried committed requests return the original sequence/state vector. Snapshot compaction records cutoff, persists full Yjs state and materialized scenes.content, then prunes covered log rows in the same transaction; receipts and audits are never pruned. Snapshot triggers at 100 updates or five seconds. Seed and writer-mode transition share the same locks; Go metadata updates exclude content entirely.

B2 verification uses random isolated schemas in a loopback *_test database, actual PostgreSQL termination/rollback and two-connection lock ordering, real WebSocket admission/update/retry, built worker execution, and the Go HTTP writer fence. No skipped database test is accepted as proof. Exact executed results are recorded in the B2 task report.

### Promotion, agents and deployment boundary

Production `COLLAB_V2_ENABLED` defaults to false and only gates new v1-to-v2 promotions. Existing v2 documents keep exclusive ownership when the switch is disabled. Initial promotion retains original JSONB/contentVersion in the document row, allowlists shared app-state, maps every legacy file through active workspace-and-scene-scoped asset metadata, and validates native elements/bindings. Unmappable assets or invalid content return PROMOTION_REQUIRED and leave writer_mode=v1 and original JSON intact. Existing native validators are not relaxed. JSON materialization never reduces the legacy contentVersion base.

Node accepts agent decisions only for scene:command/scene:read/scene:export, with principalType=agent, principalId/sub=agent:<canonical key UUID>, grantId=<same UUID>, and no grantVersion. Agent credentials cannot join WebSockets or send raw updates. Go remains responsible for current key/owner/target/action checks. Private commands and WebSocket updates share the principal rate budget, durable locking, candidate worker, receipt and audit path. Rate-limited bytes remain queued; a committed duplicate receipt is returned without consuming the persistence mutation budget.

The process owns a PostgreSQL advisory lock scoped by database/current schema to enforce one active runtime. ACL listener failure closes all sockets; reconnection reacquires ownership and LISTEN before readiness. JSON awareness accepts only finite cursor coordinates and bounded selectedIds, adds server-derived principalId, stays ephemeral, and requires current read authority. Binary frame `updateId` is the control/receipt requestId. Initial ready is followed automatically by a complete sync_chunk stream sharing the auth request ID; refresh also sends a fresh snapshot. A new sync request can carry a standard Yjs state vector to receive a differential update.

Pinned transport/storage/auth dependencies: ws 8.22.0, pg 8.23.1, jose 6.2.12; type packages @types/ws 8.18.2 and @types/pg 8.23.1. Official npm registry versions/integrities were checked before installation. No broad dependency upgrade was performed. The reviewed root graph remains authoritative; audit findings inherited through renderer/tooling dependencies remain separately tracked.

### Runtime ownership and maintenance after review

Runtime transactions acquire a shared transaction-level handoff advisory lock before the workspace row lock, verify the current listener's exact PID/backend_start primary-lock identity, then retain that barrier through COMMIT/ROLLBACK. A replacement runtime holds the primary owner session lock and waits at an exclusive handoff barrier before becoming ready. Checks before and after COMMIT suppress publication/ack after ownership loss; a commit in the final-check gap remains durably retryable. All private commands and runtime reads/promotions/appends/maintenance use this fence. Go authority HTTP requests still precede transaction locks. The exact keys and timeout behavior are in the operations guide.

Inactive scenes/workspaces are omitted from due scans and rechecked under the workspace/scene locks. A trash race is a maintenance no-op that preserves updates, receipts and document identity. Restoration makes retained work eligible again. Already-covered documents do not rewrite snapshots. Maintenance errors affect only the failed scene; scanning continues to other eligible documents.

The scheduler runs one due scan and one scene job at a time. It retains at most 64 queued scene jobs and 64 five-second/count timer entries, deduplicating scan/timer/count work against queued and running scenes. Each due query returns at most64 refs with a stable workspace/scene cursor; later pages recover overflow work from PostgreSQL. Timer/count thresholds request work; storage backlog can delay execution rather than create unbounded parallel jobs. Shutdown stops intake, clears queued/timed work (retained durably), and drains the bounded in-flight tasks. Direct compactDue() handles one bounded page, continuing after individual failures before returning an AggregateError.

## M2 presentation waiting-room contract (implemented locally; release pending)

`presentation_auth` accepts only a signed five-minute
`presentation-pending` credential. The connection is registered in the
session-local waiting queue without a scene principal, room reference,
awareness entry or durable read. It receives `presentation_waiting` for its own
connection only. Binary frames and ordinary ephemeral controls fail before any
scene, roster or media data is read.

An editor authenticated through the ordinary room starts a session with
`presentation_start`. Host controls recheck `presentation:host` against the Go
private authority. `presentation_admit` carries the exact waiting connection,
expected decimal session revision and Go-issued `presentation-audience` token.
Node verifies token/session/principal binding and requires the token grant
version to equal that expected revision. Only the selected pending socket
receives `presentation_admitted {sessionId,token,expiresAt}`; the bearer is not
sent to the host or other waiters. Initial sync, channel-1 sync-step-1 resync,
refresh and committed fanout all resolve presentation audience authority and
never fall back to ordinary room reads. The Go resolution path rechecks the
active host's current scene edit authority and `presentation_enabled` flag.
Feature invalidation, host disconnect, explicit stop or current-authority
denial ends the ephemeral session before further protected delivery. Restart
retains stored invite links but no active host, waiting or audience state.

`@collab/contracts` exposes strict bounded parsers for the audience's own
`presentation_waiting` payload and host `presentation_state` snapshots. Pending
disconnects increment the session revision and publish the new state to the
host so a stale admission cannot succeed.

Private service-authenticated `POST /internal/presentations/resolve` exposes
only active host/pending/audience decisions to Go. It uses the same isolated
private listener, ownership fence, request limit and service secret as durable
agent/version operations and must never be publicly proxied.

The public join limiter keys requests by the direct peer IP captured before
Chi's `RealIP` middleware may rewrite `RemoteAddr`. Forwarding headers from an
untrusted public peer therefore cannot create extra rate-limit buckets.

## D3 agent bridge (deployed; production write/export canary pending)

The authenticated Go agent API bridges POST `/api/v1/agent/workspaces/:workspaceId/scenes/:sceneId/commands` with `{mutationBase64,tool}` and GET the existing scene route (durable projection), `/activity?cursor=<sequence>&limit=1..50`, and `/export-projection` (distinct `scene:export` scope). Remote MCP only holds its agent key. Go mints/verifies a room credential, signs the exact decoded command bytes through A4 and forwards `{grant,mutationBase64}` to B2. The `tool` discriminator is allowlisted and checked against the command operation. There is no whole-scene replacement. Go does not hold a workspace transaction during Node HTTP. Go boundary audit success follows the Node result; only B2 receipt/audit proves durable append. An audit completion failure after append is uncertain and retains the same request bytes/ID.

Private Node POST `/internal/scenes/:id/projection`, `/activity`, `/export-projection` require the private service credential and RuntimeOwnership, just like `/commands`. Read bodies carry `{grant,workspaceId,sceneId}`; activity additionally carries canonical decimal `cursor` (default `0`) and `limit` (default50, maximum50). Grants use `scene:read` or independent `scene:export`, bound to the scene and empty-byte hash. Every operation uses the ownership barrier, current ACL version and expiry under workspace/scene locks. Projection returns `{workspaceId,sceneId,documentId,sequence,stateVector,content}` with <=500 elements and <=256KiB, rejecting oversized scenes. Export projection is separately authorized and bounded by the25MiB document limit; D4 controls downstream rendering/downloads. Activity lists ascending committed sequence, requestId, principalType, action and createdAt only; no principal identifier, payload hash, grant, key, or document. `nextCursor` is the last returned sequence only when another row exists.

MCP `generate_diagram({sceneRef,preset,data,instanceId,requestId,expectedSequence,dryRun})`, `patch_elements({sceneRef,patches,requestId,expectedSequence})`, `delete_elements({sceneRef,ids,requestId,expectedSequence})` use stable SceneMutation bytes and the shared writer. `expectedSequence` is a decimal durable revision, never contentVersion. Write-only keys may submit targeted writes with a provided revision without reading a scene. Dry-run validates/compiles and returns preview IDs/warnings without contacting the writer. Successful mutations return `{receipt,sceneUrl,createdIds,warnings}`. Repeating identical arguments/requestId retains deterministic IDs and bytes; changed payload/requestId reuse or stale sequence is409, never silently rebased/retried. Fresh read plus a new request preserves unrelated human fields.

Verification: actual Go+Node+PostgreSQL+MCP integration for exact scope/targets, generation and restart, same receipt/one durable audit on replay, payload conflict, stale sequence, fresh-read targeted patches, tombstones, dry-run immutability, activity pagination/redaction, private ingress/ownership fencing and scope-independent export. B3/E1 owns browser-visible same-ID proof. No browser or production evidence is claimed by the contract itself.

Go runtime configuration for this bridge is the pair `COLLAB_INTERNAL_URL` / `COLLAB_INTERNAL_SECRET`. The URL is an HTTP(S) origin without credentials/path/query/fragment; the secret is >=32 bytes and distinct from login, room, authority-service and mutation-grant secrets. Neither setting has a credential fallback. Leaving both absent disables durable agent paths with503. Remote requests do not follow redirects; Node responses are bounded (normal256KiB, export26MiB envelope) and have an8-second timeout. Read projection's500-element/256KiB bound is also enforced inside the Node transaction. MCP's complete text envelope is separately capped at256KiB.

The D3 test harness uses real Go HTTP handlers and A4 signing against a random disposable schema, real Node runtime/worker and SDK MCP HTTP requests. A user-token WebSocket receives generated IDs, writes a human field, and observes targeted mutation convergence. This verifies socket transport, not rendered browser UI. Root-owned release/browser evidence must establish the latter before rollout.

D3 review correction: activity cursors remain decimal strings, while SQL ordering explicitly uses the qualified bigint source column. The same rule applies to recovery-log export ordering. Verified pagination crosses9→10 and99→100 without omissions or duplicates. MCP `WHITEBOARD_PUBLIC_URL` is a trusted frontend/reverse-proxy origin distinct from `WHITEBOARD_API_URL`; audited client copies preserve it for scene links, artifact links and remote-export requests. API-only clients may omit a public base, but generating public links or forwarding exports then fails rather than leaking a private API origin. HTTP/stdio entrypoints require the public origin in production.

## Committed broadcast batches

The runtime coalesces committed Yjs updates for 100 ms and serializes fanout per room. Channel 3 can therefore contain a merged update; its server broadcast ID (`s`) is informational and must never acknowledge or remove a client pending update. Only the explicit ACK carries the original durable request ID, sequence and state vector. Clients apply every channel-3 payload idempotently. Individual frames remain at most 1 MiB.

Every batch recipient still passes a fresh Go-authorized, current ACL/version and runtime-ownership-fenced durable read immediately before delivery. There is no authorization cache or stale projection fallback. Explicit denial/expired credentials close with AUTHORITY_CHANGED; transient authority/database/runtime failure closes with UNAVAILABLE without sending protected update bytes. Reconnect uses fresh admission and durable synchronization.
