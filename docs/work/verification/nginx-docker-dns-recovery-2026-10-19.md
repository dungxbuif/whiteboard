# Frontend Docker DNS recovery verification — 2026-10-19

Status: implementation and isolated packaging proof complete; production
deployment/restart replay pending.

## Trigger and intended behavior

The production Collaboration v2 canary restarted only `whiteboard-collab` and
then observed `HTTP 502` from the public `/collab` WebSocket upgrade while the
replacement runtime itself remained healthy and reachable by the current
Docker service name. The frontend nginx process had resolved `collab` only at
startup and retained an obsolete container address.

The frontend must resolve the `backend`, `collab`, and `mcp` service names
through Docker's embedded DNS while it is running. Recreating any upstream
container must not require a frontend restart and must not strand an open scene
in `Connecting · Read-only until admitted`.

## Verification plan

1. Extend the isolated production-image packaging smoke so it admits a real
   authenticated WebSocket through nginx.
2. Remove the collab container, occupy its old bridge address, and start a new
   healthy collab container under the same service alias.
3. Without restarting nginx, require a second public WebSocket upgrade and
   authenticated `ready` admission through `/collab`.
4. Build the frontend image, run the complete packaging smoke, deploy only the
   new frontend image, and repeat the production restart/public-origin canary.

This changes service discovery and restart behavior only. It does not change
the public API, schema, authorization rules, user flows, or agent contracts, so
no public documentation surface requires a new contract entry.

## Local evidence

The regression first failed with `Unexpected server response: 502` after the
test moved the collab alias to a different bridge address while leaving nginx
running. The frontend now uses Docker's embedded resolver with a five-second
cache and variable upstreams for backend, collab and MCP. The rebuilt image
passes `nginx -t`; the full isolated packaging smoke then passes the original
admission plus a second authenticated `ready` admission after the address
replacement. The smoke also verifies that variable proxying preserves the
backend API URI and the existing MCP routes.
