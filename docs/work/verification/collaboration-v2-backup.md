# Collaboration v2 preflight database restore

Verified on2026-10-17Asia/Ho_Chi_Minh, before production deployment. Production remains v1 at migration6, clean. This is a preflight backup; create a fresh cutover backup if production changes before migration.

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

## Limits and next steps

This is database backup/restore and forward-migration proof, not the replacement-scene recovery drill or an S3 object backup. Source currently has no asset rows; preserve existing S3 data regardless. Private backup remains on VM100, so independent off-host disaster-recovery storage is not demonstrated. Keep the current release images/config and take a fresh cutover backup before deployment. Public canary, restart/replay, browser roles, voice, native MCP export and full release review remain required.

See [release checklist](../../operations/RELEASE_CHECKLIST.md) and [replacement-scene recovery](../../operations/collaboration-recovery.md).
