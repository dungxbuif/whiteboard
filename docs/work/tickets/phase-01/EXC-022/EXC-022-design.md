---
artifact_type: detail_design
id: EXC-022-DESIGN
status: approved
ticket: EXC-022
approval: user-requested-project-completion
---

# EXC-022 Detail Design: Password-Protected Links

Password-protected links return `401` with a machine-readable
`passwordRequired` flag and accept the passphrase through a one-time query
request. Passwords are bcrypt hashes and are never returned to clients. The
public viewer prompts again after an incorrect passphrase.
