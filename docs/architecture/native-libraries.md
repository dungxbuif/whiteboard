# Native workspace libraries

Workspace libraries store immutable Excalidraw library v2 revisions as metadata
owned by Go. Only workspace owners publish/import revisions; current workspace
members may browse and editors may insert into a scene. Every read and insert
rechecks current workspace and target-scene authority.

Imports are limited to 500 items and 25 MiB. The diagram-kit validator accepts
only version 2 native bundles, validates every native element/file, rejects
credential-bearing links, and requires referenced raster files to be present.
Insert remaps every element, group and file ID plus frame/container/binding and
presentation references. The original scene/library IDs never enter the target.

`library_revisions` are immutable and `library_items.current_revision` advances
only when the matching revision exists. `library_assets` pins
source assets in the same database transaction as a revision. Insert requests
bind workspace, item, revision, target scene, expected sequence, principal and
canonical body to a request ID. Go mints a `library:insert` mutation decision;
the collaboration writer performs the one canonical document write and returns
its durable receipt. `library_asset_stages` derives the target file and asset IDs
from the request ID, copies each raster object before the document write, and
records `pending`/`ready` state. Exact retries reuse the same object and metadata.
Every stage carries `cleanup_after`; only the R2 grace-period collector may
remove an abandoned copy after proving that no canonical document references it.

The browser exposes workspace items through Excalidraw's native library API as
an authorized view. Local personal-library changes remain local unless an owner
explicitly publishes them.
