# Workspace library API

- `GET /api/v1/workspaces/{workspace}/libraries` lists revisions visible to a
  current member.
- `POST /api/v1/workspaces/{workspace}/libraries/import` imports native library
  v2 JSON. Owner only; maximum 500 items and 25 MiB.
- `PUT /api/v1/workspaces/{workspace}/libraries/{item}` publishes the next
  immutable revision. Owner only and revision preconditioned.
- `GET /api/v1/workspaces/{workspace}/libraries/{item}/revisions/{revision}`
  returns or exports one immutable revision.
- `GET /api/v1/workspaces/{workspace}/libraries/{item}/revisions/{revision}/export`
  downloads native JSON.
- `POST /api/v1/workspaces/{workspace}/scenes/{scene}/library-insertions` inserts
  through the durable collaboration writer using `requestId` and decimal
  `expectedSequence`.

Library reads never grant scene access. Insertion requires current target edit
access and remaps all element, group, file, frame, container and binding IDs.
Personal browser libraries are not uploaded automatically.
