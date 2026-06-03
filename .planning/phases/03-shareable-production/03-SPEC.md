# Phase 3 Spec: Shareable Production

## 1. Goal
Enable public sharing of drawings with fine-grained access control, replicating the original Excalidraw production sharing logic.

## 2. Requirements
- **REQ-01: Public Links:** Generate obfuscated public URLs for read-only access.
- **REQ-02: Link Management:** Users can enable/disable public sharing for any scene.
- **REQ-03: Access Control:** Support for basic password protection on shared links.
- **REQ-04: Parity:** Ensure the sharing logic matches the reference server's public module.

## 3. Acceptance Criteria
- [ ] Public links are accessible without login.
- [ ] Revoking a link immediately blocks access.
