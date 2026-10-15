---
artifact_type: detail_design
id: EXC-016-DESIGN
status: approved
ticket: EXC-016
approval: user-requested-project-completion
---

# EXC-016 Detail Design: Frame Tool Activation

The embedded Excalidraw editor keeps its native frame tool available and
derives `presentationOrder` from every `onChange` callback. This means a frame
created, duplicated, or deleted through the native toolbar is synchronized
without a second editor state machine.
