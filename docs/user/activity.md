# Activity feed

Activity shows authorized workspace changes from comments, sharing, agents and
collaboration without copying canvas or comment content into the feed.

Open **Activity** from the workspace sidebar for the workspace timeline. Inside
a scene, open **Activity** from the canvas controls; the panel floats over the
canvas and closes with Escape. Older entries load with **Load older activity**.

The feed always uses your current access. If a scene is deleted or access is
revoked, its entries disappear from later pages. Security and membership
administration entries are visible only to workspace owners. Guests and agent
keys do not have this member feed.

**Available since** is the oldest entry retained in the authorized scope. The
future retention worker may advance this boundary; it is not a promise that an
event will exist for every action. Activity contains safe event kinds and
display names only. It never displays comment bodies, canvas content, tokens,
media signaling or stored credentials.

Activity links are reloadable at `/w/{workspace}/activity`. Empty and access
error states are explicit; losing access does not expose cached older pages.

