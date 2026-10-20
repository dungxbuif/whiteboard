# Task 2 final review fixes — work note

Before implementation: scene duplication must be a server-side authenticated operation because copied images require new scene-scoped asset rows and object ownership. A new duplicate endpoint will read v2 content from the collaboration writer's bounded export projection (including un-compacted acknowledged edits), copy referenced assets with the existing staging/copy pattern, rewrite file and asset references, and create the destination scene only for an editor/owner. It will avoid the workspace lock during the writer read. The frontend will call this endpoint and guard asynchronous mutation/list results by the active workspace and request generation. Tests will cover image-bearing copies, a scene above the 256 KiB projection limit, and in-flight workspace switches. Shared public and AI-readable docs are assigned to Task 4; this note records the new contract for reconciliation.

## Shipped behavior

- `POST /api/v1/scenes/{id}/duplicate` requires a user session, `X-Workspace-ID`, scene edit access and workspace edit access. JSON body: `{ "title": string, "folderId": UUID | null }`. It returns `201` with the new Scene. The handler is registered outside the scene workspace-lock middleware so its collaboration read cannot deadlock.
- For v2 scenes, the service obtains a signed `scene:export` grant and requests the collaboration writer's `export-projection` with a 26 MiB response bound. This includes acknowledged updates that have not yet compacted into `scenes.content` and bypasses the read projection's 500-element/256 KiB limit. V1 scenes copy their stored content.
- Scene-scoped assets referenced by `content.files` are copied to new object keys and `scene_assets` rows through the existing library stager. File IDs, asset IDs, and image element `fileId` references are rewritten for the new scene. A failed stage or update removes the incomplete destination scene.
- Scene and folder list fetches and move/update/duplicate completions now check request generation and active workspace. After switching workspaces, late results cannot overwrite the new workspace's visible lists or active scene. The duplicate dialog also avoids navigating to the old workspace after a late response.

## Verification

- Focused Go tests: 270 passed (usecase, handler, Postgres repository packages).
- Full Go tests: 359 passed in 14 packages.
- Focused frontend store tests: 22 passed before the final refresh regression was added; final full frontend suite: 229 passed in 50 files.
- Frontend production build passed (`tsc -b && vite build`).
- Real-stack Playwright suite attempted; all 27 tests could not start because the local PostgreSQL role `test` does not exist. The image-bearing Postgres integration test is gated by `ACCESS_TEST_DATABASE_URL` and is skipped without its dedicated test database.

Task 4 owns the shared human and AI-readable API docs. Parent was notified of the new endpoint contract for that documentation pass.

## Follow-up review plan (before implementation)

Keep a new duplicate scene hidden until its content and scene-scoped assets are complete. On any failure, use an independent bounded cleanup context, remove staged object copies and their rows, then hard-delete the hidden scene; surface cleanup failures so an operator can distinguish them, and retain a hidden scene if object deletion fails. Guard create-folder, create-scene, and open-scene completions against workspace switches, including navigation after scene creation. Reconcile `docs/architecture/API.md`, `docs/user/dashboard.md`, `src/frontend/public/docs/dashboard-management-api.md`, and the public dashboard copy so they describe the server-side duplicate endpoint and asset handling. Verify failure/cancellation cleanup, deferred workspace responses, full Go/frontend suites and build.

## Follow-up implementation

- Duplicate creation inserts a soft-deleted destination scene. The server publishes content and clears `deleted_at` in one database update only after the collaboration read and object staging finish. Cancellation before publication triggers cleanup.
- Stage records identify every copied object. Failure compensation runs with a fresh 30-second context, deletes object keys, removes their scene-asset and stage rows transactionally, then hard-deletes the hidden scene. If object deletion fails, the hidden scene and records remain; the returned error includes the cleanup failure and scene ID for recovery. Ambiguous create failures also attempt independent hard-delete by the preallocated scene ID.
- Create-folder, create-scene, and open-scene responses now honor workspace/request generation. The New scene callback checks the active workspace before submission after import parsing and before navigation after creation.
- `docs/architecture/API.md`, `docs/user/dashboard.md`, `src/frontend/public/docs/dashboard.md`, and `src/frontend/public/docs/dashboard-management-api.md` now describe the server-side duplicate route, durable content, copied image assets, and distinct scene-create/import contract. `src/frontend/public/llms.txt` already links the AI-readable API page, so its index needed no change.
- Added failure/cancellation compensation tests, DB asset cleanup and hidden-publication assertions, deferred workspace-switch store tests, and a navigation callback test. Full Go suite: 363 passed in 14 packages. Full frontend suite: 233 passed in 51 files. Frontend production build passed. The PostgreSQL-backed cleanup/publication assertions remain gated by `ACCESS_TEST_DATABASE_URL`; the previously attempted real-stack suite could not start without the local `test` role.
