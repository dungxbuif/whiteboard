# E2 isolated packaging proof

## Enterprise feature worker

Build the backend image and run the worker lifecycle proof:

```sh
rtk docker build -f src/backend/Dockerfile -t whiteboard-enterprise-backend:local src/backend
rtk node tests/packaging/feature-worker.mjs
```

The fixture creates a unique network and disposable PostgreSQL database, applies
all migrations, starts the real non-root/read-only worker with only its database
credential, assigns it a live lease, kills it, verifies a replacement cannot
steal the lease early, and then verifies recovery after expiry. Cleanup removes
only resources prefixed with the fixture process ID. The fixture uses no
production database or secrets.

Build from repository root using the root workspace lock:

```sh
rtk docker build -f src/collab/Dockerfile -t whiteboard-e2-collab:local .
rtk docker build -f src/mcp/Dockerfile -t whiteboard-e2-mcp:local .
rtk docker build -f src/frontend/Dockerfile -t whiteboard-e2-frontend:local .
rtk node tests/packaging/smoke.mjs
```

The smoke requires installed root workspace dependencies on the invoking host.
It uses unique names, a disposable PostgreSQL container, an isolated bridge,
audit volume, and a random loopback-only frontend port, then cleans them up in
finally. Never pass a production database URL. Actual compiled collab/MCP
services and the nginx image are tested; Go authorization is a signed fixture
and MCP upstream is a loopback HTTP fixture. E3 owns full real Go/private-network
and release-source proof. The smoke exercises native PNG generation inside the
production renderer resource/security settings, authenticated export/download,
MCP SDK+SSE, WebSocket ready, origins, route denial, users, and audit permissions.

## Cross-network TURN diagnostic

`turn-proof.mjs` uses local Playwright and an operator-provided SSH tunnel at
127.0.0.1:19223 to a disposable gateway Chromium CDP listener127.0.0.1:19222.
Use the pinned Playwright1.60.0 image with init, memory1GiB and pids256. Keep CDP
loopback-only, verify with ss, and remove only that QA container+tunnel afterward.
The script mints a ten-minute credential over saved VM100 SSH using the deployed
backend secret; no shared secret or short-lived credential is logged. Two
browsers send440Hz generated audio with relay-only ICE. It reports selected pair
candidate types/ports and received bytes/packets/energy; no physical microphone.

```sh
rtk node tests/packaging/turn-proof.mjs
# Diagnostic only when hostname fails: result explicitly records overridden URL.
rtk env TURN_PROOF_URL=turn:103.82.21.202:5349 node tests/packaging/turn-proof.mjs
```

The recorded2026-10-16 local network is Mac10.10.0.10/WAN117.1.89.250; gateway
WAN103.82.21.202. Revalidate those labels for subsequent runs. A direct-IP success
is not proof that the configured hostname resolves or connects correctly.

## Enterprise screen-media proof

`enterprise-media.mjs` first sends a generated 15 fps canvas stream between two
local pages, then repeats it between the local browser and the isolated gateway
browser with relay-only ICE. It asserts decoded video frames, received bytes and
relay candidates. The generated stream contains no microphone or system audio.
Use the same loopback-only gateway CDP tunnel documented for `turn-proof.mjs`:

```sh
rtk env MEDIA_LOCAL_ONLY=1 node tests/packaging/enterprise-media.mjs
rtk node tests/packaging/enterprise-media.mjs
```

## B4 idle collaboration regression

Build collab first, then run `rtk node tests/packaging/idle-collab.mjs`. This uses the existing nginx image unchanged, the current compiled host collab runtime and a random schema in the guarded loopback `devtool_a1_test` PostgreSQL database. A unique disposable bridge and Node TCP-forwarder make the random loopback runtime port reachable as nginx's `collab:8082` upstream on Docker Desktop; no product proxy timeout is changed. The forwarder does not parse or generate WebSocket traffic.

The live peer stays idle for 160 seconds (production 25-second heartbeat), while a second peer disables automatic pong after joining voice. Assertions cover six or more live pings, no live close, dead-peer removal before 60 seconds, voice-slot cleanup and readiness. Go's upstream HTTP is a signed fixture; runtime, storage, room admission, WebSocket heartbeat and nginx are real. Cleanup removes only the unique QA containers/network and random schema. This is local packaging proof, not final release-image or production proof.
