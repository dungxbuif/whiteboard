# Workspace feature policy

Workspace owners can stage enterprise capabilities without changing a person's
scene role. Comments, activity, versions, libraries, collections, presentation,
screen sharing, enterprise identity and retention each have an explicit
workspace flag.

Public link comments are opt-in per link. A view link with comments enabled can
comment only; it does not gain drawing or version permissions. Public links
cannot read workspace activity or version history.

Policy API revisions are decimal strings. Update requests require a UUID
`requestId` and the current `expectedRevision`. On
`REVISION_CONFLICT`, fetch the policy again before retrying. Reuse the same
request ID only when retrying the exact same request.
