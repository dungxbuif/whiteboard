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

Mutation idempotency is durable rather than process-local. Begin, verify,
capture, and revoke write a receipt in the same PostgreSQL transaction as the
state change. The receipt binds workspace/request ID, principal, action, and a
canonical payload hash, and stores the exact response under AES-GCM. An exact
retry after a lost response returns that stored response; in particular, begin
returns the original TXT challenge without rotating it. Rebinding any field
returns `409 REQUEST_ID_REUSED`. The UI keeps an ID across uncertain transport
failure and retires it after a definitive response or changed input.

Verification plan: usecase tests cover normalization, owner denial, expiry,
replay, conflict, parent/subdomain separation and explicit capture; guarded
PostgreSQL tests cover uniqueness, workspace scoping, migration, encrypted
receipts, timeout-after-commit replay, and binding conflicts; handler and
frontend tests cover strict bodies, request-ID forwarding, uncertain retries,
routes, and owner UI. Real public TXT proof is recorded only during release UAT.

Reconciliation: migration `000024` persists the domain challenge digest plus an
encrypted replay receipt and enforces one exact normalized domain globally.
Backend tests cover the state and trust boundaries; the owner UI displays the
one-time record and requires a separate capture action. Architecture, user,
public agent-readable, and evidence documents were added. Router and server
bootstrap wiring landed separately in root integration commit `1174f36`.
