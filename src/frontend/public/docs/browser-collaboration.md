# Browser collaboration and recovery

Status: implemented and deployed on VM100. Authenticated public deep-link save/reload passes; final restart/roles/denial canaries remain before sign-off.

Open a scene and wait for **Saved** or **Read-only**. The canvas stays read-only until the server checks current access and the document finishes synchronizing. A pending count means changes are on this device and still await the server's durable acknowledgement. Closing the tab does not mean those changes were saved on the server.

If the connection drops after admission, an editor can continue on the loaded scene. Changes stay in IndexedDB and replay after fresh admission and synchronization. Reloading without network access is not an installed/offline app feature: the page must be available, and a fresh page remains read-only until admission. When access is revoked or reduced to viewing, pending work stays available for recovery but is not transmitted as an authorized edit.

Use **Export recovery** before clearing browser data, changing accounts after a failed save, or recovering a history-limit error. The JSON contains the local Yjs checkpoint, pending update bytes with original request identities, retained image bytes and unfinished canvas drafts. Keep it private. Recovery exports are diagnostic/operational files; automatic import into another identity/document is not offered. A changed document generation stops replay and asks for recovery instead of deleting old edits.

The queue belongs to one principal, workspace, scene and document generation. Another signed-in account does not inherit it. A guest session is stored only in that tab's sessionStorage, keyed to its share link, so reloading the tab preserves the guest identity while its credential remains valid. It never replaces the account login token. Expired credentials are stripped when restored; identity metadata remains so the old queue can be exported. A new link exchange can create a new guest identity and does not reassign the old queue.

**Undo my edit** and **Redo my edit** use local Yjs transaction history and preserve other actors' changes. Keyboard undo on the canvas uses the same history; text-field editing keeps the text editor's own keyboard behavior. AI additions are atomic separate operations and are not included in local human undo. AI additions and presentation frame reordering use the collaborative content writer.

Images are retained locally before upload. The browser sends the actual Blob byte size, uses the upload service's Content-Type and commits an asset reference only after a successful PUT. Failed/expired uploads retain their bytes; **Retry connection / uploads** requests fresh capabilities. Original inline image data is not committed to collaborative documents. Pending image edits can delay the associated canvas draft until upload succeeds; keep recovery exports if storage or authorization fails.

Legacy HTTP autosave is selected only when the server explicitly reports COLLAB_DISABLED or PROMOTION_REQUIRED and a fresh scene read confirms writerMode=v1. Network failures never select it, and a scene already admitted to v2 never falls back. A racing COLLAB_REQUIRED response fences the legacy writer, retains the draft for recovery and reconnects to collaboration. Public legacy scenes remain read-only.

Implementation contracts: CollabProvider({sceneRef,principal,children}), applyLocalScene(SceneContent), applyAgentElements(NativeElement[]), enqueuePending(scope,requestId,update), ackPending(scope,requestId). Scope includes principalId/workspaceId/sceneId/documentId. Native element fields use nested Y.Maps; ordering uses shared ID arrays; data URLs, camera, selection and credentials are excluded. Channel 3 carries durable Yjs updates. Only a matching acknowledgement on the current admitted connection clears a pending request. Unsent updates may be merged atomically before first transmission; a sent request ID always retains the same bytes.

Transport hooks for presence/voice integration: subscribeControl(listener), sendControl(type,payload), sendAwareness({cursor?,selectedIds?}), subscribe/getSnapshot. Awareness strips native pointer metadata and sends at most one sample every 50 ms. The custom `/collab/` WebSocket is the only realtime path; the legacy Socket.IO client, server and proxy route are removed.

### People, follow and voice

Open **People** to inspect connected tabs and their Editor/Viewer role. Choose **Follow** to track a participant's camera and **Stop following** to exit; departure clears followers automatically. Presence, cursors, selection and camera never change the saved board.

Choose **Join voice** to request one of ten room slots. The app asks for microphone permission only after admission. A denial or full room does not disable drawing. Use **Mute**, **Unmute** and **Leave voice** in the toolbar. If the browser blocks autoplay, choose **Play voice audio**. Connection loss, access revocation and leaving the call stop local tracks and remove peer/audio objects. Media uses short-lived scoped TURN credentials; secrets are never embedded in the frontend.

Definitive authority rejection clears the retained guest token immediately; transient network failures preserve it. The guest principal and pending recovery remain. **Export recovery and rejoin** downloads that recovery before starting a fresh exchange, which may create a different guest identity.

Legacy image uploads use the same capability and PUT flow. Their blobs remain until the HTTP content save is confirmed. The local `legacy-v1` recovery namespace is never replayed into a server Yjs document after promotion.

Recovery JSON format is `devtool-collab-recovery-v1`, with `encoding=base64`. `currentBase64` stores the current Yjs update. Each scoped entry can contain `snapshotBase64`, `pending[].updateBase64` with original request metadata, a canvas `draft`, and `blobs[].blobBase64` with file metadata and actual byte size. Decode these fields as bytes, not UTF-8 text. Credentials are excluded.

If you leave the canvas while an image upload is pending, its late completion cannot write to the closed canvas or clear its recovery. The original image and draft remain scoped to that scene and identity; reopening under fresh authority retries them. This also applies to the explicit legacy fallback. Overlapping autosend and upload-completion flushes share one in-flight send and acknowledgement deadline.

Image recovery also preserves the newest canvas draft when image conversion finishes out of order. Draft revisions are stored with the scene's recovery scope, so an older conversion cannot replace newer edits made before leaving or after reopening that same scene. Late image bytes stay in their original identity/scene scope. Confirmed cleanup keeps a small revision marker so completed recovery is not recreated by an old callback.

If two tabs for the same identity and scene have different pending image drafts, a tab whose draft has been superseded stops its upload attempt and shows a recovery message. It keeps the latest scoped recovery and does not repeatedly upload or apply the older draft. Export recovery and reopen the scene to load that recovery. Retrying an unchanged older draft cannot bypass this check; a new local canvas edit reserves a new revision and can proceed normally.

### Interrupted image preparation

Moving an image while its upload or save is pending preserves the latest position and independent remote edits. If the canvas closes or local storage fails before image preparation finishes, automatic image replay pauses. The browser keeps the original draft and image bytes and shows a recovery message after reopening. Choose **Export recovery** and keep the download before clearing browser data; reopening or pressing Retry does not resolve this incomplete preparation automatically. Normally completed preparation resumes automatically after reload. Recovery exports include `blobs[].draft.pendingBaseline=true` for this interrupted case, together with the retained draft and encoded image bytes.
