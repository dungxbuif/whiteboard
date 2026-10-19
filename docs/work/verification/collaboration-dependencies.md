# Collaboration v2 dependency triage

Status: targeted dependency patch locally verified, 2026-10-16; remaining release gates below are open. Selections were recorded before package edits. This is part of E1 release validation, not a claim that an audit finding has been exploited.

## Evidence

B1's saved npm audit report contains 29 dependency entries (3 low, 13 moderate, 13 high, no critical). A dependency entry can aggregate several upstream advisories; it is not a count of distinct exploitable application defects. The B1 baseline workspace lock retained Excalidraw 0.18.1 and its original direct pins; E1 keeps Excalidraw while patching the selected dependencies below. No entries name Yjs, lib0, Zod or ELK.

| B1 baseline installed path | Observation | Release action |
| --- | --- | --- |
| Frontend axios 1.17.0 | Audit includes browser/prototype gadgets and server-only adapters. Maintainer patches identify 1.20.0. | Pin a verified patched 1.x version; run HTTP/store tests and authenticated browser flows. Do not infer every server advisory is reachable in the browser. |
| Excalidraw nanoid 3.3.3; Mermaid adapter nanoid 4.0.2 | Exact upstream pins retain several generator defects. | Test narrowly scoped overrides to patched compatible branches (3.3.18 and a verified 5.x fixing the full reported range). Verify conversion, native IDs and browser rendering before accepting the cross-major adapter override. Go public aliases use crypto/rand and do not import this npm package. |
| lodash-es 4.17.21 | Parser subtree includes affected prototype operations. | Test a patched 4.x override; validate all diagram fixtures and actual browser conversion. |
| KaTeX 0.16.47 | Audit reports trust-policy bypass after pre-existing prototype pollution. | Inspect actual Mermaid/math reachability and patched compatibility before choosing override or documented isolation. Do not silently change parser behavior. |
| socket.io-client 2.5.0 / parseuri | Legacy v1 client includes reported ReDoS dependency. | B3/B4 replacement should remove the old dependency after all v1 fallback policy is explicit. Do not upgrade to incompatible Socket.IO 4 while the Go v1 server remains. |

2026-10-19 retirement decision: B4 real-stack presence/follow/audio proof is green and the v1 fallback is HTTP content/save only. Root will now remove the unused browser and Go Socket.IO stacks plus the obsolete proxy location, while retaining the scoped TURN configuration endpoint and custom `/collab/` WebSocket. Confirm the final npm/go dependency graphs contain no Socket.IO package before release.
| PostCSS 8.5.15; Vitest 4.1.8 | Reported source-map read and test mock path traversal have patched versions. | Patch within existing major versions and rerun build/test tooling. Production images must not expose Vite/Vitest servers. |
| Tailwind/chokidar/braces/fast-glob, Sass, browserslist, esbuild | Mostly build/test dependency chains; npm suggests disruptive major changes for some entries. | Classify exact reachable inputs and production-image contents. Keep untrusted diagram compilation out of these build tools. Avoid an unreviewed Tailwind 4 migration solely to silence aggregate audit output. |

## Primary references

