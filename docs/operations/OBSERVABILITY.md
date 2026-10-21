# VM100 observation and incident triage

This is a command map for an operator with VM100 access. The detailed
[deployment](DEPLOYMENT.md), [recovery](collaboration-recovery.md), and
[release checklist](RELEASE_CHECKLIST.md) remain authoritative for changes.
Collect status and bounded redacted logs; never paste environment files,
tokens, URLs carrying credentials, scene content, or database dumps into issues.

## First checks

```sh
cd /opt/apps/whiteboard
docker compose --env-file .env.prod -f docker-compose.prod.yml ps
curl -fsS https://whiteboard.dungxbuif.com/healthz
systemctl show whiteboard-recovery.service -p ActiveState -p Result -p ExecMainStatus
```

The migration job should exit 0. Backend, collaboration, feature worker, MCP,
and frontend should be healthy. Public `/healthz` should return `ok`. A healthy
MCP probe (`401` without a token) only proves the adapter is listening;
authenticated MCP read/write/export is a separate release check.

| Symptom | Inspect | Meaning / next action |
| --- | --- | --- |
| Public 502, local frontend healthy | Frontend upstream resolution and backend/collab/MCP health | A recreated upstream may need frontend reconciliation; see Homelab recovery and Docker DNS evidence. |
| Collaboration `/readyz` fails | Collab ownership, PostgreSQL/LISTEN, Go authority | The service drops readiness rather than using stale permission state; inspect current access before retry. |
| Scene duplicate `500 DUPLICATE_CLEANUP_FAILED` | Structured scene ID, phase and request ID; pending row and stage records | Preserve hidden state and follow the [cleanup procedure](DEPLOYMENT.md#upgrade-through-migration-000029). |
| Scene duplicate `503 DUPLICATE_STATE_UNCERTAIN` | Scene ID, publish state and current list | Publication may have committed; do not blindly issue a second copy. |
| MCP mutation denied or audit failure | MCP audit journal and Go authoritative operation audit | Adapter journals are observational; Go records the authoritative operation. Follow the [MCP runbook](mcp.md). |
| Voice fails while documents sync | API-issued TURN config, Coturn allocation and split DNS | The deployed relay uses the tested gateway IP; see the Homelab Coturn record. |
| Services fail after VM reboot | `journalctl -b -u whiteboard-recovery.service`, Swarm state, three external networks | Wait for correct overlay metadata; never replace overlays with bridges. |

For dashboard management, query the release-candidate migration state and
public routes only after the [release gate](RELEASE_CHECKLIST.md) authorizes a
rollout. Prior VM100 service health does not prove migration `000029` or the
new management API.
