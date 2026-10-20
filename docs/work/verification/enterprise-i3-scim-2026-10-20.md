# I3 SCIM verification

Status: implementation and local guarded verification complete. Root router and
composition wiring landed in `dc9f3a1`; real-provider UAT remains a release gate.
The real-stack harness now supplies an ephemeral 32-byte SSO encryption key and
exact loopback callback URL so the required production configuration is exercised
without persisting test credentials.

Independent review remediation covers group-source cleanup, explicit manual
membership provenance for owner and invitation mutations, adopted-role
reconciliation, RFC paging/filter/PATCH edge cases, extension-aware discovery,
and suppression of SQL values containing SCIM PII. Verification below must be
re-run after these changes before wiring.

Final gate remediation also verifies adoption materializes active group
sources before effective-role recomputation and rejects both last-owner and
non-last-owner adoption. Owner deprovision must return retained manual access
and leave the owner membership intact.

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
- Group deletion removes its provenance rows before recomputing effective
  roles; late owner and invitation grants remain manual across deprovision.
- `count=0`, case-insensitive filter attribute spelling, pathless PATCH, and
  selective `members[value eq "…"]` removal follow RFC behavior; unsupported
  paths return SCIM 400 rather than broadening a query or mutation.
- Runtime and static schemas declare the WorkspaceAccess extension and keep
  custom role/manual state out of the core Group schema.
- Log-capture verification proves routine provisioning does not emit userName,
  displayName, or externalId values.
- Repeated provider requests are idempotent, external IDs are unique per
  workspace, and default My Workspace provisioning remains one-time.
- Existing-room boundary proof mints a room token while membership is active,
  deprovisions the user, verifies the ACL version bump, and verifies a new room
  token cannot be minted. Current-authority checks reject the older ACL version.

Local `httptest` requests and repository fixtures prove HTTP/protocol behavior;
they are fixtures, not an external IdP. A real IdP tenant remains an E2 operator
input and must not be claimed from these tests.

Verified on 2026-10-20:

- `ACCESS_TEST_DATABASE_URL=… go test ./...`: 345 backend tests passed in the
  current branch (the earlier 461 count included concurrent pre-wiring suites).
- `go test ./internal/repository/postgres -run TestSCIM -count=1`: 6 guarded
  migration/provenance/deprovision/active-room tests passed.
- `npm run test:unit -- --run src/components/enterprise/scim-settings.test.tsx`:
  2 owner-settings tests passed.
- `npm run test:unit`: 199 frontend tests passed across 48 files.
- `go build ./...`: backend build passed.
- `npm run build`: TypeScript and production Vite build passed.
