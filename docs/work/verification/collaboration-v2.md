# Collaboration v2 verification

## C1 — original dashboard presentation

Date: 2026-10-16. Branch: codex/collaboration-v2. Production remains v1.

- Baseline backend: `rtk go test ./...` — 59 tests passed before v2 code changes.
- Frontend unit: `rtk npm run test:unit` — 33 passed, including malformed nested bindings/font, duplicate IDs, size limits, session-state filtering and external image rejection.
- Browser: `rtk npm exec -- playwright test dashboard-style.spec.ts dashboard.spec.ts --workers=2 --reporter=line` — 14 passed before visual baselines; final baseline comparison recorded below.
- Production frontend build: `rtk npm run build` — successful; dependency chunk-size warning remains.
- Red proof: original app lacked sidebar navigation landmark, mobile trigger and import input. New tests failed on those missing behaviors.
- Visual review found and fixed grid min-content overflow and drawer position specificity. Added bounds assertions. Dark dialog foreground and malformed nested import fields were independently reproduced by review and covered by regression tests.
- Original Header/SceneItem source rendered independently with original CSS, replacing only framework/data plumbing. [Reference rendering](collaboration-v2/legacy-reference-components.png). New screenshots inspected at 1440×960 and 390×844; original colors, type, spacing and card geometry preserved. Existing workspace heading/actions retained. No monochrome override.
- Four macOS Chromium golden screenshots under frontend e2e verify light/dark desktop and mobile drawer/empty view. Linux baselines are pending E1; no Linux visual result claimed.
- Review: two Important findings (nested import validation, dark dialog text) fixed. Follow-up legacy font gap fixed and unit-tested. Minor background-preview issue fixed.
- Docs review: human dashboard guide and identical public Markdown; llms.txt entry; architecture/ADR/spec/plans reconciled. API/schema/operations unchanged in C1.
- UI fixtures do not demonstrate live ACL, DB or collab correctness. Those remain A/B/E work.

Final C1 verification: 2026-10-16, frontend build exit 0; 33 unit tests passed; 14 Chromium tests passed including four inspected golden image comparisons (7.1s). No production deployment performed.

## A1 — authoritative workspace and scene access

Implementation commit `1c9883c` plus fix `b89357f`; task review passed after one fix round. Production remains v1.

- Targeted access suite: 16 tests passed against disposable local PostgreSQL 15. Full backend JSON result: 79 tests passed, 0 failed, 0 test skips; four packages have no tests.
- Proof includes owner backfill and legacy member-to-editor migration, UUID/alias compatibility, unique aliases, role matrix, current-membership overrides, owner protection, deleted resource denial, list/search isolation, protected folder descendants and scene-scoped assets.
- Two database connections demonstrate authority row contention: a competing revocation cannot pass the lock during the authorized write transaction; rollback and subsequent denial are checked.
- Regression tests first reproduced incorrect alias handoff, cross-scene asset reads and missing status invalidation. Final targeted/full suites and diff whitespace checks passed.
- API/ERD/ADR document branch behavior. Public invite/share/room guides and browser role controls await A2–A4/C2 integration; no complete collaboration or production claim follows from A1.
- Remaining integration: return effective role/access to the frontend, use publicId aliases in App routing, implement public/agent principal resolvers and replace owner-only v1 Socket.IO.
- Review reproduced an unbounded request-body read holding workspace authority. Fix buffers bounded input before DB/locks (10s deadline, 26MiB scene envelope, 1MiB metadata). A real stalled TCP client no longer blocks owner revocation and receives 408; 14 scoped tests pass. Independent re-review approved the fix.

## D2 — native diagram compiler

Implementation commit `6d85a7c` plus fix `316d0bb`, task review passed. All 19 presets are covered by 40 passing compiler/validation tests and a strict TypeScript 7 build. Tests cover stable/disjoint IDs, bindings, finite geometry, canonical node overlap, theme/direction variants, compound group boundaries, label placement, long actor labels across journey rows, Unicode/lengths and explicit rejection of unsupported Mermaid syntax.

The package produces editable native elements and passes the shared scene validator. Plain Node cannot directly import the installed Excalidraw renderer, so the implementation has a bounded native normalizer and injected official browser conversion adapter. D4 must prove actual renderer parity and exports. No live MCP or browser collaboration claim is made here.

