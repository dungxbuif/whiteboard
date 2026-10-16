# Native diagram compiler dependency decision

Status: D2 compiler and D3/D4 integration/browser paths implemented and verified locally; production release remains gated by E1/E2. D2 sections below retain historical investigation evidence; the final D4 section records current browser findings. Trace: [D plan](../superpowers/plans/2026-10-16-collaboration-agents.md), [phase](../work/phases/PHASE-COLLAB-V2.md).

## Behavior and boundary

`@collab/diagram-kit` provides `listDiagramTypes`, `getDiagramTemplate`, `validateDiagram`, `compileDiagram` and a deliberately restricted `compileMermaid`. Compilation is pure: validation precedes layout and produces native editable Excalidraw elements, asset references, bounds and actionable warnings. It never writes a scene. D3 owns scoped MCP discovery and mutation; D4 owns bounded browser exports.

All 19 requested presets have strict data schemas and canonical examples. Graph-family presets use ELK layered orthogonal routing; sequence/timeline/journey/swimlane have ordered layouts; trees reject parent cycles. IDs encode instance plus semantic identity, including text, edges and groups. Repeated inputs preserve IDs; different instance IDs are disjoint. Labels are plain text. Limits: 500 nodes, 1,000 edges, 50,000 total input characters, 2,000 characters per label.

## Sources and decisions before imports

