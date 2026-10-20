# I3 SCIM verification

Status: implementation and local guarded verification complete. Root router and
composition wiring plus real-provider UAT remain release gates.

Required proof:

- ServiceProviderConfig, ResourceTypes, Schemas, Users, and Groups use SCIM
  media types and structured SCIM errors.
- Paging starts at 1, count is capped at 100, supported equality filters work,
  invalid filters/PATCH paths return SCIM 400, and stale If-Match returns 412.
- Tokens are workspace-scoped hashes with explicit rotation/revoke; login,
  agent, and user credentials cannot authenticate SCIM endpoints.
- Disable/delete removes SCIM-only membership and revokes workspace SSO
  sessions atomically while preserving manual and other-workspace access.
- Group effective role is the strongest active editor/viewer mapping and can
  never create owner access.
- Repeated provider requests are idempotent, external IDs are unique per
  workspace, and default My Workspace provisioning remains one-time.
- Existing-room boundary proof mints a room token while membership is active,
  deprovisions the user, verifies the ACL version bump, and verifies a new room
  token cannot be minted. Current-authority checks reject the older ACL version.

Local `httptest` requests and repository fixtures prove HTTP/protocol behavior;
they are fixtures, not an external IdP. A real IdP tenant remains an E2 operator
input and must not be claimed from these tests.

Verified on 2026-10-20:

- `ACCESS_TEST_DATABASE_URL=… go test ./...`: 450 backend tests passed.
- `go test ./internal/repository/postgres -run TestSCIM -count=1`: 4 guarded
  migration/provenance/deprovision/active-room tests passed.
- `npm run test:unit -- --run src/components/enterprise/scim-settings.test.tsx`:
  2 owner-settings tests passed.
- `npm run test:unit`: 192 frontend tests passed across 46 files.
- `go build ./...`: backend build passed.
- `npm run build`: TypeScript and production Vite build passed.
