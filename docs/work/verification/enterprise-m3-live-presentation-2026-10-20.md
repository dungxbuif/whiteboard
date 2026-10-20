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

- `npm test --workspace ./collab -- presentation`: 2 files, 17 tests passed
  twice consecutively.
- `npm run test:unit --workspace ./frontend -- live-presentation.test.ts
  presentation-controls.test.tsx transport.test.ts`: 3 files, 40 tests passed
  twice consecutively.
- `npm run test:unit --workspace ./frontend`: 48 files, 198 tests passed.
- `npm run build --workspace ./frontend`: TypeScript and production Vite build
  passed.
- `go test ./internal/usecase ./internal/repository/collab`: passed, including
  current-host link revision and transfer-race regressions.
- `enterprise-live.spec.ts` now contains two browser flows covering one host,
  public guests and one editor:
  manual copied invite, mobile waiting isolation, admit, reaction, remove, deny
  and current-editor host transfer, plus waiting Leave, keyboard Escape,
  permission denial and ended pending-auth recovery. Both local cases stopped
  before application
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
an active session. Regression tests cover both paths.
Host startup now also reaches a retryable error when the socket closes, the
runtime rejects the start, or no authoritative state arrives within ten seconds;
the acknowledgement timer is cleared on every terminal or successful path.

## Fresh root-gate reconciliation

The release gate additionally requires session attachment checks after every
host-start authority await, monotonic client revisions, terminal-state immunity
to late presentation state, and a server revision recheck immediately before
each presentation broadcast. Waiting-room Leave must remove the guest from the
host queue. A rejected or ended `presentation_auth` attempt must show a terminal
screen with a fresh-rejoin action instead of remaining in Joining.

Presentation link creation and revocation must bind twice to the same current
runtime host revision: once before entering the idempotent mutation and once
inside it immediately before persistence. A transfer or host change during the
operation rejects the mutation. Rejection of a proposed transfer target maps to
`PRESENTATION_TARGET` and leaves the current host attached and authoritative.

Verification adds disconnect-during-gate, stale publish, queue removal,
presentation-auth rejection, link transfer-race/non-host, and transfer-target
rejection regressions. The real-stack spec records the equivalent browser flows;
execution still requires the guarded disposable database environment.

The fresh focused and build gates pass. A repository-wide Go run reached all
packages but currently fails two concurrent I3 SCIM handler tests; the M3
usecase and runtime-client packages pass independently. These SCIM failures do
not touch the M3 paths and remain outside this commit.

## Final authority and start-timeout closure gate

Pending-queue delivery must use the same fresh recipient path as every other
presentation broadcast. Immediately before a host receives a queue containing
guest identity, the runtime rechecks the host's room access, the
`presentation:host` feature, its exact connection attachment, and the current
session revision. If any check fails, the runtime ends that presentation and
does not disclose the pending identity.

A host start that receives no authoritative state within ten seconds closes the
exact collaboration socket used for the start. Server detach therefore ends any
session that accepted the request but whose acknowledgement was delayed, while
the transport reconnects for a clean retry. Late state for the timed-out session
cannot revive the terminal client state.

Verification adds a gated feature-loss race during pending admission fanout and
a delayed-start test that advances beyond ten seconds, asserts the exact socket
abort, rejects late resurrection, and proves a retry uses the clean connection.

