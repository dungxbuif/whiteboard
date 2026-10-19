# E3 disposable collaboration recovery evidence

Status: in_review. Bounded recovery slice verified locally on 2026-10-16; full E3, production deployment and production restore are not claimed.

Scope/design: [operator runbook](../../operations/collaboration-recovery.md), [E3 release task](../../superpowers/plans/2026-10-16-collaboration-release.md), [rehearsal instructions](../../../tests/recovery/README.md). The controller approved replacement-scene recovery; same-scene epoch reset is unsupported.

## Commands and results

Executed from the worktree root `/Users/dungxbuif/workspace/devtool-worktrees/collab-v2`:

```sh
rtk proxy env COLLAB_TEST_DATABASE_URL='postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable' RECOVERY_EVIDENCE_DIR='/Users/dungxbuif/workspace/devtool-worktrees/collab-v2/.superpowers/sdd/2026-10-16-collaboration-v2/e3-recovery-evidence' /Users/dungxbuif/.nvm/versions/node/v24.0.2/bin/node tests/recovery/rehearse.mjs
```

Result: exit 0; runtime TypeScript compilation passed; 2 test files, **24 tests passed** (16 archive unit cases and 8 guarded database/runtime cases). Vitest start 23:51:01 local; duration 852 ms. The script uses existing installed dependencies and did not run npm/install.

Unsafe target check:

```sh
rtk proxy env COLLAB_TEST_DATABASE_URL='postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable' /Users/dungxbuif/.nvm/versions/node/v24.0.2/bin/node tests/recovery/rehearse.mjs
```

Result: exit 2 with the explicit disposable-target requirement, before building or accessing a database. `rtk git diff --check` passed.

RED evidence: archive positives failed against unimplemented functions before implementation (8 failures); rehearsal functions likewise failed all 8 integration cases before implementation. A subsequent meaningful receipt-vector regression failed because a forged latest acknowledgement vector was initially accepted; the implemented comparison passed in the final 24-test run.

## Controlled drill evidence

| Check | Observed result |
| --- | --- |
| Actual runtime | PostgreSQL migrations 1–9, Persistence, Yjs, runtime advisory ownership, real WebSocket ack/admission |
| Drain | Checkpoint refused a live writer; server close drained and closed the connected socket; stopped store rejected writes with UNAVAILABLE |
| Latest state | Native element `retained-element` retained; compacted update1 plus remaining update2 yielded `viewBackgroundColor=#abcdef` |
| Checkpoint | Latest sequence2, snapshot sequence1, one ordered remaining update, two receipts, two audit rows, original promotion JSON at contentVersion7 |
| Archive | Exclusive 0600 file, fsync + directory fsync, read-back integrity and latest receipt state-vector comparison |
| Original scene | Trashed/quarantined; document/log/receipts/audit retained, no cascaded history deletion |
| Replacement | New scene UUID materialized as v1, then promoted to sequence0 with a new documentId and equal native content |
| Old epoch | Old append rejected FORBIDDEN, old socket admission returned FORBIDDEN + resync_required and no ready/ack; no queued request receipt appeared |
| Restore | Original archive restored into a separate random schema; complete checkpoint digest matched exactly |
| Refusal cases | Wrong DB host/port/name/connection override; asset metadata; scene overrides; stale checkpoint; corrupt or incomplete archive |

Recorded checkpoint digest: `79face3e7ba130a9c42ec0cc3f912d81bbccfb03458a30f69745f166addcec34`.

- Original scene: `5727a689-858b-498e-8aa5-100e25fdebaa`; replacement: `d3bae3d4-49b8-461b-93fd-cb6e4e29cc33`.
- Original document: `f44d91e6-06e1-4c68-be03-648144af22b4`; replacement: `d568cdcf-e95b-4135-ae02-a032c2016325`.
- Synthetic checkpoint and result files are retained privately in the worktree's `.superpowers/sdd/2026-10-16-collaboration-v2/e3-recovery-evidence/`. Test schemas were cleaned up.

## Boundaries and documentation reconciliation

Asset-bearing scenes and scenes with access overrides remain unsupported and fail before any source mutation; no image restoration success is claimed. Original-scene Trash restore must remain prohibited by the operator procedure because it would reactivate the original identity. Recovery does not copy links, public tokens, guest sessions or targeted agent grants and does not offer same-URL continuity. Owners explicitly update external references after reviewing replacement access.

This test retains an old pending packet unchanged and proves server-side denial. Browser IndexedDB persistence, client identity switching and UI recovery belong to B3 and are not inferred from this fixture. Go authority HTTP is the existing fixture's signed-decision boundary; database tenancy/ACL-version/deleted-scene checks and runtime cryptographic verification remain real. This is scene-checkpoint restore equivalence, not a full database/object-storage backup or migrations 1–13 rollout proof.

Internal operator docs are updated in the runbook. Central context/backlog/validation/release reconciliation and public human/AI-readable limitation notices are controller-owned. No schema, HTTP API, environment variable or normal user flow changed, so this slice needs no OpenAPI or configuration reference change. Independent review remains required before integration.
