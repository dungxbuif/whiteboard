---
artifact_type: integrations_doc
id: INTEGRATIONS
status: draft
owner: shared
human_fields: [provider_choice, credential_approval, business_owner]
ai_fields: [systems, contracts, secrets_names, failure_modes]
shared_fields: [status, trace]
---

# Integrations

## Field Ownership

- Human owns provider choices, credential approval, and business owner.
- AI documents systems, contracts, secret names, and failure modes.

## External Systems

| System | Purpose | Contract | Owner | Status |
| --- | --- | --- | --- | --- |
| TBD | TBD | TBD | TBD | draft |

## Credentials And Secrets

Document required secret names only. Do not store secret values.

- TBD

## Failure Modes

- TBD

## Collaboration v2 presence and voice

The custom Node WebSocket shares the admitted scene scope for Yjs, roster, cursor/selection, follow and WebRTC signaling. Go room credentials and current ACL remain authoritative; a server connection UUID identifies each tab independently of its principal. Voice has ten active connections per scene; rejected voice admission leaves document collaboration available. Camera/follow/voice state is ephemeral. Browser media uses API-issued short-lived Coturn credentials, never credentials embedded in the frontend. See the [B4 contract and verification record](../work/verification/collaboration-presence.md).

A 25-second server protocol heartbeat maintains the bounded nginx tunnel and removes nonresponsive connections by the following interval. Disconnect/revoke cleanup releases voice admission and clears followers. Browser integration is verified with native same-user tabs, follow/camera, permission denial, generated audio, mute and leave cleanup. The legacy Socket.IO client/server/proxy path is retired; `/collab/` is the sole collaboration socket. VM100 deployment verification remains pending.

Ingress limits separate shared-IP transport resource protection from operations: 2,000 frames/second/IP, 64 MiB/minute byte refill with 8 MiB burst; authentication60/minute/IP; cursor/viewport share30/second/connection; other ephemeral controls120/minute/connection. Document writes retain120/minute/principal and1 MiB frames. Raw received bytes are charged before decoding; limiter entries expire and active entry counts are bounded. These limits accommodate normal cursor traffic from50 clients sharing one IP without consuming their authentication or durable-write budgets.
