# Reuse workspace library items

Open **Libraries** from a workspace or from the floating canvas panels. Search
the workspace catalog, preview an item, then insert it into a scene you can edit.
Inserted shapes and images receive new IDs, so changing the scene never changes
the published library item.

Workspace owners can import `.excalidrawlib` version 2 files and publish new
immutable revisions. Imports support up to 500 items and 25 MiB. Unsupported
elements, missing image data, unsafe links, and invalid bindings are rejected
before anything is stored. Personal Excalidraw library changes are not published
automatically.

Export downloads the selected immutable revision. If your membership or scene
edit access is revoked, browse/insert/export stops immediately. A failed or
uncertain insert can be retried safely with the same request ID.
