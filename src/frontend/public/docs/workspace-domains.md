# Workspace domains API

Owners manage exact verified domains under `/api/v1/workspaces/{workspace}/domains`.

## Create a challenge

`POST /workspaces/{workspace}/domains`

```json
{"domain":"example.com","requestId":"UUID"}
```

The response includes `dnsName`, `txtValue`, and `challengeExpiresAt`. Publish the value exactly at the returned name within 24 hours. The challenge contains 32 random bytes and is returned only once.

## Verify and capture

Send `{"requestId":"UUID"}` to:

- `POST /workspaces/{workspace}/domains/{domainId}/verify`
- `POST /workspaces/{workspace}/domains/{domainId}/capture`
- `POST /workspaces/{workspace}/domains/{domainId}/revoke`

Verification requires an exact TXT match at `_whiteboard-verification.<normalized-domain>`. Capture is a separate authenticated owner decision and repeats that exact DNS lookup immediately before the state change. The original challenge must still be unexpired. A missing proof leaves the domain verified for retry before expiry; an expired challenge requires a new challenge. Capture returns `dataMoved: false` and does not link email identities, merge users, change memberships, or move data.

An expired pending claim may be atomically reclaimed by another workspace. Verified, captured, and revoked claims cannot be taken over.

List current records with `GET /workspaces/{workspace}/domains`. All endpoints require an authenticated workspace owner.

`GET /identity/domains/discover?domain=example.com` is public and returns only a captured workspace route hint. It does not expose users, memberships, email addresses, or pending verification claims.
