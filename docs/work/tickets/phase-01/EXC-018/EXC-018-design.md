---
artifact_type: detail_design
id: EXC-018-DESIGN
status: approved
ticket: EXC-018
approval: user-requested-project-completion
---

# EXC-018 Detail Design: Slide Navigator and Keyboard Support

The presentation overlay includes an ordered slide navigator with frame
labels, selected state, and move-left/move-right controls. Reordering updates
the scene's `presentationOrder` through the existing debounced save path.
Arrow keys and Space navigate while presenting; Escape exits. Inputs and
buttons are excluded from global keyboard handling.
