# Enterprise feature policy

## Authority flow

```text
Authenticated principal
        |
        v
Existing workspace lock -> current scene/workspace access
        |                         |
        v                         v
workspace_feature_policies   active share-link revision
        |                         |
        +-----------+-------------+
                    v
              FeatureAction
                    |
                    v
          Casbin role/action policy
                    |
                    v
feature_mutation_receipts -> feature transaction callback
```

Go owns policy and metadata retry authority. Node remains the only canonical
Yjs writer. A feature permission never changes the effective scene role.

## Actions and default policy

The stable action catalog is `comment:read`, `comment:write`,
`comment:moderate`, `activity:read`, `version:read`, `version:create`,
`version:restore`, `library:read`, `library:write`,
`collection:manage`, `presentation:host`, `screen:publish` and
`enterprise:manage`.

New workspaces receive policy revision `1`. `memberComments` and
`memberHistory` default to true. Runtime feature flags default to false until
their implementation slice is enabled. Public links default
`allowComments=false`; link comment permission can be selected independently
of view/edit access and is invalidated with the link revision.

The share-link dialog exposes this opt-in on both view and edit links and shows
the stored comment permission for every existing link. Leaving it unchecked
preserves the secure default.

## HTTP contract

`GET /api/v1/workspaces/{workspace}/feature-policy` returns the current
workspace policy. `PATCH` is owner-only and requires a UUID `requestId`, a
canonical decimal-string `expectedRevision`, and at least one policy field.
Unknown fields are rejected.

Successful responses contain a decimal-string `revision`. Stale revisions
return `409 REVISION_CONFLICT`. Reusing a request ID with different canonical
mutation bytes, principal, action or target returns
`409 REQUEST_ID_REUSED`.

## Data model

| Table | Key | Purpose |
| --- | --- | --- |
| `workspace_feature_policies` | `workspace_id` | Versioned member preferences and rollout flags |
| `feature_mutation_receipts` | `workspace_id, request_id` | Exact response bytes and retry binding |
| `share_links.allow_comments` | share link | Guest comment permission independent of view/edit |

Receipts use the composite workspace/scene foreign key when a scene is present.
Workspace policy updates bump their own revision; share-link changes continue
to bump link revision and workspace ACL through the existing trigger.

## Verification

- Unit: feature role/action matrix and non-escalation.
- PostgreSQL: exact retry, changed-byte rejection, revocation inside a locked
  mutation and policy retry revision.
- HTTP: canonical decimal revision and strict request decoding.
