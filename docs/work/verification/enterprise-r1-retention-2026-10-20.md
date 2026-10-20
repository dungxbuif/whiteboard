# R1 retention verification

Status: implementation and focused verification complete; root router/worker/UI
wiring remains pending. Production deletion remains disabled.

The verification gate covers default cutoffs, the seven-day activation delay,
policy-revision cancellation, legal-hold races, bounded/idempotent purge, owner
authorization, exact mutation retry, comment redaction pagination, and proof
that canonical collaboration history is outside the deletion inventory.

Required evidence is a guarded disposable PostgreSQL suite, focused use-case and
handler tests, frontend owner/mobile/keyboard tests, production build, and a
dry-run worker result containing identifiers/counts only. No local fixture is
evidence that production retention was enabled.

Current local evidence: four use-case tests, two strict handler tests, eight
guarded PostgreSQL tests (including migration up/down, defaults, seven-day
activation, exact retry, hold/policy races, comment redaction, automatic-version
limits, feature disable and canonical-history preservation), the three-case
frontend settings suite, the full 407-test Go suite across 14 packages, `go
vet`, the full 170-test frontend suite across 43 files, and a TypeScript 7/Vite
production build. Re-run these counts after root integration because I2/V2 are
being developed concurrently in the shared worktree.
