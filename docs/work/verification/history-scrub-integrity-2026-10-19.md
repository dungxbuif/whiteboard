# Git history scrub integrity repair — 2026-10-19

Status: complete. The repaired source and all subsequent release-hardening commits are on `origin/master`; the final reachable-history credential scan is clean.

GitHub Push Protection correctly blocked historical OAuth credentials. The subsequent whole-history replacement removed those credentials, but an over-broad database-value expression also rewrote four test-only `COLLAB_TEST_DATABASE_URL` fallback expressions into duplicated, invalid TypeScript/JavaScript. The current production runtime was not rebuilt or deployed from that commit, so production behavior is unaffected; the pushed source gate is invalid until repaired.

Affected source is limited to the collaboration PostgreSQL fixture, recovery integration fixture, compiled-runtime integration fixture and packaging idle-collaboration fixture. Restore each guarded environment-variable fallback to the standard loopback `*_test` URL. Do not restore any retired OAuth/JWT/database credential.

Verification requires a repository search for malformed duplicated fallback text, TypeScript/build import success, the full root npm suite against the guarded disposable test database, packaging scripts, GitHub-compatible credential-pattern scans across reachable `master` history, and a clean push of the repair. This internal verification note is both human-readable and agent-readable; no public API, schema, configuration or user behavior changes.

The repaired gate passes149 collaboration,43 MCP,13 contract and41 diagram tests. Frontend93 unit tests and the production build also pass. The four restored fallbacks accept only the existing guarded loopback `*_test` database path; their runtime safety checks remain intact.

The compiled-runtime packaging proof also passes after160.161seconds: six live heartbeat pings, zero live closes, dead-peer eviction after48.842seconds, stale voice-slot release and readiness success. A final scan of every commit reachable from `master` finds no retired Google OAuth client-ID or `GOCSPX-` secret pattern. GitHub accepted the rewritten history and later commits through `d891388` on `origin/master`, closing the Push Protection incident without bypassing the guard.
