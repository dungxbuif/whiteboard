---
artifact_type: detail_design
id: EXC-017-DESIGN
status: approved
ticket: EXC-017
approval: user-requested-project-completion
---

# EXC-017 Detail Design: Presentation Engine

Presentation state is local to the viewer. A Zustand store tracks whether the
scene is presenting and the active frame index. The wrapper passes
`viewModeEnabled` to Excalidraw and uses its imperative `scrollToContent` API
to center the selected frame with a short animation. A compact overlay exposes
previous/next controls and an exit action.
