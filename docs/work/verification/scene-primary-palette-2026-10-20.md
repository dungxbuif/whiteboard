# Scene primary palette and dashboard parity — 2026-10-20

## Intended behavior

- Scene-level shared controls use Excalidraw's indigo primary accent (`#6965DB`).
- The dashboard keeps the legacy app's existing monochrome `.dashboard-theme`
  tokens and visual style.
- Excalidraw's native canvas toolbar and element colors remain owned by the
  embedded editor.

## Technical decision

Change only the global scene token in `src/frontend/src/index.css`. The scoped
`.dashboard-theme` override in `src/frontend/src/styles/dashboard.css` remains
unchanged, so dashboard buttons and navigation preserve the legacy palette.

## Verification plan

- Run the frontend unit suite and production build.
- Verify the scene primary button/link accent in the production browser.
- Verify the dashboard still renders its monochrome legacy primary token.
- Reconcile public documentation only if the shipped user-facing theme
  behavior needs a note.
