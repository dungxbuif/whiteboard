---
artifact_type: detail_design
id: EXC-015-DESIGN
status: approved
ticket: EXC-015
approval: user-requested-project-completion
---

# EXC-015 Detail Design: Frame Persistence

Frames are identified from Excalidraw elements with `type: "frame"`. Their
ordered IDs are stored inside scene JSON as `presentationOrder`. A sync helper
preserves existing order for surviving frames and appends newly created frames
in canvas order, preventing deleted frames from remaining in a slide deck.

The existing debounced scene save remains the persistence boundary, so no new
API or database migration is required. Tests cover ordering and save payloads.
