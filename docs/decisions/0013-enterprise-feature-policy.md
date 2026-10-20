# ADR 0013 — Workspace feature policy and idempotent metadata mutations

Status: accepted for implementation on 2026-10-20.

## Context

Enterprise comments, activity, versions, libraries, live presentation and
identity features share the existing workspace and scene authority boundary.
Each feature needs current-role checks and safe retries without granting broader
scene edit rights. The approved enterprise design requires decimal revisions,
principal-bound UUID request IDs and staged feature flags.

## Decision

Go remains the authority for enterprise metadata. A single
`workspace_feature_policies` row belongs to every workspace and starts with
member comment/history preferences enabled while runtime feature flags remain
disabled. Feature actions resolve the current scene decision inside the existing
workspace lock, then pass the effective owner/editor/viewer/public role and
requested action through a dedicated Casbin policy before feature flags and
member preferences are applied. Public link comments additionally require that link's
`allow_comments` flag and current link revision. Agent keys receive no new
implicit feature rights.

Metadata mutations use `feature_mutation_receipts`, keyed by workspace and
request UUID. The receipt binds scene, feature action, principal kind/ID and a
SHA-256 payload hash. An identical retry returns the exact stored response
bytes; a changed binding returns `REQUEST_ID_REUSED`. Policy writes also use
an expected decimal revision.

## Consequences

- Comment permission cannot become scene edit permission.
- Revocation is checked under the transaction-bound workspace lock.
- Retries are deterministic across process restarts.
- Feature flags support staged rollout without changing existing access.
- The role/action matrix is declarative and can be exhaustively tested against
  current PostgreSQL authority rather than duplicated as action branches.
- Receipt storage is durable metadata and requires retention policy review
  before any future deletion.

## Alternatives

- Separate policy checks in each module were rejected because they drift.
- Storing request IDs only in memory was rejected because restart loses retry
  safety.
- Encoding comment permission in scene content was rejected because moderation,
  retention and share-link revocation are metadata authority concerns.

## Trace

- [Enterprise design](../superpowers/specs/2026-10-20-enterprise-parity-design.md)
- [Foundation plan F1](../superpowers/plans/2026-10-20-enterprise-foundation.md)
- [Feature policy architecture](../architecture/enterprise-feature-policy.md)
- Migration `000014_enterprise_feature_policy`
