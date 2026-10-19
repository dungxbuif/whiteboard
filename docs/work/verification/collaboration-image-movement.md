# E1 pending-image movement regression

Status: reproduced observation; bounded repair design in review. P2 release blocker, separate from B3 fix3 scoped review and overall E1 acceptance. Controller owns central backlog/release records.

## Intended behavior and affected flow

A locally inserted image must retain later local movement while its PUT or durability flush is pending. Merging must apply only fields changed since a known local baseline and preserve independent remote style/position changes, tombstones, and unrelated same-ID insertions. Blob retention, revision comparison and lifetime fences remain unchanged. Internal affected docs: this verification/design note and task-E1-image-report.md; public/AI guidance changes will be reconciled once the final behavior is determined. No schema/API/runtime/dependency changes are proposed.

## Root cause and design constraint

The pending draft retains the base from before insertion. Once its image has been projected at x90, a subsequent desired x270 is merged against that empty base and an existing current image. src/frontend/src/collab/scene-diff.ts:9 treats this as an addition colliding with an existing ID and preserves x90. This protects unrelated same-ID additions, so replacing that branch with wholesale overwrite would break conflict preservation. The current three-input stateless merge has no knowledge of our prior projection or per-field original values. Seed/fileId equality is not ownership proof.

No product edits before controller coordination: a sound fix needs an explicit known applied local baseline/provenance, likely maintained by the wrapper and recoverable across pending persistence, or another approved equivalent. Frozen B3 wrapper/offline-store files remain untouched.

## Verification plan

Start from preserved /tmp/b3-fix3-image-move.test.tsx. Add a maintained one-controller mounted regression: image insertion then movement during held PUT, and during held durability flush. Use actual adapter, IndexedDB and upload helper; control native canvas callbacks/network/flush boundaries. Add scene-diff tests for field-wise remote preservation and unrelated same-ID insertion conflicts. Record RED before implementation, then focused GREEN and build after an approved repair. These tests do not substitute for E1 real-browser/S3 acceptance.

## Maintained RED — 2026-10-17 00:02 local

`rtk proxy env PATH=/Users/dungxbuif/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:/usr/local/bin:/usr/bin:/bin npm run test:unit --workspace frontend -- src/collab/image-movement.test.tsx src/collab/scene-diff.test.ts` (cwd src): 1 failed, 3 passed. During-flush movement expected270, observed90 after matching recovery cleanup. During-PUT movement passes; existing-element field-wise merge and unrelated same-ID addition protection pass. No product files changed for this reproduction.

Controller expanded narrow ownership to wrapper image retry/onChange and draft storage if necessary; B4 holds wrapper, while its provider/toolbar work continues. Proposed durability coordination awaiting confirmation: record a local baseline only after actual application, excluding newly requested IDs that already existed before apply. Preserve local desired values as the diff baseline instead of copying unrelated remote fields from the full merged projection. Each later callback reserves its revision before slow conversion, retains its bytes, awaits the applied update's durability, then conditionally persists that exact revision's rebased draft. Retry awaits the selected draft's preparation; newer controller revisions cannot be overwritten by late baseline work. No database version/store change. Add failure/reload, concurrent remote-field and collision regressions before implementation.

Controller approved the local-intent baseline and conditional rebase approach. Extra interruption boundary under discussion: an optional unfinished-baseline marker preserves the original draft if durability rejects or the owning lifetime ends before preparation finishes. Such a record must never silently apply with the old base and clear recovery on reopening. A normal completed rebase remains replayable; incomplete rebase would require explicit recovery rather than inferred ownership.

Final controller ruling: interrupted baseline preparation uses an optional pendingBaseline marker atomically reserved with the original draft revision. Retain marker and exact Blob bytes on rejection/lifetime loss; reopen refuses automatic projection/cleanup and offers recovery export. Successful conditional matching rebase clears the marker, with normal automatic replay. Marker is included in existing recovery export metadata. No new IndexedDB version/store. This bounded explicit recovery flow avoids inventing causal ownership.

## Implemented behavior and verification

Status: implementation verified locally; independent review pending. advanceLocalBaseline excludes unrelated colliding additions and records local desired fields, so later local moves do not overwrite remote fields merged before or after the first projection. The wrapper records that baseline and the exact durability promise after a successful local apply. New callbacks reserve their original draft and optional pendingBaseline marker before conversion. They retain bytes, await durability, then revision-conditionally update draft/Blob baselines and remove the marker. Retry waits for preparation and refuses unfinished markers before projection or cleanup. Rejected durability or expired controller lifetime retains the original marked recovery; a newer revision rejects the stale rebase. No presence/UI behavior was edited.

Final evidence: focused new tests10/2files pass; current collaboration suite52/16files passes (includes independent concurrent B4 tests). TypeScript/Vite build passes after a single nullable-ref narrowing correction. Native collab-live Playwright regression1/1 passes in5.7s using real Go authority, Node and PostgreSQL; its HTTP-shell delivery fixtures remain mocked and it is not signed-S3 movement proof. Screenshot visually inspected: src/frontend/test-results/collab-live-native-canvas--7b0bd-reload-replays-durable-work-chromium/native-canvas-two-elements.png (two native rectangles, Saved, disabled reload Undo, original styling). Exact commands and attempts are in the task report.

New tests cover movement while PUT and flush wait; remote fields before/after projection; collision preservation; completed-rebase automatic reload; failed durability/lifetime-loss marked recovery retained on reopen and included in export; newer-revision rejection. Existing draft-order and revision-conflict test transports gained only the real whenDurable method in their fixtures. Human guide and public Markdown mirrors document the bounded interruption behavior. No API/schema/runtime/dependency changes or full E1 completion claim.

## Image review fix round 1 — intended local intent accumulation

Review task-E1-image-review.md found observed remote fields included in later native callbacks were compared against the older applied baseline and replayed as local edits. Keep the existing known applied baseline, collision guard, durable preparation marker and revision checks. Before extending a pending draft, derive only the current native before→desired field delta and merge it into the previous pending local-intent snapshot. Observed remote fields that did not change in this gesture must not replace that stored intent. This is a bounded wrapper change; no storage metadata/schema change is needed.

Regression must actually deliver remote green through incoming native projection so lastNative is green, then move locally with a green callback, deliver later remote blue during flush, and retain blue after x270/cleanup. Add a deliberate local style change negative so local edits are not accidentally filtered out. Preserve all prior ordering, interruption and collision tests. Root will snapshot a scratch frozen e82c7ce wrapper plus image-only hunk, excluding concurrent B4 UI and E1 hydration work.
