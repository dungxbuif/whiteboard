---
artifact_type: detail_design
id: EXC-019-DESIGN
status: approved
ticket: EXC-019
approval: user-requested-project-completion
---

# EXC-019 Detail Design: Share Permission Persistence

Store one opaque share token per scene in `scene_permissions`. The row is
workspace-scoped, has an active flag, and optionally stores a bcrypt password
hash. Public readers resolve the token without JWT access; owners manage the
row through the authenticated scene route. Token generation uses UUIDs and the
database enforces uniqueness.
