# Legacy dashboard port — C1

Decision authorized on 2026-10-16: copy the original references/devtool UI, including original colors. User explicitly removed the monochrome variant. Keep native scene styling.

Sources: references/devtool/src/app/globals.css, tailwind.config.ts, Sidebar, WorkspaceSwitcher, NavUser, DashboardHeader and SceneItem. Reference lives in the main checkout. Port presentation into React/Vite/Zustand; do not copy Next/Jotai data plumbing or dead controls.

Scope tokens to DashboardShell and portal surfaces through React context; do not set a global theme class on body. Preserve Arial/Helvetica, original light/dark shadcn tokens, sidebar blue focus, red destructive, 0.5rem radius, 16:9 cards and p-8 header. Use Radix focus management for mobile drawer and menus.

Import flow: validate type/version and element geometry, file map, JSON structure and 25 MiB limit before POST /scenes/ with title, folderId and content atomically. Only persisted app state is retained. Backend contracts/schema unchanged.

Verification: existing unit/browser suite; import validation tests; responsive and keyboard navigation; theme persistence and portal inheritance; no scene theme leakage; import failure issues no writes. Inspect screenshots against rendered reference component styles.

Human docs: docs/user/dashboard.md. Publish equivalent plain Markdown under frontend public/docs and index it in llms.txt after verification. No deployment change in C1.

## Implemented details and scope

- Original CSS tokens are copied verbatim from the reference, including destructive/chart/sidebar accent colors. Only selectors are scoped. Color is applied to portal roots as well as variables.
- Radix dialogs keep focus trapping/Escape handling. Desktop collapse persists in browser storage; mobile uses a drawer with a constrained grid track for long names.
- Workspace heading, current folder actions and explicit Open/action controls are retained to preserve working v1 navigation. Cards use original aspect ratio, rounded border, compact title and hover shadow with native SVG previews. Theme tokens allow readable cards in dark mode instead of retaining the old hard-coded white card surface.
- Original Header and SceneItem were rendered as an isolated fixture using their source, with only Next image/link and data hooks substituted. Compared against desktop/mobile screenshots. Missing creator/privacy fields are not invented.
- Import validates nested native fields and legacy font before POST. Raster file types only; SVG import is explicitly unsupported in C1. Preview errors fall back to a placeholder without blocking navigation.
- No new API endpoint, schema, server authorization or deployment config is introduced by C1.
- Browser tests use API fixtures and are UI proofs; real-stack permissions/persistence remain E1's integration gate.
- macOS Chromium visual baselines are checked in. A Linux visual baseline must be rendered and inspected in E1 before enabling visual checks on Linux CI.
