# I1 workspace domain verification — implementation evidence

Status: implemented and wired; public DNS operator UAT remains an integration
step. Injected resolver tests prove application behavior only.

The workspace owner starts an exact IDNA-normalized domain challenge. The server
stores only a SHA-256 digest of a 32-byte random value, expires it after 24 hours,
and verifies only the exact `_whiteboard-verification.<domain>` TXT name. Verify,
capture, and revoke run under the workspace lock. Capture is an explicit owner
action after verification; it never moves users, scenes, or other workspace data
and DNS evidence alone never grants membership.

Review remediation requires current proof at capture: the service repeats the
exact TXT lookup and expiry check inside the workspace transaction. Missing DNS
leaves the record verified for retry; expiry requires a new challenge. Every
mutation rechecks owner status after acquiring the workspace lock. An expired
pending row can move to one competing workspace under a PostgreSQL row lock;
verified, captured, and revoked rows remain non-transferable.

Verification plan: usecase tests cover normalization, owner denial, expiry,
replay, conflict, parent/subdomain separation and explicit capture; guarded
PostgreSQL tests cover uniqueness, workspace scoping and migration; handler and
frontend tests cover strict bodies, routes and owner UI. Real public TXT proof is
recorded only during release UAT.

Reconciliation: migration `000024` persists only the challenge digest and
enforces one exact normalized domain globally. Backend tests cover the state and
trust boundaries; the owner UI displays the one-time record and requires a
separate capture action. Architecture, user, public agent-readable, and evidence
documents were added. Router and server bootstrap wiring landed separately in
root integration commit `1174f36`.
