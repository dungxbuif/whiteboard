---
artifact_type: feedback_log
id: FEEDBACK_LOG
status: active
owner: shared
human_fields:
  - raw_feedback
  - source
  - triage_override
ai_fields:
  - type_recommendation
  - severity
  - converted_artifact
  - notes
shared_fields:
  - feedback_items
  - status
updated: TBD
---

# Feedback Log (User Feedback Intake)

Use this file as the funnel for all raw user feedback, bug reports from end-users, and feature requests.
Feedback must be triaged here before it can be entered into the `BACKLOG.md`.

## Triage Rules

1.  **Intake:** Record raw feedback in the `Feedback Items` table. Leave Type and Converted Artifact empty initially.
2.  **Triage:** AI or PO reviews the raw text and determines the Type (`Bug`, `Feature`, `Enhancement`, `Noise/Question`).
3.  **Conversion:**
    *   If `Bug`, create a `BUG.md` in `docs/work/bugs/` and add to `BACKLOG.md`.
    *   If `Feature` or `Enhancement`, create a `TICKET.md` in `docs/work/tickets/` or update `USER_STORIES.md`, then add to `BACKLOG.md`.
    *   If `Noise`, mark status as `closed`.
4.  **Traceability:** The converted ticket/bug must include the `FB-XXX` ID in its trace links.

## Feedback Items

| ID | Date | Raw Feedback | Source | Type (Bug/Feat/Enhance/Noise) | Status | Converted Artifact | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| FB-001 | TBD | TBD | TBD | TBD | raw | TBD | TBD |
