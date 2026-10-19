# B4 presence, follow and voice verification

Status: implemented and verified locally; production rollout pending. Authorized by the [approved design](../../superpowers/specs/2026-10-15-collaboration-v2-design.md), [runtime plan](../../superpowers/plans/2026-10-16-collaboration-runtime.md) and [phase](../phases/PHASE-COLLAB-V2.md).

## Design before implementation

One authenticated custom WebSocket owns document traffic, presence, follow and signaling. Server-generated UUID connection IDs differ from authenticated principal IDs: two tabs are two participants. The server binds identity and room from admission, ignores claimed sender identity, and checks current authority for sender and recipients. Names are bounded display labels, never identity evidence. No RelayHub or second Socket.IO connection is used by v2.

Control messages retain `{type,requestId,payload}`. `ready` adds `connectionId`. `roster` returns `participants` with connectionId, principalId, principalType, displayName, access, voiceJoined, voiceMuted and following. `presence` accepts displayName. `voice_join`/`voice_leave` take an empty payload; `voice_mute` takes muted. `voice_admission` returns admitted, connectionId, peers and optional reason. `voice_signal` accepts to and one description or candidate, and server forwards its own from/to. Follow takes targetConnectionId (or null); server rejects self, absent and cyclic targets. `viewport` carries scrollX, scrollY and zoom to followers only. Cursor/selection binary awareness remains compatible and gains server identity on forwarding.

Voice admission is synchronous after current authorization: ten connections per room, repeat join idempotent, eleventh denied voice only. Leave, close, revoke and stale-peer termination release a slot. Voice-specific or follow-input errors retain document access. Browser permission is requested only after admission; denial must send leave and keep editor working. Early ICE is buffered; deterministic offer ownership avoids glare; all tracks/peers/audio are cleaned on leave, connection change or loss.

E2 reproduced nginx's bounded 75-second upstream idle timeout at 75,082ms. Root cause: no outbound heartbeat. Add WebSocket protocol ping every 25 seconds, terminate if the next interval sees no pong, and remove ephemeral presence immediately. This bounds idle stale slots at 50 seconds and keeps legitimate idle connections through nginx without increasing its timeout. Browser protocol pong requires no app handler. Timers stop during shutdown.

No schema or document mutation is added. Camera, selection, follow and voice state never enter Yjs, snapshots or HTTP scene content. The legacy reference is sibling `references/ws-whiteboard/src/index.ts` plus Excalidraw follow behavior; roster and voice metadata are retained with stronger room/identity checks. The rejected alternative is retaining the legacy presence socket: it would duplicate authority and connection lifecycle.

## Ownership and verification plan

First slice owns collab awareness/voice modules, Rooms lifecycle, tests, this note and architecture/INTEGRATIONS.md. Frontend integration waits for explicit B3 ownership transfer. Root owns central protocol/architecture/operations/user/public agent surfaces and final validation matrix/backlog/release reconciliation; this note provides their exact contracts and evidence.

Prove RED then GREEN: distinct IDs for two tabs; 10/11 admission with continued durable edits; repeat join and disconnect release; current-ACL and cross-room signaling denial; spoofed sender replaced; self/cycle follow rejected; target removal clears followers; viewport ephemeral; ping/pong keeps a socket while no-pong releases admission. Use actual WebSockets, disposable guarded PostgreSQL for durable proof, and later actual idle traffic through nginx for >75 seconds. Frontend tests must prove delayed permission/denial, early ICE, stable negotiation and cleanup; browser smoke verifies user controls.

## Evidence

Server foundation evidence (2026-10-16; frontend handoff still pending):

- RED: `npm test --workspace ./collab -- voice awareness` failed all seven new real-WebSocket cases for missing connection IDs and controls. Build and fixture succeeded.
- GREEN: same command passed 7/7 after implementation. Expanded lifecycle/input suite then exposed stale roster access (`edit` after current authority returned `view`); refreshing the roster identity from each successful authority read fixed it. Expanded suite passes 10/10: delayed admission after disconnect, repeated concurrent joins, bounded SDP/ICE and sanitized labels included.
- PASS: `npm test --workspace ./collab -- presence.integration.test.ts server.integration.test.ts` — 16 tests. The new guarded disposable PostgreSQL regression admits ten voice connections, denies eleventh voice only, commits and acknowledges the eleventh connection's Yjs edit, checks durable receipt, compacts and proves camera/follow/voice absent from persisted content, then frees/reuses a voice slot on disconnect.
- PASS: `npm test --workspace ./collab` — 11 files, 116 tests, no skipped tests. Existing persistence, private ingress, runtime ownership, revocation and worker checks remain green.

Commands ran via `rtk proxy env PATH=/Users/dungxbuif/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:/usr/local/bin:/usr/bin:/bin` from `src`. Unit transport fixtures substitute only persistence/authority boundaries; the new integration test uses real signed authority verification, Persistence, Yjs and a random schema in `127.0.0.1:55439/devtool_a1_test`.

PASS: `rtk proxy env PATH=<above> node tests/packaging/idle-collab.mjs` from repository root — elapsed 160,190ms; 25,000ms production heartbeat; six live pings; zero live closes; nonresponsive peer terminated at 47,586ms with expected abnormal close1006; stale voice admission absent; readiness200. Exact unchanged nginx image: `sha256:ed1878f5a767c7cb66f75822b2f83117f1ae05904eea4cd9da49b6b2255c3c14`. The raw TCP forwarder only bridges Docker Desktop to the random host loopback port; actual compiled Rooms/Persistence and random-schema PostgreSQL perform admission/state. Both QA containers, unique network and schema were removed in finally. This run preceded the independently tested ingress-budget change; heartbeat/lifecycle code is unchanged by that change.

