---
artifact_type: detail_design
id: EXC-020-DESIGN
status: approved
ticket: EXC-020
approval: user-requested-project-completion
---

# EXC-020 Detail Design: Sharing Backend

The authenticated share endpoint upserts an active permission and returns the
opaque token. Revoke sets `active=false`. The public usecase checks active
state, verifies an optional bcrypt password, and returns only read-only scene
data. All owner operations remain workspace-scoped through the existing
middleware.
