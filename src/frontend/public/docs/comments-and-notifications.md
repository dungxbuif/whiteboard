# Comments and notification API

## Scene comments

All member requests send `X-Workspace-ID`. Public callers send their current
scene-session bearer token. Public sessions require the link's current
`allowComments=true` revision and the workspace comment feature.

- `GET /api/v1/scenes/{scene}/comments?cursor&limit=50`
- `POST /api/v1/scenes/{scene}/comments` with
  `{requestId,anchor:{elementId?,x,y},body}`
- `GET /api/v1/scenes/{scene}/comments/{thread}?messageCursor&messageLimit=50`
- `POST /api/v1/scenes/{scene}/comments/{thread}/replies` with
  `{requestId,body}`
- `PATCH /api/v1/scenes/{scene}/comments/{thread}` with
  `{requestId,expectedRevision,resolved?,subscribed?}`
- `PATCH|DELETE /api/v1/scenes/{scene}/comments/{thread}/messages/{message}`

Bodies contain plain text only and are limited to 1–4,000 Unicode code points.
Render names and bodies as text. A `409 REVISION_CONFLICT` requires an
authorized refetch; clients must retain the unsent local draft. Message pages
are stable, capped at 50, and continue through `nextMessageCursor`.

Anchors store an optional native Excalidraw element ID and fallback scene
coordinates. Resolve the current element geometry for display. When the ID is
absent from current scene elements, use the fallback and label the anchor
`Element unavailable`; never synthesize an Excalidraw element or retarget it.

## Member notification inbox

- `GET /api/v1/notifications?cursor=<notification UUID>&limit=1..50`
- `POST /api/v1/notifications/{id}/read` with `{requestId:<UUID>}`

The inbox accepts login principals only. Each item contains a server-created
deep link `/w/{workspace}/scenes/{scene}?thread={thread}`. Delivery is unique by
`(user,event)`, excludes the actor, and rechecks current comment-read authority.
List and mark-read recheck authority again, so stale notification IDs cannot
reveal revoked scenes. Guests never have a private inbox.