| Dependency/reference | Pin/license | Decision |
| --- | --- | --- |
| [ELK JS](https://github.com/kieler/elkjs), [layered options](https://eclipse.dev/elk/reference/algorithms/org-eclipse-elk-layered.html) | registry queried 2026-10-16: `elkjs` 0.12.0, EPL-2.0 OR GPL-3.0-or-later | Use bundled layout runtime; choose EPL-2.0, retain upstream notices in distribution, do not modify upstream. No workers/network or plugin execution. |
| [Excalidraw skeleton API](https://docs.excalidraw.com/docs/@excalidraw/excalidraw/api/excalidraw-element-skeleton) | installed `@excalidraw/excalidraw` 0.18.1, MIT | Consume installed types via shared contracts. Official converter with `regenerateIds:false` is available through an injected browser adapter. Plain Node import experimentally fails on upstream JSON import attributes, so server normalizes only the supported rectangle/diamond/ellipse/text/line/arrow skeleton subset without importing the React bundle. Record browser parity as D4 proof. |
| [Official Mermaid adapter](https://github.com/excalidraw/mermaid-to-excalidraw) | existing upstream transitive 2.2.2, MIT | Research reference only for D2; it requires browser rendering and can produce image fallback. D2 accepts an explicit small flowchart/sequence grammar and rejects unsupported syntax with a DSL alternative. Do not claim complete Mermaid support. |
| [Community library catalog](https://github.com/excalidraw/excalidraw-libraries) | per-item provenance required | Do not ingest unknown assets. Ship original native semantic motifs with content hashes and provenance manifest; external library candidates remain opt-in review references. |
| [Official Excalidraw MCP](https://github.com/excalidraw/excalidraw-mcp) | reference only | Discovery/stream preview reference; reuse no service code or hosted credentials. |

No provider key, external fetch, HTML labels, SVG ingestion or arbitrary URLs are involved. Input is JSON-only with unknown properties rejected. Native fields pass the shared serializable scene validator before return. An ELK deadline limits awaiting; process-level cancellation of pathological layout belongs to the D4 worker boundary.

## Verification plan

Red-green tests cover discovery, schemas/examples for all presets, native element output, stable/disjoint IDs, resolved bindings, finite geometry, shape non-overlap, cycles, missing endpoints, Unicode, long labels, strict unknown-field/oversize rejection, explicit Mermaid capability errors and shared scene-schema compatibility. Run from `src`: `rtk npm test --workspace ./packages/diagram-kit`, plus workspace build. Browser visual/export verification remains D4/E1 and is not implied by compiler unit proof.

## Evidence

- Baseline Node import of installed Excalidraw 0.18.1 failed: `open-color.json needs an import attribute of type json`.
- Tested installation: `elkjs` 0.12.0, `@excalidraw/excalidraw` 0.18.1 (shared type dependency), TypeScript 7.0.2, Vitest 4.1.8. Root lockfile owned by B1. ELK package integrity: `sha512-YZcKynxVxYoKIOEpywEPwCFdg+BTbxQRNf3pbwdDCvc8O3kQD8bmIwSxKU1eOTVc4Xo+VG9Te+575mlfvOrhEQ==`.
- Initial red proof: 24/24 tests failed on stub behavior, not import/infrastructure errors. Later red cases independently reproduced ignored Mermaid directives, group boundaries enclosing nonmembers, and endpoint-positioned edge labels; each passed after its targeted implementation.
- Final `npm test --workspace ./packages/diagram-kit` via `rtk proxy env PATH=<bundled Node + npm paths>`: **39/39 pass**, 2026-10-16. `npm run build --workspace ./packages/diagram-kit`: **pass**, TS7 noEmit.
- Coverage includes 19 presets × two directions × three themes; strict shared scene validation runs on every compiled result. Compiler fixtures verify group separation, semantic namespaces, edge-label placement, invalid references, duplicate IDs, cycles, Unicode and long labels.
- `npm audit --workspace ./packages/diagram-kit --omit=dev --json`: **nonzero**, 14 advisories (six high, six moderate, two low) in the inherited Excalidraw/Mermaid/Sass/NanoID subtree. No `elkjs` advisory was listed. This is workspace dependency risk, not a clean-audit claim. D2 does not import those upstream browser/parser runtimes; browser renderer hardening/upstream remediation remains D4/E1. No automatic dependency changes applied.
- Provenance: `src/packages/diagram-kit/curated-libraries.json` pins SHA-256 for original `native-motifs.json`; external library catalog is a review candidate, not imported art. Upstream ELK license is retained under `licenses/elkjs-LICENSE.md` and notices under `THIRD_PARTY_NOTICES.md`.

## Native field/default mapping and approved deviation

The controller approved a bounded Node-native normalizer after the direct-import incompatibility was reproduced. No upstream converter implementation was copied. Field provenance is installed `@excalidraw/excalidraw` 0.18.1 `dist/types/excalidraw/element/types.d.ts` and the official skeleton API linked above.

| Field group | D2 mapping |
| --- | --- |
| Identity/reconciliation | Instance fingerprint + role namespace + semantic fingerprint; text IDs derive from their container. Deterministic seed/versionNonce, version 1, updated 0, index null; duplicate native IDs are rejected. Fingerprints are non-security identifiers. |
| Base geometry/state | Finite x/y/width/height from layout; angle 0, opacity 100, deleted/locked false, frameId/link null; per-theme stroke/fill/roughness/roundness. |
| Labels | Native text, originalText, fontSize 18, installed fontFamily 1 (sketch) or 2 (clean/pastel), lineHeight 4/3, explicit wrapped geometry, autoResize false, textAlign center, verticalAlign middle. Bound box labels use containerId + inverse boundElements; boundary labels are free native text. |
| Connectors | Native arrow, local points, arrowheads with ER/class semantics, elbowed false, lastCommittedPoint null; graph endpoint start/endBinding + inverse arrow backlinks. Ordered sequence messages/lifelines use unbound native arrows. |
| Groups/files | Editable compound group boundaries come from ELK hierarchy; shape and label group IDs are instance-scoped. No image assets are emitted, so files is empty and no file-remap path is claimed. |
| Browser adapter | `convertWithExcalidraw(elements, convertToExcalidrawElements)` requests regenerateIds false, repairs backlinks, rejects count/ID changes and reruns shared validation. Actual converter/render parity is pending D4, not simulated by a mock test. |

Sequence and user-journey retain their conventional ordered orientation (participants/stages across, messages/actions below). Direction changes layered graph, timeline and swimlane layouts. The layout promise has a ten-second deadline; in-process ELK cannot be forcibly interrupted by that timer. A hard CPU cancellation boundary needs D4 worker/process isolation.

## Community integration review

Reviewed primary repository documentation for [yctimlin/mcp_excalidraw](https://github.com/yctimlin/mcp_excalidraw), [debu-sinha/excalidraw-mcp-server](https://github.com/debu-sinha/excalidraw-mcp-server), [excaliwow/mcp](https://github.com/excaliwow/mcp), and [hangarbay/excalidraw.mcp](https://github.com/hangarbay/excalidraw.mcp). They provide useful element CRUD, live canvas, browser conversion and export interface references. No community service/runtime code was imported: this application's Go authority, shared revision/grant path and offline persistence remain mandatory. Hosted client wrappers do not establish that a hosted application's backend is reusable public source. Only pinned ELK is a new D2 runtime dependency; native integration uses the existing Excalidraw element contract.

Current workspace patch versions and later audit findings are recorded in [E1 dependency remediation](../work/verification/collaboration-dependencies.md). The D2 audit above is historical evidence, not the current release result. E1 browser probes found class/ER native fallback limitations in the current upstream adapter; D4 must reject raster fallback or implement a verified native path.

## D4 intended dependency and process boundary — 2026-10-16

Renderer uses the installed Excalidraw 0.18.1 official browser exporter and Playwright 1.60.0, bundled with Vite 7.3.5. Diagram-kit will emit Node ESM with .js local import specifiers so the private service uses built packages. No runtime plugin is loaded. Current curated shape provenance remains in curated-libraries.json.

The [KaTeX maintainer advisory](https://github.com/KaTeX/KaTeX/security/advisories/GHSA-238p-pmpm-9mq7) identifies versions >=0.11.0,<0.18.2 and confirms 0.18.2 fixes inherited settings/metadata/namespaces. Registry lookup on 2026-10-16 confirms 0.18.2 and integrity sha512-3snve4y0SXTMequLim1FMiPvLGElySam0blN5xQZBqEY3lOJz1TnuaaiugnBZBqHzlSAGmFTYL7MCQkORuLKCA==. Root installer will override the externally imported core dependency; standalone Mermaid distributions with vendored KaTeX are excluded. Planned proof includes normal math and inherited trust plus resource/link payloads, prototype cleanup, Mermaid configuration rejection and native editable fixtures. Compatibility is not yet claimed.

## D4 implementation evidence

The build now emits diagram-kit ESM/declarations with `.js` import specifiers and succeeds under ordinary Node ESM. The renderer uses the pinned official Excalidraw converter/exporter in real Chromium. Restricted Mermaid remains the D2 native parser: upstream adapter fixtures are compatibility/security probes, not permission to widen its syntax or introduce image fallback. A visual regression exposed the skeleton converter's alignment-anchor interpretation of complete native text. The renderer now retains original label-box centers, fits measured text within them, and preserves stable IDs/version/versionNonce/seed/updated; repeated typed input is byte-equivalent. The saved multilingual PNG shows complete Vietnamese/Japanese labels and native connectors.

Root installed KaTeX **0.18.2** through the override and `npm ls katex --all` confirms the resolved version. Real-browser tests prove normal math and native flowchart/sequence/math adapter fixtures, inherited `trust` settings and polluted Object.prototype trust with explicit cleanup, hostile URL/image/background commands, and denial of raw unsafe output. These fixtures exercise the externally imported core KaTeX path; vendored standalone Mermaid distributions are excluded. This addresses GHSA-238p-pmpm-9mq7 on this path, not a claim that every dependency is vulnerability-free. The remaining root audit result is tracked separately in the dependency ledger.

The font manifest pins 234 upstream Excalidraw 0.18.1 WOFF2 files by SHA-256 and the Vite build verifies every byte. Notices and primary-source provenance are in `src/mcp/licenses/README.md`. In particular, the bundled Liberation Sans metadata is the legacy Ascender GPL-2.0-with-font-exception font, not assumed to be the newer OFL release. No third-party artwork was imported. Official export embeds subset fonts in SVG and loads bundled fonts before PNG rasterization.

Tests confirm hard termination of both a noncooperative Node worker and a real Chromium renderer executing an infinite loop, exact browser resource allowlisting, metadata/loopback/external/file denial, native assets and export MIME, current authority and revocation over the actual Go/Node/HTTP stack. The final D4 report records commands, counts and deployment limitations; E2 owns container-build and proxy proof.