At this checkpoint no frontend/media/browser or production pass had been claimed; final evidence is recorded below.

## Frontend completion and legacy transport retirement decision — 2026-10-19

The real-stack browser now proves distinct same-user tabs, server-backed native follow/camera behavior, microphone-denial recovery, and two native peers exchanging generated audio with mute/leave cleanup. The first follow run exposed an Excalidraw emitter loop: projecting a new `userToFollow` object for every roster caused repeated UNFOLLOW/FOLLOW controls. A native-emitter regression now requires stable projection by server target and coalesces genuine native target changes before sending them.

With the replacement path proven, remove the unused frontend Socket.IO client/types, Go Socket.IO server/package, router mount, and nginx `/socket.io/` location. The HTTP legacy scene fallback remains available for unpromoted v1 scenes; it does not require the retired presence socket. Keep `/api/v1/realtime/config` on the scoped capability handler and keep `/collab/` as the only v2 realtime ingress. Verification must include frontend unit/build, all Go tests, the complete collaboration runtime suite, the three real-stack presence/voice flows, dependency-tree checks, and documentation/API/runbook reconciliation.

## Ingress-budget ruling before implementation

Controller approved replacing the inherited 600-frames/minute/IP bucket: ordinary 20 Hz cursors from even one user exhausted it, conflicting with the 50-client acceptance workload. Preserve write limits at 120/minute/principal and limit authentication to 60/minute/IP. Share a 30/second/connection budget between cursor and viewport; other ephemeral controls use 120/minute/connection. Apply a separate 2,000-frames/second/IP envelope budget and 64 MiB/minute/IP byte refill with an 8 MiB burst. Count received raw bytes before JSON/binary decoding, including rejected payloads. Keep the existing 1 MiB frame and 32 KiB control limits. Expired limiter entries are pruned and live entry counts bounded; these are resource safeguards, not participant limits. Add simulated 50-client normal cursor/write traffic, flood/byte exhaustion and limiter-expiry proof before changing production code.

Ingress RED: the new absent limiter first failed module loading; a minimal interface stub then produced three behavioral failures for missing frame/byte enforcement, clock/window handling and expiry. GREEN implementation adds bounded IP token buckets, raw-byte accounting before decode and separate authentication/operation budgets. A concurrent-close test exposed a fixture synchronization assumption: client close can fire before the server consumes its close event. The assertion now waits for the server roster departure before retrying admission, matching the actual lifecycle contract.

Final server-foundation verification after ingress changes: `npm test --workspace ./collab` PASS12 files/119 tests/0skips (17.54s).

## Frontend integration design — 2026-10-17, before code

B3 fix3 reviewed; frontend ownership transferred, except wrapper image baseline repair remains temporarily owned by adapter_b3. E1 owns runtime performance files; B4 will not modify those. Per-provider Presence and Voice controllers subscribe to the existing CollabTransport, avoiding global cross-scene state or a second socket. Transport exposes connectionId and returns request IDs for bounded ephemeral request correlation; voice/follow/input errors notify controls without changing document authority or queues. Connection loss clears ephemeral state immediately; same-connection token refresh preserves it.

Roster uses server connection IDs and effective editor/viewer labels. Native Excalidraw collaborators receive names/cursors/selections, userToFollow and followedBy. Native follow and an accessible roster control use the same server follow graph; viewport is throttled, applied only for the current target and never sent through content writers. Follow chains cannot form cycles. Late viewport from an old target is ignored.

Explicit Join voice/Leave voice/Mute controls expose waiting, permission and failure states. Admission precedes TURN config retrieval and microphone permission. Same-user tabs remain independent. Config comes from GET /api/v1/realtime/config with X-Workspace-ID and a newly minted scene-scoped room JWT (the API rejects ordinary login JWTs); no endpoint/secret hardcoding. Denial, cancellation, connection loss, refresh failure or teardown releases admission and stops all media. Late permission/config promises are guarded by lifecycle generation and stop late tracks. Peers use deterministic offer ownership (lower connection ID initiates), serial signaling, early ICE buffering and bounded pending signals; all audio elements are cleaned. Refresh short-lived ICE configuration before expiry. Browser tests cover native roster/follow/media denial and a continued drawing edit, with screenshots. Unit tests cover delayed permission, early ICE, stable negotiation, quota and cleanup.

No image-draft revision/queue/undo/content logic is to change. Existing upload tests must remain green. Old Socket.IO store becomes a compatibility export for per-provider presence types; dependency/router retirement is reported to root only after replacement proof.

## Final local evidence — 2026-10-19

- Frontend presence/transport/follow/native/voice tests: 24 passed. Full frontend suite after the follow and image-reopen regressions: 25 files, 93 tests passed. TypeScript/Vite production build passed.
- Collaboration runtime: 15 files, 149 tests passed against disposable PostgreSQL. All Go packages passed with the guarded test database.
- Real-stack browser: distinct same-user tabs, native cursor/follow/camera, target departure cleanup, microphone denial with continued durable editing, and two generated-audio peers with mute/leave cleanup all passed (3/3). The same complete real-stack run passed 17/17.
- Load on the final local HEAD: 650/650 durable acknowledgements, p50 9.742 ms, p95 13.236 ms, delivery p95 750.461 ms, peak runtime RSS 399,097,856 bytes on Apple M4 Pro ARM64. This is local acceptance; VM100 AMD64 remains the deployment gate.
- `npm ls socket.io-client @types/socket.io-client --all` is empty. Go Socket.IO source/module, frontend dependency and nginx location are removed. `/collab/` is the only collaboration socket; `/api/v1/realtime/config` remains scoped through the capability handler.

Production proof is intentionally pending until the frozen branch is merged, deployed on VM100 and exercised through the public HTTPS proxy and external TURN path.