Dependency/provenance decisions, copied licenses and identical human/public Markdown guides are recorded. Scoped npm audit found 14 inherited advisories in the Excalidraw subtree (6 high, 6 moderate, 2 low), none for ELK; browser dependency triage remains a release check.

## Workspace migration regression check

After B1 unified Vite/React/jsdom dependency resolution, frontend unit tests again passed (33). Controller reran the two dashboard Chromium suites: 14 passed in 8.0s, including four unchanged visual baselines. The runner emitted its existing conflicting NO_COLOR/FORCE_COLOR warning; there were no application test failures.

## B1 — protocol and isolated candidate validation

Implementation `1967b35` plus fix `a43e194`, task review passed. Final focused suites: 52 runtime tests and 13 protocol tests. Exact frame/document limits, native schema/bindings, candidate isolation, actual worker deadline/heap termination and malformed payloads are exercised.

Review reproduced hidden Yjs map/sequence compartments bypassing materialized checks and BOM-only/prefixed frame IDs becoming empty or aliased. Regression tests first failed, then passed with structural/history checks and canonical UTF-8 ID decoding. Independent re-review approved both fixes. Yjs internals are pinned and must be revalidated on upgrade.

Durable PostgreSQL append/ack, refresh/revocation, browser offline replay and controlled history-limit recovery remain B2/B3/E3. B1 tests do not establish those outcomes. [Dependency remediation tracking](collaboration-dependencies.md) remains part of the release checks.

## A2 — invitation API and atomic onboarding

Commit `1f29438` plus fix `33b1a90`; task review approved. Fifteen targeted invitation tests pass with disposable PostgreSQL, plus one frontend-origin configuration test. Proof covers concurrent single use, seven-day boundary, email binding, no membership downgrade, atomic account/personal-workspace/invited membership creation, rollback, existing password preservation, generic duplicate-email rejection, owner management and bounded HTTP inputs.

A broad intermediate backend run had 106 passes and two failures in the concurrently developing A3 share-link tests. That run is not a passing integration gate; rerun after A3 wiring. Browser invite controls remain C2, production remains v1. Human [invitation guide](../../user/collaboration.md) and public Markdown API guide describe the branch-only behavior.

A2 review found one date-dependent concurrency test: its two acceptors used different clocks. Both now share one immutable fixture clock. Controller repeated the real PostgreSQL concurrency test (one pass); scoped re-review approved the fix. No production-path finding remained.

## E1 dependency patch subset

Commit `b1f2fc9` passed task review. Final root clean install and resolved dependency tree agree; runtime 52, contracts 13, diagram 40 and frontend 33 tests pass, as do production/diagram builds and 14 Chromium flows with four unchanged goldens. Audit entries decreased from 29 to 17; [reachability and unresolved checks](collaboration-dependencies.md) remain explicit. KaTeX browser trust, legacy Socket.IO removal and root-workspace Docker parity are still release checks.

Baseline warning follow-ups: Vite reports existing large chunks; Playwright's environment reports NO_COLOR/FORCE_COLOR conflict. These warnings do not fail the verified commands and did not change visual goldens. Their cleanup is separate from dependency remediation; no pristine-output claim is made.

## A3 — independent public links and room tokens

Commit `870e73f`, task review approved with only a password-limit documentation correction. Sixteen targeted tests pass with actual PostgreSQL and HTTP wiring. Coverage includes migrated legacy links, independent view/edit grants, purpose-separated room credentials, renewal, current membership/link revision, revoke rollback, exact expiry after authority-lock waits, credential-safe logs and POST-only password exchange.

The final broad run had 118 passes and eight failures from concurrent A4 capability stubs. It is not a passing whole-backend integration gate. C2 must replace the old browser query-password request; B2 implements socket revocation/persistence. Production remains v1.

## A4 — scoped asset and mutation authority

Commit `014ef8c` plus fix `4bf3f65`; task review approved after one fix round. Thirty focused tests pass without skips across seven packages: current scene access, public grant expiry/revocation, same-workspace foreign-scene isolation, signed Content-Type/Content-Length, guest uploader audit, private HTTP route isolation and fail-closed secrets. Actual PostgreSQL contention proves authority waits behind revocation then denies; B2 separately owns actual append rejection.

