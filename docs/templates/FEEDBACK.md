---
artifact_type: user_feedback
id: FB-000
status: raw
owner: shared
human_fields:
  - raw_feedback
  - source_user
  - business_impact
ai_fields:
  - sentiment_analysis
  - triage_classification
  - root_cause_hypothesis
  - recommended_actions
shared_fields:
  - status
  - trace
trace:
  backlog_item: TBD
  requirement: TBD
  ticket_or_bug: TBD
---

# User Feedback Details

Use this template for complex or long-form user feedback that requires deep analysis before triage (e.g., a massive support ticket, survey results, or a detailed complaint). For simple feedback, just use the row in `docs/work/FEEDBACK_LOG.md`.

## Status

- ID: FB-000
- Status: raw | triaged | converted | closed
- Source: TBD
- Updated: TBD

## 1. Raw Feedback

Human/System fill:
- [Paste raw text, screenshots, or transcripts here]

## 2. Business Impact & Urgency

Human fill:
- Severity: TBD
- Number of users affected: TBD
- Financial/Reputation impact: TBD

## 3. Analysis & Triage (AI Fill)

- Sentiment: TBD
- Classification: Bug / Feature Request / Enhancement / UX Issue / Question
- Core Problem / Root Cause Hypothesis: TBD
- Recommended Next Steps: TBD

## 4. Conversion & Traceability

- Action taken: [Created Ticket | Created Bug | Ignored | Answered]
- Artifact ID: TBD
