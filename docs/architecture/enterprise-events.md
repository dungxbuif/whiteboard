# Enterprise workspace events

Status: foundation implemented under the enterprise feature programme. User-facing
activity routes and panels remain owned by task C3 and are not advertised yet.

## Purpose and ownership

Workspace events are a safe metadata projection for comments, access changes,
agent operations, durable collaboration mutations, and later enterprise jobs.
They are not a copy of scene content or a replacement for canonical Yjs history.

- Go metadata writers append an event inside the same transaction as the state
  change by using the transaction-bound context. A direct repository call
  without such a context creates its own transaction, so the event and outbox
  row can never commit separately.
- Go projects `acl_audit_events` and `agent_operation_audit` in bounded batches.
- The Node collaboration owner projects `collab_mutation_audit`; it remains the
  sole canonical Yjs writer.
- Projectors keep persistent `(cursor_time,cursor_key)` observability state, but
  select source rows by anti-joining the unique `(source,source_key)` event
  provenance. The cursor is never an exclusion boundary: a transaction that
  commits late behind the high-water mark is still projected. A crash before
  commit advances neither the cursor nor the event, and a retry cannot duplicate
  the event or outbox notification.

## Data model

```text
workspace_events
  id bigint                   decimal JSON cursor
  workspace_id uuid           tenant boundary
  scene_id uuid nullable      composite workspace/scene FK
  actor_kind                  user | public | agent | system
  actor_id text
  kind text
  subject_id text             opaque stable identifier
  occurred_at timestamptz
  safe_metadata jsonb         object, at most 4 KiB
  source + source_key         unique projection provenance

workspace_event_outbox
  event_id bigint             one row per committed event
  workspace_id + scene_id
  delivered_at + attempts     worker delivery state

workspace_event_source_cursors
  source primary key
  cursor_time + cursor_key
```

`safe_metadata` is a flat positive allowlist. Identifier fields must be UUIDs;
revision, sequence and timing fields must be numeric; action, outcome, scope and
status fields must be bounded tokens. Unknown, nested or free-form fields are
rejected, including compound credential/content names such as `access_token`,
`rawBody` and `iceCandidate`. Projectors copy only these validated identifiers,
tokens and bounded numbers.

## Invalidation protocol

An outbox insert emits PostgreSQL `NOTIFY feature_changed` only after commit.
The collaboration runtime listens on its existing database listener and sends
the following cursor-only control message over the authenticated `/collab/`
socket:

```json
{"type":"feature_changed","requestId":"server","payload":{"eventCursor":"42"}}
```

The runtime rechecks current scene authority before each send. Foundation fanout
is limited to authenticated user principals; guest comment fanout is added only
with task C1's explicit feature-policy authorization. Invalidations contain no
event kind, actor, subject, title, or content. They may be dropped: clients
refetch authorized pages after reconnect, route focus, or manual refresh.

## Paging and authorization

`EventService.ListWorkspace` accepts authenticated workspace users, a decimal
cursor, and a maximum page size of 50. It checks current workspace membership,
filters every scene event through current scene authority before returning it,
and withholds `security.*` and workspace ACL/admin activity from non-owners.
Public guests and agents do not receive a workspace feed. C2 consumes comment
events through idempotent, current-authority notification jobs; task C3 owns
the activity HTTP handlers and human-facing activity guide.

## Verification

- Go: positive metadata allowlist, injected outbox failure rollback, identical
  retry/no-second-outbox, late-commit and stale-cursor recovery, owner-only ACL
  activity, and restricted-scene filtering.
- Node: crash-before-commit retry, late-commit/stale-cursor recovery, bounded
  batch validation, recipient reauthorization, and cursor-only fanout.
- Frontend/contracts: strict cursor-only parsing and transport delivery without
  changing document admission.

Trace: [enterprise phase](../work/phases/PHASE-ENTERPRISE.md),
[approved design](../superpowers/specs/2026-10-20-enterprise-parity-design.md),
[foundation task F2](../superpowers/plans/2026-10-20-enterprise-foundation.md).
