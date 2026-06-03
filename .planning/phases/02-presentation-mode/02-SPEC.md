# Phase 2 Spec: Presentation Mode

## 1. Goal
Implement a high-fidelity presentation mode that mirrors Excalidraw Plus functionality, allowing users to define frames as slides and navigate through them.

## 2. Requirements
- **REQ-01: Frame Management:** Users can create, resize, and label "Frames" in Excalidraw that act as slide boundaries.
- **REQ-02: Presentation UI:** A dedicated mode that hides all editing tools and focuses on the current frame.
- **REQ-03: Navigation:** Support for Next/Prev slide via keyboard (Arrows/Space) and UI controls.
- **REQ-04: Slide List:** A sidebar or overlay showing all frames in order for quick jumping.
- **REQ-05: Persistence:** Frame positions and order must be saved in the Scene JSON.

## 3. Acceptance Criteria
- [ ] Frames can be created and ordered.
- [ ] Presentation mode activates/deactivates cleanly.
- [ ] Keyboard navigation works between frames.
