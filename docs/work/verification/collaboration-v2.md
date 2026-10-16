# Collaboration v2 verification

## C1 — original dashboard presentation

Date: 2026-10-16. Branch: codex/collaboration-v2. Production remains v1.

- Baseline backend: `rtk go test ./...` — 59 tests passed before v2 code changes.
- Frontend unit: `rtk npm run test:unit` — 33 passed, including malformed nested bindings/font, duplicate IDs, size limits, session-state filtering and external image rejection.
- Browser: `rtk npm exec -- playwright test dashboard-style.spec.ts dashboard.spec.ts --workers=2 --reporter=line` — 14 passed before visual baselines; final baseline comparison recorded below.
- Production frontend build: `rtk npm run build` — successful; dependency chunk-size warning remains.
- Red proof: original app lacked sidebar navigation landmark, mobile trigger and import input. New tests failed on those missing behaviors.
- Visual review found and fixed grid min-content overflow and drawer position specificity. Added bounds assertions. Dark dialog foreground and malformed nested import fields were independently reproduced by review and covered by regression tests.
- Original Header/SceneItem source rendered independently with original CSS, replacing only framework/data plumbing. [Reference rendering](collaboration-v2/legacy-reference-components.png). New screenshots inspected at 1440×960 and 390×844; original colors, type, spacing and card geometry preserved. Existing workspace heading/actions retained. No monochrome override.
- Four macOS Chromium golden screenshots under frontend e2e verify light/dark desktop and mobile drawer/empty view. Linux baselines are pending E1; no Linux visual result claimed.
- Review: two Important findings (nested import validation, dark dialog text) fixed. Follow-up legacy font gap fixed and unit-tested. Minor background-preview issue fixed.
- Docs review: human dashboard guide and identical public Markdown; llms.txt entry; architecture/ADR/spec/plans reconciled. API/schema/operations unchanged in C1.
- UI fixtures do not demonstrate live ACL, DB or collab correctness. Those remain A/B/E work.

Final C1 verification: 2026-10-16, frontend build exit 0; 33 unit tests passed; 14 Chromium tests passed including four inspected golden image comparisons (7.1s). No production deployment performed.
