---
artifact_type: user_stories
id: USER_STORIES
status: active
owner: shared
updated: 2026-10-19
human_fields: [role, need, benefit, acceptance_criteria]
ai_fields: [story_rows, trace_links, status_updates]
shared_fields: [stories]
---

# User Stories

## Field Ownership

- Human owns user intent and acceptance criteria. The rows below derive from the
  approved phase and Collaboration v2 specifications.
- AI maintains trace links and evidence-backed status updates.

`Verified` means implementation and local proof exist. Production cutover is a
separate operator story and remains `in_progress`.

## Delivered And Release Stories

| ID | Role | Need | Benefit | Acceptance Criteria | Requirement | Status |
| --- | --- | --- | --- | --- | --- | --- |
| US-01 | Account user | Sign in with email and password | I can enter my private workspace without a third-party identity provider | Valid credentials return a JWT and invalid credentials reveal no account detail | REQ-04 | verified |
| US-02 | Operator | Create or update a password user from the CLI | I can bootstrap access safely | `userctl` hashes the password and transactionally ensures exactly one default `My Workspace` | REQ-04 | verified |
| US-03 | Workspace user | Switch workspaces and organize scenes in nested folders | I can manage drawings like files | Workspace isolation, folder navigation, deep links, search, Starred, Trash, and restore work through stable routes | REQ-05–REQ-09 | verified |
| US-04 | Diagram author | Open the native Excalidraw canvas and return through a link | My drawing remains editable and addressable | `/w/{workspace}/scenes/{scene}` reloads with authorized persisted content and assets | REQ-03, REQ-10 | verified |
| US-05 | Presenter | Treat frames as ordered slides and navigate them | I can present a scene without editing distractions | Frame order persists; controls and keyboard move between frames | P2-REQ-01–03 | verified |
| US-06 | Workspace owner | Create independent public view and edit links | I can share one scene with the intended capability | Each link is scoped, optional-password, returned once, independently revocable, and never grants workspace membership | P3-REQ-01, P3-REQ-03 | verified |
| US-07 | Guest | Open a shared scene without creating an account | I can view or edit only what the link permits | `/p/{token}` enforces password/access/expiry; edit sessions require a display name and keep identity scoped to the link | P3-REQ-02–04 | verified |
| US-08 | Workspace owner | Invite an email-bound editor or viewer by copying a link | I can add members without an email-delivery dependency | The link is single-use, expires after seven days, binds the normalized email, and never downgrades an existing member | V2-REQ-01–02 | verified |
| US-09 | Invited new user | Register a password account through my invitation | I can join the target workspace and still receive a personal workspace | Account, default workspace, invited membership, and invitation consumption commit atomically | V2-REQ-02 | verified |
| US-10 | Editor | Collaborate concurrently and continue after network/browser/runtime interruption | My acknowledged changes are not lost | Custom `/collab/` sync converges, ACKs only durable updates, queues offline work, and replays after resync | V2-REQ-03–04 | verified |
| US-11 | Collaborator | See cursors/selections, follow another participant, and join voice | Coordination stays in the drawing experience | Native presence/follow work; voice signaling uses the same socket; at most ten active voice participants; drawing continues after media denial | V2-REQ-05 | verified |
| US-12 | Public editor | Paste an image and reopen the scene | Shared editing includes durable media | Upload capability is scene-scoped and edit-only; signed storage PUT/GET, reload, renewal, and failure retention preserve asset identity | REQ-03, P3-REQ-04 | verified |
| US-13 | Workspace owner | Issue a narrowly scoped, revocable agent key | Automation cannot silently exceed my grant | Raw key is shown once, stored hashed, checked against current authority, and independently scoped by capability and scene | V2-REQ-06 | verified |
| US-14 | Agent | Discover workspaces/scenes and draw or export native diagrams through MCP | I can collaborate through the same rules as people | HTTP/stdio discovery, current reads, native presets/Mermaid, durable patch/delete, activity, and export enforce scopes and preserve human edits | V2-REQ-06–07 | verified |
| US-15 | AI-assisted author | Ask an OpenAI-compatible model for chat, diagrams, image reasoning, or code | I can accelerate diagram work without exposing the provider key | Backend proxy validates bounded input/output; canvas UI renders results; key remains backend-only | P5-REQ-01–03 | verified |
| US-16 | Release operator | Promote Collaboration v2 on VM100 with a recoverable, auditable cutover | Production behavior matches local acceptance | Reboot recovery, backup/restore, secrets, frozen images, migrations, public canaries, dependency/security review, and docs evidence all pass | V2-REQ-08 | in_progress |
| US-17 | AI-assisted author | Use the configured AI provider from the public production origin | The released AI workflow is proven end to end | Chat, text/image-to-diagram, and to-code pass with the user-provided backend-only provider key | P5-REQ-04 | blocked |

## Approved Backlog Stories

| ID | Role | Need | Benefit | Acceptance Criteria | Requirement | Status |
| --- | --- | --- | --- | --- | --- | --- |
| US-E01 | Collaborator or guest | Add moderated comments anchored to scene elements | Discussion remains attached to the work | Durable threads, guest policy, moderation, notifications, ACL, and audit have dedicated contracts and tests | ENT-P1-01 | ready |
| US-E02 | Voice participant | Share a screen with explicit consent | The room can review external material | Separate media-track permissions and participant limits are enforced without weakening scene/voice ACL | ENT-P1-02 | ready |
| US-E03 | Presenter or audience member | Run a controlled live presentation | Large sessions can follow safely | Leadership, spotlight, waiting room, reactions, and view locking have explicit durable/ephemeral state and authorization | ENT-P1-03 | ready |
| US-E04 | Workspace member | Browse meaningful activity | I can understand recent changes and access events | The UI consumes redacted audit data under a documented retention policy | ENT-P1-04 | ready |
| US-E05 | Scene editor | Inspect and restore named historical versions | I can recover an earlier user-visible state | History, labels, diff, restore, ACL, promoted-scene recovery, and audit are specified and tested | ENT-P2-01 | ready |
| US-E06 | Workspace member | Reuse shared libraries and collections | Repeated diagram work is easier to organize | Sharing, ownership, search, versioning, and import contracts are defined | ENT-P2-02 | ready |
| US-E07 | Enterprise administrator | Control domains, SSO/SCIM, and retention | Organization identity and lifecycle policy are enforceable | Domain proof, provisioning/deprovisioning, retention, audit, and failure recovery are specified after password ACL stability | ENT-P2-03 | ready |
| US-H01 | Repository maintainer | Enforce Harness hard gates, loop guard, and security standards | Automated work follows the documented lifecycle | The ready Harness milestone is implemented with tests or explicitly removed by the owner | HARNESS-REQ-01 | ready |

## Traceability

- Requirement definitions: [REQUIREMENTS.md](REQUIREMENTS.md).
- Approved product scope: [SPEC.md](SPEC.md) and
  [Collaboration v2 design](../superpowers/specs/2026-10-15-collaboration-v2-design.md).
- Proof and release boundaries: [VALIDATION_MATRIX.md](../work/VALIDATION_MATRIX.md),
  [real-stack acceptance](../work/verification/collaboration-v2-real-stack.md), and
  [release audit](../work/verification/2026-10-19-release-pause-audit.md).