The broad intermediate run has 133 passing and four failing records (including subtests) in D1 incomplete scope and B2 old fixture schema paths. B2 has since moved those fixtures to isolated migration9 schemas; final whole-suite proof remains required. D1 must enable exact-scope agent authority after its own resolver tests. Live object-store PUT, public proxy and cross-network TURN remain E checks.

C2 public transport subset: [browser regression](collaboration-ui.md) reproduces account-token clearing by a public401 and password queries. Commit `ebd66cf` fixes both; three Chromium flows and production build pass. Full C2 management/share/route review remains pending.

### A4 fix round 1 and current backend integration

Commit `4bf3f65` fixes absolute asset expiry across database/cache/signing work and safe same-file renewal. Delayed database and actual AWS SDK credential-provider tests first reproduced overlong URL lifetimes; the fix checks the actual signed expiry. Real PostgreSQL renewal preserves asset ID/key/original audit and returns 409 for each incompatible immutable metadata field. Independent scoped re-review approved both fixes; zero open findings.

Full Go suite after A4/B2/D1 integration: 158 pass records, zero failures, zero skips, exit 0. Fixtures are disposable local PostgreSQL schemas. This supersedes earlier intermediate broad failures above; it does not prove live object-store PUT or deployment.

## D1 — scoped keys and native MCP transports

Commit `00edde1`, task review pending. Seven real PostgreSQL agent tests cover independent scopes/current owner/targets, hash-only storage, room and authority grants, HTTP management, audit/log secrecy and the owner-revocation lock race. Official SDK HTTP and stdio tests cover discovery/calls, origin rejection, revoked keys, session principal binding and encoded response limits. Frontend settings tests verify owner gating and one-time secret clearing.

Worker's final root run: 90 collab + 6 MCP + 13 contracts + 40 diagram tests pass; frontend 35 tests and frontend/MCP builds pass. MCP production audit reports zero vulnerabilities; inherited frontend advisories remain in dependency triage. D1 reads the Go materialized snapshot, not current durable sequence; D3 must wire the current durable projection before mutation preconditions. C2 must expose settings in dashboard navigation. No production/MCP-client deployment claim.

## B2 — durable WebSocket runtime

Commit `1dda631`, task review pending. Final clean install/root tests pass: runtime 92, MCP 6, contracts 13, diagram 40 (151 total); frontend 35 tests and production build pass. Full Go suite passes with all legacy repository tests moved to guarded isolated PostgreSQL schemas. Existing large-chunk build warning remains.

Forty new runtime tests cover actual database termination with no ACK/broadcast, committed-but-response-lost retry, permanent receipts after compaction, workspace lock races/current ACL/expiry, exact 100-update and five-second compaction, listener loss/readiness recovery, single process ownership, private HTTP isolation/backpressure and a built-server/worker subprocess. Legacy promotion preserves the original JSON/version and resolves images only from same-scene assets; invalid promotion leaves v1 untouched. A valid >64 MiB retained-history recovery export is reconstructed without silently changing document identity.

These tests use actual PostgreSQL and sockets with a controlled Go authority fixture. Full Go+Node+browser convergence/load, actual restore and production remain B3/E gates. Root lock incorporates coordinated D1 dependencies. No v2 deployment performed.

### B2 review fix round 1

Commit `afd16b2` addresses three reproduced findings: private commands after owner/LISTEN loss, dirty trash repeatedly disrupting compaction, and unbounded overlapping scans. Transactions now use a shared handoff barrier and verify the exact owner backend; replacement readiness waits for the exclusive barrier. A deterministic pre-COMMIT loss test retains committed receipts without old ACK/publication. Trash/racing deletion/restart/restore preserve data and isolate other rooms. Compaction uses single-flight bounded scans and deduplicated jobs.

An intermediate full run had 104 passing runtime assertions but failed from an unhandled PostgreSQL57P01 during listener shutdown. The lifecycle fix was verified; final root `npm test` exits0 with 104 runtime,13 MCP,13 contracts,40 diagram tests (170 total), no skips/unhandled errors. Scoped re-review pending. No production deployment.

### D1 review fix round 1

