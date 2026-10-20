# Workspace domain identity

Workspace domain capture is an owner-controlled, two-step trust boundary. It does not create, link, merge, or move user accounts.

## State and ownership

`workspace_domains` stores one globally unique normalized ASCII domain. Input is processed with the IDNA Lookup profile, lowercased, and stripped of one terminal dot before persistence. Parent domains and subdomains remain separate exact values.

The state machine is `pending -> verified -> captured -> revoked`:

- `pending` has a SHA-256 digest of a 32-byte random challenge and a 24-hour expiry. The raw challenge is returned once and is never persisted or logged.
- `verified` means an exact TXT record was observed at `_whiteboard-verification.<domain>` before expiry.
- `captured` records a separate, authenticated owner acceptance. Capture repeats the exact TXT lookup and requires the original challenge to remain unexpired, so DNS proof must still be current at acceptance time. DNS evidence alone cannot activate identity behavior.
- `revoked` disables the capture while preserving the unique claim and audit state.

Every operation resolves the workspace route and requires its current owner. Each mutation checks ownership once before work and again inside the existing workspace lock, closing an owner-change race. A consumed challenge cannot be replayed. Capture reports `dataMoved: false`; membership and content ownership are outside this transaction.

If capture cannot find the exact TXT proof, it fails closed with `DNS_VERIFICATION_FAILED` and leaves the domain `verified`, allowing the owner to restore DNS and retry before expiry. At or after expiry it returns `CHALLENGE_EXPIRED`; the owner must issue a fresh challenge.

An expired `pending` claim can be reclaimed by another workspace. Creation locks the exact domain row and atomically replaces its workspace, ID, digest, and expiry only when the previous row is still pending and expired. Concurrent reclaimers produce one winner; verified, captured, and revoked rows cannot be taken by another workspace.

## API

Authenticated owner endpoints are:

- `GET /api/v1/workspaces/{workspace}/domains`
- `POST /api/v1/workspaces/{workspace}/domains`
- `POST /api/v1/workspaces/{workspace}/domains/{domainId}/verify`
- `POST /api/v1/workspaces/{workspace}/domains/{domainId}/capture`
- `POST /api/v1/workspaces/{workspace}/domains/{domainId}/revoke`
- `GET /api/v1/identity/domains/discover?domain=example.com` (public; captured workspace hint only)

Mutation bodies carry a UUID `requestId`. Create also carries `domain`. Responses use `Cache-Control: no-store`; the raw TXT value appears only in the successful create response.

Discovery returns only the normalized domain and workspace route hint for a captured domain. It never returns users, memberships, email addresses, or an unverified claim.

## DNS boundary

The production adapter uses the process resolver with a three-second deadline. Lookup failures and timeouts fail closed. The resolver receives only the exact `_whiteboard-verification` host; it does not walk parents, accept wildcard proof, or query the base domain.

Fixture resolver tests prove exact-name, expiry, IDNA, ownership, conflict, and state behavior. They do not prove that a public authoritative DNS server has published a record. Production activation therefore still requires an operator or owner to publish the generated TXT value and run verification against public DNS.
