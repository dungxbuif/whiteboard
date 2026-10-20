# M3 live presentation controls — implementation record

Status: implementation complete; real-stack execution blocked by missing local test database role

## Intended behavior

- The existing authenticated collaboration socket carries every live-presentation
  control. No second socket or RelayHub dependency is introduced.
- The runtime owns one authoritative session revision. Host frame, spotlight,
  viewport lock, participant removal, and host transfer require the exact current
  revision. Transfer targets a currently authorized signed-in editor.
- Pending guests receive only their own waiting status. Admitted audiences receive
  a sanitized presentation state without the pending queue. View locking moves the
  audience camera only; it never grants or removes scene edit authority, and Leave
  plus Escape remain available.
- Reactions use a fixed allowlist, one reaction per connection per second, and a
  server-issued three-second expiry. A spotlight is cleared when its participant
  leaves. Host disconnect or current-authority loss ends the session.
- Frame navigation uses stable Excalidraw frame IDs. If the selected frame no
  longer exists, the audience sees an explicit fallback instead of silently moving
  to another frame. Invite links are copied manually and are never emailed.

## Contract and affected surfaces

- Node extends `PresentationSessions` and `Rooms` with revisioned frame, spotlight,
  lock, remove, and transfer controls plus ephemeral reaction and leave controls.
- `@collab/contracts` validates strict host and audience presentation-state payloads.
- The frontend `LivePresentation` controller consumes the existing transport and
  exposes host/audience actions. Host, waiting, and audience controls float inside
  the full-height canvas.
- `/present/{token}` is the public waiting/admitted entry route and uses the M2
  HTTP join contract plus `presentation_auth` on the same collaboration socket.

## Verification plan

- Node unit/runtime: stale revision, editor-only transfer with current authority,
  spotlight cleanup, reaction allowlist/rate/expiry, pending isolation, host exit.
- Frontend unit: strict control parsing, locked viewport with Escape/Leave,
  removed-frame fallback, waiting/admitted/denied/removed states, autoplay and
  media-permission failures without loss of drawing.
- Real stack: one host and two guests cover admit, deny/remove, transfer, refresh,
  revoke, mobile waiting UI, screen/voice denial, and continued drawing.
- Full collab/frontend suites and production build must pass. External/two-network
  media proof remains part of E1 and is not inferred from a local browser fixture.

## Reconciliation and evidence

- `npm test --workspace ./collab -- presentation`: 2 files, 14 tests passed.
- `npm run test:unit --workspace ./frontend -- live-presentation.test.ts
  presentation-controls.test.tsx transport.test.ts`: 3 files, 37 tests passed.
- `npm run test:unit --workspace ./frontend`: 46 files, 192 tests passed.
- `npm run build --workspace ./frontend`: TypeScript and production Vite build
  passed.
- `enterprise-live.spec.ts` covers one host, two public guests and one editor:
  manual copied invite, mobile waiting isolation, admit, reaction, remove, deny
  and current-editor host transfer. The local run stopped before application
  startup because the configured loopback Postgres URL has no `test` role
  (`role "test" does not exist`). No presentation assertion failed.

The repository-wide npm run also reached all non-database suites: contracts
passed 22/22 and diagram-kit passed 41/41. Database-backed collab/MCP suites
could not initialize for the same missing local Postgres `test` role; the
disposable recovery fixture additionally requires its guarded port 55439 URL.

The shipped UI mounts the waiting page without the canvas; after admission it
reuses `CollabProvider`, so existing drawing, voice, screen-share permission and
autoplay behavior remain in place. Presentation controls never request media.
Viewport lock invokes only `scrollToContent`; Escape and the floating Leave
button still send `presentation_leave`. Credential loss closes the canvas and
offers a fresh waiting-room join.

Independent M3 review reported host-end resurrection, stale audience reconnect,
transfer authority TOCTOU, ordinary-member audience classification and
runtime/schema bound mismatch. Reconciliation now sends a terminal host event,
ends an offline audience into explicit fresh rejoin, rechecks transfer authority
after the feature await, models non-admitted members as observers, and bounds
admitted/reaction state consistently. Focused regression tests cover each fix.
The final lifecycle pass also terminates the caller host when authority disappears
during a control, ends stale observers when their socket goes offline, and sends
the current sanitized presentation state to collaborators that reconnect during
an active session. Regression tests cover both paths; no P1 or P2 review finding
remains open.
Host startup now also reaches a retryable error when the socket closes, the
runtime rejects the start, or no authoritative state arrives within ten seconds;
the acknowledgement timer is cleared on every terminal or successful path.

