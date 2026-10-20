# Collections agent reference

Canonical browser route: `/w/{workspace}/collections/{collectionId}`.

API root: `/api/v1/workspaces/{workspace}/collections`. Authenticate with a
login bearer token. The workspace may be a UUID or public alias. Mutation bodies
are strict JSON and use a new UUID `requestId`. Revisioned bodies use the exact
decimal-string `expectedRevision` returned by the preceding response.

Operations:

- `GET root?limit=1..50&cursor=<collection UUID>` returns `{items,nextCursor?}`.
- `POST root` accepts `{requestId,name,visibility}`.
- `GET|PATCH|DELETE root/{id}` reads, updates, or removes a container.
- `POST root/{id}/scenes` accepts `{requestId,expectedRevision,sceneId}`.
- `DELETE root/{id}/scenes/{sceneId}` accepts `{requestId,expectedRevision}`.
- `PUT root/{id}/order` accepts `{requestId,expectedRevision,sceneIds}` and the
  array must contain exactly the collection's current scenes once each.

`visibility` is `workspace` or `private`. Private containers are creator-only.
Responses contain `mine` and `canSetPrivate` booleans and do not expose creator
identity. Only a caller with `canSetPrivate:true` may change a workspace
collection to private.
Workspace names are unique case-insensitively per workspace. Private names are
unique case-insensitively per creator and workspace; conflicts return
`409 COLLECTION_NAME_CONFLICT`.
Returned scene rows and `sceneCount` reflect current scene authority before the
response page is formed. Collection membership does not grant scene authority.
Deleting or changing a collection never moves or deletes scenes.

Remove checks current scene authority before testing collection membership.
Reorder accepts the exact currently accessible subset; under the same locked
transaction the service prunes stale rows for deleted or no-longer-visible
scenes, validates the remaining set, then assigns stable positions. This avoids
both hidden membership oracles and permanently blocked ordering.

An exact retry is bound to the operation, principal, and payload. Receipt replay
reauthorizes current workspace, collection, and referenced-scene access under
the workspace lock, then rebuilds the response from current filtered state.
Reuse the same `requestId` after a transport failure until an HTTP response or
explicit cancellation; cross-operation reuse returns `REQUEST_ID_REUSED`.

Conflict codes: `REVISION_CONFLICT`, `REQUEST_ID_REUSED`,
`COLLECTION_NAME_CONFLICT`. Policy denial:
`FEATURE_DISABLED`. Inaccessible workspaces, private collections, foreign
scenes, and inaccessible collection membership do not expose foreign content.