Commit `7a0fecd` adds migration13 authoritative Go operation audit and a bounded fsynced adapter-observed journal. Successful Go create and audit commit atomically; failure records omit credentials/content. HTTP requires a persistent journal path; stdio observations remain explicitly client-controlled. Upstream413 retains actionable PROJECTION_TOO_LARGE. Ten targeted real PostgreSQL tests,13 MCP HTTP/stdio/journal tests and full Go suite pass; scoped re-review pending. D3 must reuse correlation/audit wrappers.

## C2 — access workflows and deep links

Commit `ce386b4` plus earlier `ebd66cf`, review pending. Final24 Chromium flows,35 frontend unit tests and production build pass; full Go suite passed after C2 backend edits. Browser proofs use controlled API fixtures, while separate PostgreSQL tests prove current computed role/access and new public-link snapshot authorization. UUID/alias reload/back/forward/home, invitation login/register/wrong email/expiry, independent links/password/revoke, viewer/owner controls and login-token isolation are covered. Three golden images were inspected and updated for owner settings navigation; original dashboard colors/style and native canvas style remain.

A broad route mock accidentally intercepted the Vite share-links module; narrowing to API paths fixed the harness without another product change. New-link snapshot production routing did need a real fix: canonical scene scope is derived from validated session and authority rechecked under lock. Public live canvas editing, session renewal and live revoke remain B3/E gates; its current guarded read-only seam is not presented as completed realtime UI.

### Review closure checkpoint

D1 transport audit fix `097fd47` passed scoped spec and quality review: real HTTP and raw stdio invalid tool arguments produce one correlated start/completion pair; valid operations are not duplicated and raw arguments/credentials/SDK error text are excluded. MCP15 and full root173 tests passed. B2 startup fix `197f6cd` has40 relevant tests passing; C2 navigation fix `803abd8` has29 Chromium,35 unit and build checks passing. Their scoped reviews are pending at this checkpoint. No v2 production deployment yet.

B2 startup and C2 navigation scoped reviews are now clean. B2 reviewer independently reproduced40passing tests; C2 reviewed route/history/stale-response assertions and29browser/35unit/build evidence. Dependency declarations for B3/D3 installed with npm; npm ls confirms shared pinned contracts/diagram/Yjs/lib0 and fake-indexeddb6.2.6, no duplicate Yjs. VM100 read-only check still shows both v1 whiteboard containers healthy.

### B3/D3/D4 integration in progress

Diagram-kit built ESM prerequisite `89b1b9c` passes41tests, including normal Node import of19presets. Root npm graph resolves a single Yjs13.6.33 and KaTeX0.18.2 after targeted lock correction. D3 actual MCP SDK→Go→private Node→PostgreSQL tests cover live sequence, retries/conflicts and scoped writes; a real WebSocket replica verifies matching IDs and preserved human edits, not browser visual UAT. Worker full Go gate passed and root npm187tests passed (runtime105/MCP28/contracts13/diagram41), including in-progress D4 tests. These are implementation evidence awaiting task reviews. B3 browser/runtime fixture will explicitly identify mocked metadata/login boundaries; full E1auth/assets/voice release acceptance remains required.

B3 focused browser/runtime proof now passes1/1 (6.4s): two native canvases, actual WebSocket/Node/Go authority/PostgreSQL; native rectangle persists and converges, offline edit enters durable IndexedDB queue without advancing server sequence, online reload replays and second browser converges. HTTP shell metadata/login/room-token handoff remain fixtures, not full production API acceptance. Interleaved committed update before final sync chunk has RED→GREEN coverage. Image/recovery completion and fresh B3 review pending. Controller diagnosed test keyboard focus before selection, with unchanged product code proving native draw; no UI failure was dismissed without evidence.


### D3/D4 frozen review checkpoint

#### D4 Chromium startup endpoint race — 2026-10-19

The final workspace gate reproduced a renderer startup race: Chromium can create `DevToolsActivePort` after writing its numeric port but before writing the second endpoint-path line. Treating that partial file as permanently malformed produced intermittent `RENDER_FAILED` during otherwise valid MCP exports and lifecycle tests. A deterministic fake browser now writes the two lines in separate steps and proves the old behavior fails. The worker polls partial content within the existing wall deadline, proceeds only after both lines validate, and retains the same process ownership, timeout, cleanup and fail-closed rules. Public API/config/schema behavior is unchanged; this is internal renderer reliability evidence.

