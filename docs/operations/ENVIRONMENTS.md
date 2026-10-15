# Environments

| Environment | Purpose | URL | Owner | Notes |
| --- | --- | --- | --- | --- |
| local | Development and mocked browser verification | `http://localhost:5173` | shared | `src/.env`; Vite proxies API to `:8080` |
| production | Public self-hosted release | `<public-origin>` | human | Docker Compose; HTTPS termination is required |

## Secrets

Document secret names only. Do not store secret values.

- `DATABASE_URL`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_REGION`
- `FRONTEND_URL`
- Password users are provisioned with the backend `userctl` CLI; no Google OAuth credentials are required.
- `AI_API_KEY`, `AI_BASE_URL`, `AI_MODEL_FAST`, `AI_MODEL_REASONING`
- `TURN_URL`, `TURN_USERNAME`, `TURN_PASSWORD`, `TURN_AUTH_SECRET`, `TURN_REALM`, `TURN_MIN_PORT`, `TURN_MAX_PORT`
