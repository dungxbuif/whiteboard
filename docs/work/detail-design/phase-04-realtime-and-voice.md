# Phase 4 Detail Design: Realtime Collaboration and Voice

## Intended behavior

- An authenticated user can join a scene room and receive presence, cursor, and scene update events for up to ten peers.
- The server validates the bearer token during the Socket.IO handshake and isolates rooms by workspace and scene.
- Scene updates are broadcast optimistically and persisted through the existing scene PATCH flow; the last writer remains the source of truth.
- Voice controls use browser media permissions and WebRTC signaling through the realtime channel. TURN configuration is supplied through environment variables.

## Technical decisions

- Keep the HTTP API as the persistence boundary. Realtime messages are ephemeral coordination events.
- Use a Socket.IO-compatible WebSocket endpoint at `/socket.io/` and a small room manager in the Go delivery layer.
- Use a provider in the React shell so the whiteboard component can publish cursor and scene events without owning connection lifecycle.
- Add a coturn service definition and explicit `TURN_*` configuration, while keeping local development usable without TURN.

## Affected flows and contracts

- New events: `scene:join`, `scene:leave`, `cursor:move`, `presence:update`, `scene:update`, `voice:offer`, `voice:answer`, `voice:ice`.
- New REST endpoint: `GET /api/v1/realtime/config` returns non-secret ICE server configuration for authenticated clients.
- Scene updates carry `{sceneId, content, contentVersion, clientId}` and are rejected when the room does not match the authenticated workspace.

## Verification plan

- Go tests cover room membership, workspace isolation, cursor fan-out, and malformed event rejection.
- Frontend unit tests cover connection state, peer presence, and voice mute state.
- Playwright smoke coverage joins a scene with a mocked realtime server and verifies the collaboration toolbar.
- `docker compose config` validates the coturn service and environment interpolation.