- [Axios maintainer advisory](https://github.com/axios/axios/security/advisories/GHSA-vh66-26gq-q6x8): affected fetch behavior needs an existing prototype-pollution source; patched 1.20.0.
- [NanoID 3.3.18 release](https://github.com/ai/nanoid/releases/tag/3.3.18): current 3.x correction for zero-size custom generators. Runtime IDs in this app use fixed positive lengths, but patched upstream code is preferable to retaining known defects.
- [Lodash maintainer advisory](https://github.com/lodash/lodash/security/advisories/GHSA-f23m-r3pf-42rh): patched 4.18.0; array-path deletion of prototype properties.
- [PostCSS maintainer advisory](https://github.com/postcss/postcss/security/advisories/GHSA-fxqj-rqcc-2cmp): patched 8.5.23; attacker CSS/source maps are the relevant input boundary.

## Acceptance and documentation

Regenerate the root lock through the single assigned package owner. Verify clean installation, workspace/unit/build suites and browser render/export parity against that exact lock. Run a fresh audit after removing the legacy runtime; document residual packages, their actual exposure, and any necessary containment. An empty audit report is not a substitute for these checks.

This note changes no public behavior, API or runtime configuration. Public dependency notices and operations documentation must be reconciled with the actual selected release versions after remediation.

## E1 selected remediation (before implementation)

Approved scope: [phase](../phases/PHASE-COLLAB-V2.md) and [programme](../../superpowers/plans/2026-10-16-collaboration-v2.md), E1 dependency brief. Only npm manifests/root lock and this note change; the controller owns release/context/public operations reconciliation.

Registry metadata checked with Node 24.19.0 / npm 10.1.0 on 2026-10-16:

| Selection | Compatibility and scope |
| --- | --- |
| axios 1.20.0 | Direct frontend 1.x patch-line update; covers all ranges in the saved audit. Browser HTTP/store and authenticated browser flows must pass. |
| postcss 8.5.23 | Direct frontend pin and root override for all transitive 8.x copies; fixes both reported source-map read ranges without changing Tailwind 3. |
| vitest 4.1.12 | Root, frontend and diagram-kit pins stay consistent; patched mocker 4.x, no major tooling migration. |
| lodash-es 4.18.1 | Registry marks 4.18.0 as a bad deprecated release; select nondeprecated 4.18.1. Root override retains 4.x API for the external Chevrotain 11 parser subtree. Mermaid bundled lodash source maps identify 4.18.1; this override does not rewrite that code. |
| @excalidraw/excalidraw → nanoid 3.3.18 | Both production and development entry/chunks contain external NanoID imports. Calls use `nanoid()` with default positive size; scoped 3.x patch replaces actual imported code. |
| @excalidraw/mermaid-to-excalidraw → nanoid 5.1.16 | Adapter 2.2.2 parser/converter imports only named `nanoid`, called without arguments. Verified ESM API and Node >=18 compatibility permit the narrow 4→5 override; browser conversion proof required. |

Keep Excalidraw 0.18.1, Tailwind 3.4.17, TypeScript 7.0.2 and Yjs 13.6.33 unchanged. No audit force, broad dependency refresh, Socket.IO migration or parser feature change.

### Reachability inspected before edits

Excalidraw production `index.js` / `chunk-K2UTITRG.js` import NanoID and dynamically import the Mermaid adapter. Adapter `dist/parseMermaid.js` imports `mermaid`; package exports resolve to `dist/mermaid.core.mjs`. Its core math chunk imports `katex` externally for `$$...$$` labels. This is real browser runtime reachability through Excalidraw's Mermaid dialog; browser-only usage is not proof of isolation. The separate Mermaid standalone distributions embed KaTeX 0.16.47 and lodash-es 4.18.1 according to source maps; dependency overrides alone cannot patch those files. They are not the selected `mermaid` package export in this app.

KaTeX 0.16.47 remains affected by [the maintainer trust advisory](https://github.com/KaTeX/KaTeX/security/advisories/GHSA-238p-pmpm-9mq7), patched in 0.18.2. Crossing the 0.16→0.18 compatibility line and math rendering behavior is outside this narrow patch slice. D4/E1 must verify parser trust/sanitization under hostile input and decide a compatible upstream update or effective isolation; no claim that browser runtime is safe solely because exploitation also needs existing prototype pollution. Diagram-kit D2's constrained native Mermaid parser does not import upstream Mermaid or KaTeX, but it does not remove Excalidraw's own dialog.

Build-only findings include PostCSS source maps, Vitest mock servers, Tailwind/chokidar/braces/glob recursion, browserslist configuration/stats and esbuild's Windows development-server read. The frontend final Docker stage copies static `dist` into nginx, not node_modules; compiler inputs remain trusted repository source/config. Excalidraw declares Sass as a dependency even though application imports do not execute Sass in the browser. E1 must inspect actual final images and Node runtime dependency surfaces when Docker/workspace installation is reconciled, and must not expose development/test servers. Legacy Socket.IO/parseuri remains browser runtime and is explicitly reserved for B3/B4 removal.

### Verification plan

Regenerate only the root workspace lock, then `npm ci` from `src`; run contracts/runtime/diagram suites, diagram build, frontend unit/build, and all 14 existing Chromium flows with their golden assertions. Record a new full audit, exact installed packages and before/after diff. Meaningful adapter conversion verification must exercise patched native IDs and rendered diagrams. The frontend standalone lock and current frontend-only Docker context remain incompatible with the root workspace installer; they are not equivalent and must be retired/reworked before release by the controller's deployment slice. No claims of Docker parity are made by this dependency task.

Additional primary evidence: [Vitest advisory](https://github.com/advisories/GHSA-82fw-gwwq-j7x9), [NanoID negative-size advisory](https://github.com/advisories/GHSA-28wg-ghj8-5hjv). Registry exact-version lookups are recorded in the E1 task report.

Resolver evidence: initial install/ci retained stale override entries and a frontend Axios lock node without resolved/integrity metadata. `npm ls` correctly failed; initial suite runs are not final patched-package proof. Targeted update corrected overrides. Explicit workspace Axios installation will correct its stale node; final clean install and suites must be repeated.

## E1 final result — 2026-10-16

The root workspace installer now resolves Axios **1.20.0**, PostCSS **8.5.23**, Vitest / @vitest/mocker **4.1.12**, lodash-es **4.18.1**, Excalidraw-scoped NanoID **3.3.18**, and adapter-scoped NanoID **5.1.16**. PostCSS independently resolves NanoID 3.3.20 through its existing compatible range. Excalidraw 0.18.1, Mermaid 11.17.2 / adapter 2.2.2, KaTeX 0.16.47, Tailwind 3.4.17, Vite 7.3.5, Yjs 13.6.33, lib0 0.2.119, Zod 4.6.7, TypeScript 7.0.2 and React 19.2.7 are unchanged.

Fresh full `npm audit --json` reports **17 dependency entries: 4 low, 5 moderate, 8 high, 0 critical**, versus B1's29 (3 low,13 moderate,13 high). No final entries name Axios, form-data, NanoID, lodash-es, PostCSS or Vitest. The low-count increase reflects aggregate dependents' changed severity, not a clean runtime safety claim. The selected narrow updates also resolve Axios's required form-data4.0.6. No unrelated existing package version changed outside these target families; lock movement/metadata is installer-generated.

### Installer correction and final proof

Node24.19.0 / npm10.1.0 initially preserved stale lock entries despite new declarations and a successful `npm ci`; `npm ls` failed and proved the mismatch. Targeted updates corrected overrides. Explicit `npm install axios@1.20.0 --workspace ./frontend --save-exact --ignore-scripts --no-audit` still preserved the old nested node and rewrote the direct manifest to1.17.0. The controller approved bounded resolver hygiene after reviewing evidence: remove only the confirmed stale `frontend/node_modules/axios` directory/lock node, restore the direct manifest pin, and root `npm update axios` (no workspace install). Final manifest, root lock metadata and actual frontend-resolved import all agree on1.20.0; `npm ls` exits0 after a subsequent clean `npm ci`. Initial suite runs before this correction are not counted as final patched-package proof.

All final commands ran from `src` with prefix `rtk proxy env PATH=/Users/dungxbuif/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:/usr/local/bin:/usr/bin:/bin` (the shell form exported this Node path before npm):

| Command | Final result |
| --- | --- |
| `npm ci --no-audit` | pass, exit0;406 packages installed, no lock mutation required |
| `npm ls axios postcss vitest @vitest/mocker nanoid lodash-es typescript yjs @excalidraw/excalidraw` | pass, exit0; exact resolved tree verified |
| `npm test` | pass; runtime52, contracts13, diagram40 =105 tests |
| `npm run build --workspace ./packages/diagram-kit` | pass, strict TypeScript7 noEmit |
| `npm run test:unit --workspace ./frontend` | pass,33 tests /7 files |
| `npm run build --workspace ./frontend` | pass; existing chunk-size warning remains |
| `npm exec --workspace ./frontend -- playwright test dashboard-style.spec.ts dashboard.spec.ts --workers=2 --reporter=line` | pass,14 Chromium tests, including all4 committed golden screenshot assertions; no golden regeneration |
| `npm audit --json` | expected nonzero,17 remaining entries; saved full report and classified below |
| `git diff --check` | pass |

Browser adapter probe used the exact Vite-optimized dynamic chunk imported by Excalidraw, the official `convertToExcalidrawElements`, and official `exportToSvg`. Flowchart returned5 native elements with5 unique IDs and SVG5302bytes; sequence returned14 native elements with14 unique IDs and SVG9865bytes. This checks default generated IDs and native rendering against the scoped overrides. Additional class and ER fixtures logged missing rendered DOM IDs (`classId-Animal-0`, `entity-CUSTOMER-0`) and returned one image/file each. Their generated IDs were valid; no NanoID API exception occurred. This is observed current-graph DOM/parser fallback behavior, not a demonstrated regression or proof of native class/ER compatibility; D4 owns investigation/parity. D2's separate restricted native parser still passes its supported fixtures. The one-off probe initially bypassed Vite CommonJS optimization and then assumed every sequence has group IDs; both harness errors were corrected without product changes. No new product tests merely mirror package versions.

### All remaining audit entries and release handling

| Entries | Actual boundary | Required follow-up |
| --- | --- | --- |
| `katex`, `mermaid`, `@excalidraw/mermaid-to-excalidraw` | Browser math/parser code through Excalidraw Mermaid; active core dynamically imports affected KaTeX0.16.47. Existing prototype pollution can bypass trust restrictions. | D4/E1 compatible patched upstream selection or demonstrated effective isolation; hostile math/config/sanitization checks. Standalone vendored copies require rebuilt upstream artifacts if used. |
| `socket.io-client`, `engine.io-client`, `parseuri` | Browser runtime legacy v1 client; URI regex is retained. | B3/B4 remove old client after replacement/fallback policy. No incompatible v4 upgrade here. |
| `braces`, `chokidar`, `fast-glob`, `micromatch`, `tailwindcss`, `sass` | Build/watch glob/selector inputs from trusted repository config/source. Sass is declared under Excalidraw dependencies but not imported by its browser entry. These packages can appear in install inventories without browser execution. | Preserve Tailwind3 styling. E1/E2 final image/package inspection; do not route untrusted diagram input into build tools or expose watch servers. Separate upstream remediation if trust boundary changes. |
| `postcss-selector-parser` | Tailwind/build CSS selector parsing; affected selector recursion/complexity. No live diagram compiler import. | Keep attacker CSS out of compilation; assess compatible upstream parser/tool patch separately with CSS/golden evidence. PostCSS core patch does not fix this distinct package. |
| `browserslist`, `baseline-browser-mapping` | Babel/CSS/browser target and custom stats/config processing at build time. | Trusted build configuration only; scoped upstream refresh can follow independently. No browser or scene-runtime reachability was identified. |
| `esbuild` | Vite bundling/dev server; advisory concerns Windows development-server file access. Verified build host is macOS, final frontend is nginx/static. | E1/E2 inspect final image, keep development server unexposed; compatible bundler patch separately if required. |
| `@excalidraw/excalidraw` | Aggregate parent remains high because Sass build/watch dependency; its runtime also dynamically reaches parser entries above. | Do not read parent severity as a standalone new defect; retain parser release gate and final artifact review. |

The17 entries are accounted for by the table (3+3+6+1+2+1+1). No claim of exploitability/absence of exploitability follows solely from npm's parent aggregation. Production image proof is still pending because the existing frontend-only Docker context/standalone lock is not the verified root workspace install. Controller must retire/rework them together before release; this slice deliberately preserves that visible gate.

### Documentation reconciliation

This internal note records intended selections before mutation and actual results afterward. No API, schema, permission, user flow or public configuration contract changed, so the human/AI public product guides need no behavioral edits for this slice. Existing third-party notice covers ELK and does not enumerate changed transitive versions; no new package/license is introduced. Commit `b1f2fc9` reconciles the current Vitest4.1.12 protocol reference and links the historical D2 research to this current audit. Task review approved the dependency slice; the root patched graph replaces the standalone installer as the tested source. Central verification records the existing large-chunk and NO_COLOR/FORCE_COLOR warnings as baseline follow-ups, not feature regressions. Release installation and final-image checks remain open.

## B3/D3 direct workspace dependencies

Before integration: declare frontend contracts0.1.0/Yjs13.6.33/lib00.2.119 from the existing locked graph and MCP contracts0.1.0/diagram-kit0.1.0. Add test-only fake-indexeddb6.2.6 (npm registry version checked) for IndexedDB transaction/reload isolation evidence. Root owns the install; verify lock graph and focused consumers after implementation. No production algorithm/version changes in this declaration.

## D4 renderer dependency intent

D4 verified the registry package and official GHSA-238p-pmpm-9mq7: affected KaTeX >=0.11,<0.18.2, own-property fix in0.18.2. Root pins the external core KaTeX path to0.18.2; actual hostile browser fixtures remain required and the separately vendored standalone bundle is not considered patched by this override. MCP declares Playwright1.60.0 and Excalidraw0.18.1; diagram-kit receives built ESM exports for production Node. Validate built import, browser renderer/font/network/deadline and compatibility before completion.

D4 install verification: initial npm install retained stale KaTeX0.16.47 despite override; npm ls failed ELSPROBLEMS. Targeted npm update katex repaired lock/tree; npm ls katex --all now exits0 with0.18.2. Audit count14 (1low5moderate8high), down from17; renderer compatibility/hostile fixtures pending. No broad audit fix used.

Diagram-kit ESM prerequisite verified by D4: npm run build --workspace ./packages/diagram-kit exits0 and npm test --workspace ./packages/diagram-kit passes41tests/3files. New built-import test failed without dist then passed under normal Node import with19presets. Changes are explicit .js local import specifiers and declaration/dist build, with no compiler behavior change. Renderer/MCP build integration remains in D4; frontend package declarations are covered by upcoming B3tests.

## E3 frozen graph remediation intent — 2026-10-19

The final root audit reports11 aggregate entries (1low,2moderate,8high). The production-only audit reports four high entries, all on one chain: Excalidraw0.18.1 declares Sass1.51.0, which resolves through Chokidar3 to affected Braces. The static frontend runtime copies only built assets into nginx, but the MCP renderer intentionally carries Excalidraw and therefore retains this installed chain even though it never invokes the Sass compiler or watch mode.

Registry metadata checked on2026-10-19 reports Sass1.105.2 with Chokidar5.0.0 and no Braces dependency. Add a root override for exactly Sass1.105.2 while retaining Excalidraw0.18.1, Tailwind3 and all rendering behavior. This is a transitive build-tool remediation, not an Excalidraw feature update. Before release, prove a clean lock install; zero production audit entries; collaboration/contracts/diagram/MCP tests; frontend unit/build; native MCP renderer security/lifecycle coverage; and exact final-image inventories. The full development audit may still contain build-only findings and must be recorded separately rather than hidden by `audit fix --force`.

No public API, schema, configuration or user flow changes. Human and agent-facing product guides therefore need no behavioral edit for this override; this verification note and release evidence are the affected documentation.

### E3 remediation result

The first package-lock-only install exposed the previously documented stale-lock behavior: `npm ls` rejected installed Sass1.51.0 against the new override. A targeted `npm update sass --ignore-scripts --no-audit`, followed by a fresh `npm ci`, resolved one Sass1.105.2 shared by Excalidraw and Vite. Sass now uses Chokidar5; Tailwind's existing Chokidar3/Braces chain remains development/build-only.

`npm audit --omit=dev --json` now reports **zero production vulnerabilities**. The full development graph reports9 aggregate entries (1low,2moderate,6high), all in the already documented Tailwind/browser-target/build-server tool boundary. Root npm passes149 collaboration,43 MCP,13 contract and41 diagram tests; frontend passes93 unit tests and the production build. The MCP renderer bundle and lifecycle/security tests are included in the43-test suite. The dependency change therefore closes the shipped dormant Sass-chain finding without changing the UI or parser feature set. Final Linux/amd64 image inventory remains part of the VM100 release gate.
