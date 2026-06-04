# EXC-025: Socket.io Backend + Frontend Client

## Status

planned

## Lane

high-risk

## Product Contract

Go Socket.io server, mount on chi router /socket.io/*, FE socket.io-client singleton, SocketContext provider

## Relevant Product Docs

- `04-01-PLAN.md`

## Acceptance Criteria

- Socket.io service handles WebSocket connections in Go.
- Socket route is mounted under chi router.
- React SocketContext provider provides client connections for dashboard.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-04/EXC-025-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-025-design.md)

- **Modified Files**: `internal/delivery/http/socket.go`, `router.go`, `dashboard/src/lib/socket.ts`, `dashboard/src/context/SocketContext.tsx`
- **Sprint Phase**: phase-04

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./cmd/server && cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-025 in the harness database.

## Evidence

(No evidence yet)
