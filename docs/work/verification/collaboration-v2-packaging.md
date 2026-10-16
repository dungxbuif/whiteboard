# E2 packaging validation — 2026-10-16

Status: in_review; authorized by the [programme](../../superpowers/plans/2026-10-16-collaboration-v2.md), [E2](../../superpowers/plans/2026-10-16-collaboration-release.md). Production remains v1; deployment is E3.

## Intended behavior and decisions

Build frontend/collab/MCP from the root context using src/package-lock.json and npm ci. Compile contracts before dependent workspaces; include MCP native render-dist, bundled fonts/licenses, and pinned Playwright browser. All long-running images run non-root. MCP gets init, memory/PID limits, and a persistent mode-0700 audit directory (journal mode0600, active+two archives).

Keep frontend port80/whiteboard-frontend alias and existing external homelab-net/edge-http-net Swarm overlays. A dedicated internal bridge (default172.29.240.0/28, conflict check before deployment) connects only backend172.29.240.2 and collab172.29.240.3; authority8081 binds only .2, commands8083 only .3. Public8080/8082 binds container interfaces. No host ports. MCP Go client uses the public HTTPS origin; internal Docker HTTP is intentionally rejected by that client. Four dedicated collaboration secrets remain distinct; Go login JWT is separate. COLLAB_V2_ENABLED remains explicitly gated false until the controller's canary.

Nginx routes /api/, exact /collab, /mcp and /mcp/ descendants, denies /internal and /api/internal, supports Upgrade and unbuffered MCP streaming, and disables access/error request logging to prevent path-token disclosure. Legacy Socket.IO route retained until B3/B4 replacement proof.

Rejected: wildcard Go authority bind (violates validation), shared-overlay authority (widens exposure), Docker-hostname HTTP for MCP (violates HTTPS policy), replacing overlay with bridge (breaks Pi5 routing).

## Verification plan

Quiet compose validation with disposable mode0600 local env; isolated local image builds; real collab/MCP clients through nginx with disposable DB/authority fixture; public private-route negative cases; image user/artifact/audit checks. Read-only SSH inventory VM100/gateway/Pi5. Generated-tone browsers on local host and VM100 with relay-only ICE, selected relay candidates plus received RTP/audio-energy evidence and cleanup. Never log credentials or deploy v2. Existing v1 image/compose rollback target remains unchanged.

## Documentation reconciliation

Owned: this note, Homelab architecture/whiteboard/coturn runbooks and compose template. Controller owns central architecture/API/env/public guides/release/context/validation documents and commits. No schema or product behavior change in E2. Public deployed feature docs remain v1 until E3 succeeds; this plain Markdown operator evidence is agent-readable.

## Results

- PASS: quiet Compose validation with disposable mode0600 `.env.prod`, removed after use. Parsed model assertions confirm zero published host ports and only backend/collab attached to the internal authority bridge.
- PASS: Linux ARM64 Docker builds for frontend/collab/MCP from the root lock; Node24.19.0, nginx-unprivileged1.27, Playwright1.60.0. Root frontend lock entry matches frontend manifest; obsolete standalone lock removed only after clean container build passed. Unscoped npm prune removes root development tools (typescript/vite/vitest verified absent in collab runtime).
- PASS: `rtk node tests/packaging/smoke.mjs` against disposable PostgreSQL/isolated network. Actual compiled collab WebSocket Upgrade+authenticated ready; MCP SDK initialize/discovery/list-workspaces, SSE headers/unbuffered delivery, bad Origin403, authenticated `/mcp/exports` and artifact download; private route404; all app image users nonroot; audit directory0700/file0600. Native compile+PNG rendering succeeded inside memory2GiB/pids256/read-only/cap-drop container. Fixtures replace Go HTTP authority/API; this is packaging proof, not full Go production topology UAT.
- PASS: VM100/Pi5/gateway read-only inventory; no existing subnet overlap with proposed authority bridge. Pi5 actual container is `edge-traefik-pi5`, upstream `whiteboard-frontend:80`. Current compose is `/opt/apps/whiteboard/docker-compose.prod.yml`, `.env.prod` symlinks mode0600 `whiteboard.env`. v1 backend/frontend remain healthy.
- PASS (direct IP diagnostic): two actual browser networks, Mac WAN117.1.89.250 and gateway103.82.21.202, generated440Hz audio using `turn:103.82.21.202:5349`; relay↔relay UDP ports49164/49244, received20142/20174bytes and250/251packets, audio energy4.898/4.910. Shared secret equality checked without printing values. QA container/tunnel/peers cleaned up; Coturn/NetBird untouched.
- FAIL (configured hostname): local DNS maps `turn.relayhub.dungxbuif.com` to Pi5 `10.10.0.5`; TCP5349 refused and browser failed connection. E3 option: use tested plain-TURN IP URL, retaining realm/secret, or correct the specific split-DNS record and repeat the hostname proof. Neither fix applied here.

## Remaining release gates

These images are packaging validation artifacts from a dirty working tree, not a frozen release. Initial base432462f; latest capture b8aab4c plus B3/E2 work. Source snapshot hash and per-file manifest are in `.superpowers/sdd/2026-10-16-collaboration-v2/task-E2-source-manifest.json`; source may change after the capture. E3 must build linux/amd64 at whole-branch freeze after B4/reviews, verify real Go/private binds and authenticated public API/MCP connectivity, fix/test TURN endpoint, backup/restore and promotion rollback, inspect upstream proxy logging, then deploy/canary. No v2 rollout performed.

Public feature pages remain v1 because deployment has not happened. Internal operator docs and Homelab compose template were reconciled; human-readable Markdown is also directly agent-readable. Controller owns central env/API/release/context/validation updates and commits. No schema change in E2.
