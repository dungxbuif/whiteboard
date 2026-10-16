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

Implementation commit `309bf08` plus fix `eaa3c7d`; task review passed after one fix round. Production remains v1.

- Targeted access suite: 16 tests passed against disposable local PostgreSQL 15. Full backend JSON result: 79 tests passed, 0 failed, 0 test skips; four packages have no tests.
- Proof includes owner backfill and legacy member-to-editor migration, UUID/alias compatibility, unique aliases, role matrix, current-membership overrides, owner protection, deleted resource denial, list/search isolation, protected folder descendants and scene-scoped assets.
- Two database connections demonstrate authority row contention: a competing revocation cannot pass the lock during the authorized write transaction; rollback and subsequent denial are checked.
- Regression tests first reproduced incorrect alias handoff, cross-scene asset reads and missing status invalidation. Final targeted/full suites and diff whitespace checks passed.
- API/ERD/ADR document branch behavior. Public invite/share/room guides and browser role controls await A2–A4/C2 integration; no complete collaboration or production claim follows from A1.
- Remaining integration: return effective role/access to the frontend, use publicId aliases in App routing, implement public/agent principal resolvers and replace owner-only v1 Socket.IO.
- Review reproduced an unbounded request-body read holding workspace authority. Fix buffers bounded input before DB/locks (10s deadline, 26MiB scene envelope, 1MiB metadata). A real stalled TCP client no longer blocks owner revocation and receives 408; 14 scoped tests pass. Independent re-review approved the fix.

## D2 — native diagram compiler

Implementation commit `8b2fd90` plus fix `3f532e6`, task review passed. All 19 presets are covered by 40 passing compiler/validation tests and a strict TypeScript 7 build. Tests cover stable/disjoint IDs, bindings, finite geometry, canonical node overlap, theme/direction variants, compound group boundaries, label placement, long actor labels across journey rows, Unicode/lengths and explicit rejection of unsupported Mermaid syntax.

The package produces editable native elements and passes the shared scene validator. Plain Node cannot directly import the installed Excalidraw renderer, so the implementation has a bounded native normalizer and injected official browser conversion adapter. D4 must prove actual renderer parity and exports. No live MCP or browser collaboration claim is made here.

Dependency/provenance decisions, copied licenses and identical human/public Markdown guides are recorded. Scoped npm audit found 14 inherited advisories in the Excalidraw subtree (6 high, 6 moderate, 2 low), none for ELK; browser dependency triage remains a release check.

## Workspace migration regression check

After B1 unified Vite/React/jsdom dependency resolution, frontend unit tests again passed (33). Controller reran the two dashboard Chromium suites: 14 passed in 8.0s, including four unchanged visual baselines. The runner emitted its existing conflicting NO_COLOR/FORCE_COLOR warning; there were no application test failures.

## B1 — protocol and isolated candidate validation

Implementation `16e1abf` plus fix `4eb3242`, task review passed. Final focused suites: 52 runtime tests and 13 protocol tests. Exact frame/document limits, native schema/bindings, candidate isolation, actual worker deadline/heap termination and malformed payloads are exercised.

Review reproduced hidden Yjs map/sequence compartments bypassing materialized checks and BOM-only/prefixed frame IDs becoming empty or aliased. Regression tests first failed, then passed with structural/history checks and canonical UTF-8 ID decoding. Independent re-review approved both fixes. Yjs internals are pinned and must be revalidated on upgrade.

Durable PostgreSQL append/ack, refresh/revocation, browser offline replay and controlled history-limit recovery remain B2/B3/E3. B1 tests do not establish those outcomes. [Dependency remediation tracking](collaboration-dependencies.md) remains part of the release checks.

## A2 — invitation API and atomic onboarding

Commit `9e1ec91` plus fix `873ce82`; task review approved. Fifteen targeted invitation tests pass with disposable PostgreSQL, plus one frontend-origin configuration test. Proof covers concurrent single use, seven-day boundary, email binding, no membership downgrade, atomic account/personal-workspace/invited membership creation, rollback, existing password preservation, generic duplicate-email rejection, owner management and bounded HTTP inputs.

A broad intermediate backend run had 106 passes and two failures in the concurrently developing A3 share-link tests. That run is not a passing integration gate; rerun after A3 wiring. Browser invite controls remain C2, production remains v1. Human [invitation guide](../../user/collaboration.md) and public Markdown API guide describe the branch-only behavior.

A2 review found one date-dependent concurrency test: its two acceptors used different clocks. Both now share one immutable fixture clock. Controller repeated the real PostgreSQL concurrency test (one pass); scoped re-review approved the fix. No production-path finding remained.

## E1 dependency patch subset

Commit `3754f1b` passed task review. Final root clean install and resolved dependency tree agree; runtime 52, contracts 13, diagram 40 and frontend 33 tests pass, as do production/diagram builds and 14 Chromium flows with four unchanged goldens. Audit entries decreased from 29 to 17; [reachability and unresolved checks](collaboration-dependencies.md) remain explicit. KaTeX browser trust, legacy Socket.IO removal and root-workspace Docker parity are still release checks.

Baseline warning follow-ups: Vite reports existing large chunks; Playwright's environment reports NO_COLOR/FORCE_COLOR conflict. These warnings do not fail the verified commands and did not change visual goldens. Their cleanup is separate from dependency remediation; no pristine-output claim is made.

## A3 — independent public links and room tokens

Commit `f6d61b5`, review pending. Sixteen targeted tests pass with actual PostgreSQL and HTTP wiring. Coverage includes migrated legacy links, independent view/edit grants, purpose-separated room credentials, renewal, current membership/link revision, revoke rollback, exact expiry after authority-lock waits, credential-safe logs and POST-only password exchange.

The final broad run had 118 passes and eight failures from concurrent A4 capability stubs. It is not a passing whole-backend integration gate. C2 must replace the old browser query-password request; B2 implements socket revocation/persistence. Production remains v1.
