# Collaboration v2 preflight database restore

Preflight restore verification completed on 2026-10-17 Asia/Ho_Chi_Minh. A
fresh cutover backup was then verified on 2026-10-19 immediately before the
production migration.

## Backup and ownership

The running VM100 backend supplies the database connection in memory. The production database is `whiteboard` on the existing shared PostgreSQL16.14 service. Credentials were placed only in a private temporary client env file and deleted after each command; no connection URL/password was printed. No shared database/container/network was changed.

Private backup directory: `/opt/apps/whiteboard/backups/collab-v2-preflight-20261016T170034Z`, mode0700. Custom-format dump, manifest/environment copies and evidence are mode0600. The35038byte dump SHA-256 is `238c40a109b95d0814d51acc5302af339e791fddfc2cb84480cb09a5cd3d8c47`. Keep this directory outside Git and user-facing artifacts: its files contain private account and configuration data.

Captured running images:

- Backend: `sha256:aa2aca6bd49b01bd719b6b637a5db523101a7c6cfdc7aef2a9cd49efaa02cbcd`.
- Frontend: `sha256:48cb747e8e422fc1cbe9001f1874544289c730c08d49fa3841615c3947a0fd5d`.

The production backend currently has none of the four new collaboration secrets configured. Deployment must create distinct values without changing shared JWT/S3/TURN ownership by assumption.

## Actual restore and forward migration

A generated-name PostgreSQL17container on network `none`, with no published port, restored the complete dump using `pg_restore --no-owner --no-acl --exit-on-error`. It was bounded to512MiB/100PIDs. The fixture uses a generated password; no production connection is supplied to it.

Restored schema state: migration6, `dirty=false`. Every public base table's row count matches the production source count queried after restore. Counts and exact metadata are retained in the private evidence file. This count check complements successful full restoration; it does not claim an atomic comparison of a changing source or restore database-owner/ACL privileges.

The actual `migrate/migrate:v4.17.1` container then applied the repository's unmodified forward migrations7,8,9,10,11,12,13 against the isolated fixture. Final state: migration13, `dirty=false`. Existing users/workspace/folder/scene counts remain intact. Owner membership and the legacy public view link are backfilled as expected. No down migration ran. The fixture and its anonymous volume were removed; label-based inventory confirmed no restore fixture remained.

Two setup corrections were required before successful evidence: VM100 Python3.10 lacks `hashlib.file_digest`, so the completed dump was hashed in bounded chunks; initial Unix-socket readiness saw PostgreSQL's temporary bootstrap server, so the clean retry waited for final TCP loopback readiness. Neither issue touched production data. The first failed restore fixture was removed before retry.

## Fresh production cutover backup — 2026-10-19

The private cutover bundle is stored on VM100 at
`/opt/apps/whiteboard/backups/collab-v2-cutover-20261019T155632Z`. The custom
PostgreSQL dump is 35,037 bytes with SHA-256
`ee45fff0d0d1c0fd99c1ab5467e7eda3de67d8dd5eb55464262001697c00e953`.
`pg_restore --list` read the archive successfully.

The dump was restored into an isolated PostgreSQL container on network
`none`. All 14 source/restore table row counts matched and both source and
restored databases reported migration 6 with `dirty=false`. Production then
applied migrations 7–13 and reported version 13 with `dirty=false`; owner
workspace memberships were present after migration.

The rollback bundle retained the pre-cutover manifest/configuration and v1
images. Its archive is 42,890,152 bytes with SHA-256
`5db116506e05cb87b2f577bec7a8376aec38f8ba107261a3a19423295810fef0`.
The captured Linux/amd64 rollback images are:

- Backend: `sha256:35301824c0273d6528b88949b8b7a5817e4fd79096291888b45bd9ab6ed9fef2`
  (`homelab/whiteboard-backend:rollback-20261019-v1`).
- Frontend: `sha256:48cb747e8e422fc1cbe9001f1874544289c730c08d49fa3841615c3947a0fd5d`
  (`homelab/whiteboard-frontend:rollback-20261019-v1`).

No secret values, connection URL or account data are stored in this
repository. The private bundle remains on VM100 and its access mode remains an
operator responsibility.

## Limits and next steps

This is database backup/restore and forward-migration proof, not an S3 object
backup. Existing immutable S3 objects must remain available. Independent
off-host disaster-recovery storage is not demonstrated. Public browser
management checks, authenticated MCP reads, S3 write/read/delete and deep-link
persistence, controlled VM reboot recovery and deployed TURN relay allocation
have passed. Native MCP write/export, public browser media and the remaining
product canaries stay on the release checklist.

See [release checklist](../../operations/RELEASE_CHECKLIST.md) and [replacement-scene recovery](../../operations/collaboration-recovery.md).