Commit `98be44b` adds native renderer/export and D3 review corrections to `d0c71fc`: numeric activity/recovery ordering across102events, explicit public frontend origin and cold build instructions. Final root workspace gate passes196tests (runtime105, MCP37, contracts13, diagram41), all Go packages and server build pass. Native renderer tests include isolated deadline termination, actual Chromium active-loop termination, Unicode/font containment, network denial, KaTeX0.18.2 hostile-input fixtures, current export permission and revoke/download checks. Independent review is active, including browser startup process cleanup before PID reporting. No release approval or production deployment follows from this checkpoint.

E2 local Linux/arm64 image builds and nginx routing proof are separate from final frozen-source Linux/amd64 release images. Actual two-network generated audio passed with the gateway IP; the configured TURN hostname failed due to local split DNS. See [packaging evidence](collaboration-v2-packaging.md) for both results and fixture boundaries.

D3 scoped re-review of `98be44b` passed all three original findings. D4 review reproduced two lifecycle defects outside that correction: a browser spawned before its PID is reported survives startup timeout, and successful renders leave Playwright temporary directories. Original renderer implementer owns fix round1 with focused process/storage cleanup regressions. D4 is not accepted until re-review.

B3 commit `f55b6b5` records56unit tests,30Chromium tests and production build passing. Final live test additionally verifies remote/reload Undo disabled, native restoration preserving authoritative null bindings, and no server sequence advancement after replay settles. Preview Blob URL lifecycle regression passed3repeats after3reproduced failures. Independent B3 review remains active. E2 app `8877124` and Homelab `28076d8` are packaging snapshots under independent review; neither is a production rollout.


D4 lifecycle fix `6c64b12` passed scoped spec/quality review: both original findings addressed, reviewer independently ran5lifecycle tests successfully. Implementer24focused tests/build and controller nonroot Linux/arm64 nativePNG image smoke passed. Exact image and fixture boundaries are recorded in [packaging evidence](collaboration-v2-packaging.md). Final Linux/amd64 release acceptance remains outstanding.


E2 scoped heartbeat re-review passed against `d625d44`:25second server ping, next-interval dead-peer termination, presence/voice cleanup and160second actual nginx proof address the75second idle disconnect. Full B4 server/frontend task review remains separate. E2 packaging is accepted within its explicit local ARM64/signed-authority-fixture scope; final frozen AMD64/public release gates remain.


### Real-stack acceptance and recovery checkpoint

B3 fix3 `3a20bb5` passes34collaboration tests and the frontend build; independent scoped review is pending. The same-account draft-conflict upload loop is corrected, while a separately reproduced pending-image movement regression remains a release blocker.

The actual Go/Node/PostgreSQL/MCP/browser/S3 harness is running. The measured50-client workload currently **fails**:650attempts,178ACKs,219durable receipts, observed ACKp958557ms. Every observed ACK matches its receipt; this does not imply all attempted updates completed. All50sockets eventually closed1013. Profiling identifies per-update fresh validation worker startup under the workspace lock and per-commit/per-recipient full reads; repair and remeasurement are required. The target remains p95below1second at50clients each sending1smallupdate/second.

Recovery slice `93d57a1` passes24tests and compilation; independent review is pending. It preserves original history in an archive and a quarantined scene and creates a distinct replacement scene/document. Unsupported assets and access overrides refuse before mutation. This is not a full PostgreSQL backup restore or a production rollout. See [recovery evidence](collaboration-recovery.md) and [operator runbook](../../operations/collaboration-recovery.md).


B3 fix3 scoped review now passes both spec and quality (13independent targeted checks); B4 frontend integration is released while the separate image-movement repair remains required. The recovery slice also passes independent spec/quality review with24actual checks. Two nonblocking recovery hardening/coverage suggestions remain for whole-branch triage; full E3 acceptance is still open.


E1 runtime repair `fe4bb5b` passed independent spec/quality review:58focused tests and a separate merged-frame-size/convergence probe passed; reviewer verified all650raw ACK/receipt pairs and50converged replicas. Quiet measured ACKp9513.268ms, remote deliveryp95776.186ms, peak runtimeRSS400,424,960bytes on the documented localARM64host. Final error-path capacity adjustment passed149runtime tests after measurement; exact-head load remains required with final E1 acceptance. Current non-load actual-stack browser suite11/13passed; asset viewer fetch and offline-helper cases remain under investigation.
