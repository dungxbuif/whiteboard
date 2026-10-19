# VM100 Compose tmpfs compatibility — 2026-10-19

Status: verified in production.

During the first collaboration-v2 production create, Docker Compose v5.6.0
rejected `mode=1777` as an independent mount path. The YAML flow-sequence form
`tmpfs: [/tmp:size=64m,mode=1777]` treats the comma as a list delimiter,
producing two tmpfs entries even though `docker compose config --quiet` accepts
the file. Backend creation completed first; collab/MCP creation stopped before
either service ran.

The production manifest and Homelab VM100 template now use a block-sequence
scalar for each complete `/tmp:size=...,mode=1777` value. The minimal
reproduction retained the original create failure and proved the block scalar
creates `HostConfig.Tmpfs` with one `/tmp` key and the complete requested
options.

Production verification passed after the correction:

- `whiteboard-collab`: `/tmp` has `size=64m,mode=1777`.
- `whiteboard-mcp`: `/tmp` has `size=512m,mode=1777`.
- `whiteboard-frontend`: `/tmp` has `size=64m,mode=1777`.
- The complete backend, collaboration, MCP and frontend stack became healthy.
- `https://whiteboard.dungxbuif.com/healthz` returned successfully after the
  corrected create.

Application commit `d885975` records the production-manifest correction. The
change affects packaging only: API, schema, permissions and user-facing
behavior are unchanged, so no product/API contract change is required.

The remaining runtime proof is a controlled VM reboot through the installed
Homelab recovery unit. That proof is tracked separately in the release
checklist and is not implied by this create-time compatibility result.
