# Enterprise parity planning verification — 2026-10-20

Status: planning review; runtime implementation not started.

## Intended artifact and scope

Produce a proposed enterprise detail design and implementation programme for
all ENT-P1-01–04 / ENT-P2-01–03 plus the user's canvas-control placement request.
Preserve the old dashboard and reuse actual legacy code patterns. Source
baseline is `1d2af8a`; inspect migrations/working tree again at execution time.

## Code and reference evidence

- Existing ACL/access service, scoped authority/capabilities and active-user JWT
  middleware: correct integration boundary; no SSO session assurance today.
- `collab/persistence.ts` captures materialized scene state, appends validated
  updates/receipts/audit with ownership fencing, projects per-scene activity.
  `commands.ts` supports add/patch/delete only; restore requires a new scoped
  internal operation rather than overloading an agent command.
- Native scene schema validates bindings, files and frame references; renderer
  API types expose `renderTopRightUI`, `MainMenu` and `Sidebar`.
- Current wrapper reserves a header row; UI1 explicitly removes that row.
  Local presentation store has no server session/host state; voice has ten
  participants and assumes microphone admission.
- Legacy CollectionList/CollectionItem show Private/workspace sidebar behavior,
  but menu handlers and backend CRUD are incomplete; copy presentation while
  implementing current ACL and handlers. Old Socket.IO is a behavior reference.
- Official go-oidc, RFC7644, Excalidraw API and display-capture references are
  linked in the design; proposed dependency selection remains an execution gate.

## Coverage / self-review

| Requirement | Coverage |
| --- | --- |
| Canvas controls | UI1 full-height native slots; UI2 overlays/focus |
| ENT-P1-01 | F1/F2/C1/C2 |
| ENT-P1-02 | M1 |
| ENT-P1-03 | M2/M3 |
| ENT-P1-04 | F2/C3/R1 |
| ENT-P2-01 | V1/V2/R1 |
| ENT-P2-02 | L1/L2/R2 |
| ENT-P2-03 | I1/I2/I3/R1/R2 |
| Cross-feature rollout | E1/E2 |

Self-review corrections: asynchronous media admission needs a second explicit
Start sharing gesture for getDisplayMedia activation; voice/screen share a union
slot count but separate peers; pending presentation tokens cannot enter the room
writer/sync or asset API; offline edits remain later valid edits after restore;
SCIM deprovision preserves independent manual memberships and reports them;
retention never prunes canonical receipt/history. No feature direction removed.

Planning validation checks: unique 22 task IDs, six plan headers, linked phase,
complete requirement coverage, balanced Markdown fences, existing local Markdown
links and `git diff --check`.

Executed from the application root:
- PASS: `rtk node --input-type=module` with an inline read-only fs/path validator
  enumerating `docs/superpowers/plans/2026-10-20-enterprise-*.md`, the spec, phase
  and this evidence. Result: six plans, nine documents, 22 unique task IDs,
  40 resolved local Markdown links, all seven requirement rows mapped, balanced
  fences, required headers present, no unresolved planning placeholders.
- PASS: `rtk git diff --check` (no whitespace errors).
- Inspection: `rtk cat src/frontend/playwright.collab.config.ts` confirms
  `e2e/real-stack/**/*.spec.ts` discovery and the plan's browser command target.

Runtime tests are not run for a docs-only planning change; no enterprise feature
may be marked implemented based on this document. Public runtime docs are
unchanged for the same reason.

## Documentation review

Internal updates: context/backlog/roadmap/requirements next-artifact links,
enterprise phase, proposed design/plans and this evidence. Public runtime guides
and `/llms.txt` do not change: these features are proposals and have not shipped.
API/ERD/protocol/ADRs and Homelab changes are identified in the owning execution
tasks and reconciled only against implemented behavior.

## Plan review handoff

Review the defaults in the detail design and the package order in the programme.
User already authorized agents; subagent-driven bounded tasks are recommended
after plan approval, beginning UI1/UI2 then foundation. Implementation approval
is not silently inferred from a request to prepare plans.
