# Local development

Start with [setup](SETUP.md). Run commands from the directory shown below;
`src/package-lock.json` is the one workspace lockfile.

| Directory | Command | Purpose |
| --- | --- | --- |
| `src/backend` | `go test ./...` | API, usecase, repository, and command tests; database cases need `ACCESS_TEST_DATABASE_URL`. |
| `src` | `npm run build:contracts` | Build the shared collaboration contract before dependent packages. |
| `src` | `npm run test:unit --workspace ./frontend` | Frontend unit tests. |
| `src` | `npm run build --workspace ./frontend` | TypeScript and production frontend build. |
| `src/frontend` | `npm run dev` | Serve the local UI at `http://localhost:5173`. |
| `src/frontend` | `npx playwright test e2e/dashboard-style.spec.ts e2e/dashboard.spec.ts --workers=1` | Dashboard management browser regression. |

The default Playwright config starts Vite and excludes the real-stack suite.
Its route fixtures check UI flows; they do not prove PostgreSQL, S3, the
collaboration writer, or VM100. The [management verification](../work/verification/dashboard-management-2026-10-21.md)
records local checks; the [release checklist](../operations/RELEASE_CHECKLIST.md)
records production gates.

When changing the management API, update [the API index](../architecture/API.md),
the public [plain Markdown reference](../../src/frontend/public/docs/dashboard-management-api.md),
and the human [dashboard guide](../user/dashboard.md). Distinguish locally
checked behavior from deployed behavior in every release note.
