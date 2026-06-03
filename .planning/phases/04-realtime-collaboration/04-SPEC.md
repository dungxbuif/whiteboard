# Phase 4 Spec: Real-time Collaboration & Calling

## 1. Goal
Implement seamless multi-user collaboration and voice calling with a focus on self-hosted infrastructure and minimal port exposure.

## 2. Requirements
- **REQ-01: Real-time Sync:** Multi-user editing with cursor visibility and presence.
- **REQ-02: Voice Calling:** Integrated P2P voice calls using WebRTC.
- **REQ-03: Relay Infrastructure:** Deployment of a unified TURN/STUN relay (e.g., Coturn) on a single port.
- **REQ-04: Performance:** Low-latency synchronization for up to 10 concurrent users per scene.

## 3. Acceptance Criteria
- [ ] Multiple users can edit the same scene simultaneously.
- [ ] Voice calls are clear and stable.
- [ ] Infrastructure runs behind a single endpoint.
