---
artifact_type: detail_design
id: EXC-021-DESIGN
status: approved
ticket: EXC-021
approval: user-requested-project-completion
---

# EXC-021 Detail Design: Public Share Surface

The backend exposes `GET /api/v1/public/scenes/{token}` without JWT. The SPA
detects `/p/{token}` before the authenticated dashboard and renders a read-only
Excalidraw canvas. Owners receive a share dialog with a copyable URL.
