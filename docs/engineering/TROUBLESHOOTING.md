# Troubleshooting

| Symptom | Check | Next step |
| --- | --- | --- |
| Go database tests skip or fail at connection setup | `ACCESS_TEST_DATABASE_URL` and disposable PostgreSQL role/database | Run focused tests without the gate, or provision an isolated database. |
| Browser test cannot start | Playwright browser install, Vite port `5173`, and `src/frontend/playwright.config.ts` | Resolve the local port/browser dependency, then rerun the failed spec. |
| Dashboard Collections says unavailable | Workspace policy and `403 FEATURE_DISABLED` | This is the intended disabled state; other transport/server errors show Retry. |
| Projection returns 401, 403, or 404 | Login token, `X-Workspace-ID`, current `scene:read`, and scene reference | Recheck current access; public links and agent keys cannot call this route. Every response has `Cache-Control: no-store`. |
| Duplicate fails with `INVALID_SCENE_ASSET` | Scene-owned file row, MIME, and referenced file metadata | Repair the source asset before retrying. |
| Duplicate returns `DUPLICATE_STATE_UNCERTAIN` | Current scene list and operator state for the hidden copy | Check whether publication committed before retrying. Follow the [migration/cleanup runbook](../operations/DEPLOYMENT.md#upgrade-through-migration-000029). |
| VM100 shows 502 or a service is unhealthy | Compose health, frontend upstream resolution, and service logs | Use the [production runbook](../operations/DEPLOYMENT.md) without printing credentials. |
