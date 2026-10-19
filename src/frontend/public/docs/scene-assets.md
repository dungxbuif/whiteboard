# Images, uploads and realtime credentials

Backend and browser integration are implemented and verified in the local
real-stack release candidate and deployed on VM100; the production `whiteboard`
bucket write/read/delete canary passes.

Upload PNG, JPEG, GIF or WebP images between 1 byte and 10 MiB. Public viewers can display existing images; public editors can upload only to their shared scene. Existing SVG and AVIF images remain readable. A batch supports up to 100 distinct file IDs.

For an authenticated user, call `POST /api/v1/scenes/{sceneId}/assets/upload-capability` with a login bearer token and `X-Workspace-ID`. A public editor uses their scene-session bearer token at the same endpoint; a room token is also accepted. Send `{ "assets": [{ "fileId": "local-image-id", "mimeType": "image/png", "byteSize": 1234 }] }`. UUID and public alias scene paths are accepted. The response contains `urls`, keyed by file ID, and `expiresAt`. Each entry includes `assetId`, `uploadUrl`, `storageKey`, `expiresAt`, and `headers`.

PUT the original image Blob to `uploadUrl`, using the returned Content-Type header. The signed URL enforces the declared MIME and exact byte count; browsers provide Content-Length automatically. Complete the upload promptly: URLs live at most 60 seconds and may expire sooner with your credential or share link. That absolute deadline is retained across database work and signing; delayed requests cannot extend it. Request a new capability with the same file ID if it expires or the first response is lost. Renewal keeps the asset ID, storage key and original uploader audit when MIME type, byte size, optional SHA256 and optional dimensions match exactly. Changing any of those values returns HTTP 409 `ASSET_CONFLICT` without replacing the stored metadata or returning an upload URL; use a new file ID for a different image. Do not store signed URLs in scene content; persist file references.

Public image reads use `POST /api/v1/public/scenes/{sceneId}/assets/presign-view` with the scene-session or room bearer token and `{ "fileIds": ["local-image-id"] }`. The response `urls` contains only assets belonging to that exact workspace and scene. Revocation stops new capabilities immediately through current authorization checks; previously issued storage URLs remain usable until their short expiry.

Mint a room token before requesting `GET /api/v1/realtime/config`. Send that room token (or a scene-session token) as bearer authorization. Current scene access is required; a login token alone is insufficient. The response contains `iceServers` and `expiresAt`; TURN credentials expire with the scoped credential or share link. The shared TURN secret is never returned.

Public guest uploads retain a guest audit identity and do not create or impersonate a user account. Errors use code/message/requestId JSON: invalid or expired credentials return 401, denied access 403/404, invalid input 400, and authority/storage failure 503. Refresh or re-exchange scoped credentials before retrying; a denied request must not fall back to an unscoped route.
