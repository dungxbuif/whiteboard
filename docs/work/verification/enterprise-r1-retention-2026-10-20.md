# R1 retention verification

Status: security remediation and focused verification complete; root
router/worker/UI wiring remains pending. Production deletion remains disabled.

The verification gate covers default cutoffs, the seven-day activation delay,
policy-revision cancellation, legal-hold races, bounded/idempotent purge, owner
authorization, exact mutation retry, comment redaction pagination, and proof
that canonical collaboration history is outside the deletion inventory.

The remediation gate additionally proves that a young notification survives an
old activity event, a feature disable committed after candidate scan wins under
the workspace lock, leading held rows cannot consume the batch limit, scene
holds protect every supported scene-owned row, and manual versions remain
indefinite unless `manualVersionDays` is explicitly configured. Pinned versions
remain indefinite under every policy.

Required evidence is a guarded disposable PostgreSQL suite, focused use-case and
handler tests, frontend owner/mobile/keyboard tests, production build, and a
dry-run worker result containing identifiers/counts only. No local fixture is
evidence that production retention was enabled.

Current local evidence: five use-case tests, three strict handler tests, and 13
guarded PostgreSQL tests. The database suite includes migration up/down,
defaults, seven-day activation, exact retry, hold/policy races, comment
redaction, automatic/manual-version limits, feature-disable-after-scan,
held-prefix progress, notification cascade safety, scene holds, and canonical
history preservation. The four-case frontend settings suite, the full 174-test
frontend suite across 43 files, TypeScript 7/Vite production build, the full
420-test Go suite across 14 packages, and `go vet` pass.
