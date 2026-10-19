# Collaboration v2 release audit — 2026-10-19

Status: superseded by the active VM100 release. This file records the transition
from the earlier pause checkpoint to the deployed state; the release checklist
is the current source for remaining gates.

## Completed since the pause

- Reachable Git history is scrubbed of the retired Google OAuth client values;
  rewritten `master` was accepted by GitHub Push Protection without bypass.
- Whole-branch auth/deployment review is complete. All Go packages, the root
  collaboration149/MCP43/contracts13/diagram41 tests, frontend93 unit tests and
  production build pass.
- `npm audit --omit=dev` reports zero production findings. Nine development/build
  entries remain classified in the dependency review.
- A Homelab recovery unit is installed and enabled; it waits for Swarm/network
  metadata and reapplies the stack. Its non-reboot proof passes.
- The fresh cutover PostgreSQL dump was listed, restored in isolation and matched
  across all14 table row counts. Production advanced from migration6 to13 with
  `dirty=false`.
- Four independent collaboration service secrets of at least32 bytes are
  installed and distinct from the login JWT secret; values were not printed.
- Frozen Linux/amd64 backend, collaboration, MCP and frontend image IDs plus the
  v1 rollback images are recorded in the release note.
- Collaboration v2 is deployed from `772d3ea`; all four services are healthy and
  `COLLAB_V2_ENABLED=true`.
- Public HTTPS health, authenticated deep-link save/reload, dashboard/settings
  access UI, S3 bucket write/read/delete, native MCP authentication/reads and
  revoked-key rejection pass.
- Internal-route variants return 404; inspected local service/proxy logs contain
  no credential-bearing URL. Scoped production TURN config and its
  unauthenticated STUN challenge pass without exposing credential values.
- The Compose tmpfs YAML parsing incident is corrected and the actual container
  tmpfs maps are verified.
- Production commit `772d3ea` re-resolves Docker service names at runtime;
  public WebSocket admission/sync passed across a collaboration-container IP
  change without restarting the frontend.
- Authenticated production TURN Allocate returned a relayed address and Refresh
  lifetime zero released it.
- A controlled VM reboot restored external networks, all four healthy services,
  migration13 and public health automatically through the recovery unit.
- Homelab commit `9722490` reconciles the deployment template, semantic Compose
  regression, recovery tests, architecture, runbook and release evidence. The
  Homelab repository has no configured remote.

## Remaining release gates

1. **Final production canaries:** finish public view/edit mutation,
   invitation/role/revoke, native MCP write/export, two-peer browser voice/media
   and mobile/keyboard coverage.
2. **Live AI provider UAT (external):** requires the user-provided backend-only
   `AI_API_KEY`.
3. **Retired OAuth revocation (external):** revoke the retired credential in
   Google Cloud. Its values are absent from reachable Git history.

## Separate product backlog

Enterprise parity still includes comments/guest comments, screen sharing, live
presentation/spotlight/waiting-room/activity UI, version history/restore,
libraries/collections, domain capture, SSO/SCIM and retention. Harness CLI hard
gates, loop guard and security-standard enforcement also remain separate
milestone work. Neither backlog changes the accuracy of the deployed v2 status;
neither is claimed as part of this release.

Do not perform image-only rollback after a v2 scene is promoted. Use the
replacement-scene recovery procedure or a verified database restore.
