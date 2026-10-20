# Comments and notifications

Open **Comments** from the floating canvas controls to review or start a
discussion. Comments stay outside the drawing document, so they never create
or change Excalidraw shapes.

## Discuss a scene

1. Open **Comments**.
2. Choose an existing thread or select **New**.
3. Write 1–4,000 Unicode characters and submit.

An anchor attached to a native element follows that element when it moves or
resizes. Its marker also follows canvas pan and zoom, including the canvas
offset within the page. If the element is deleted, the marker stays at its saved fallback
position and says **Element unavailable**. Restoring an element does not attach
an old discussion to a different element automatically.

View-link and edit-link guests may comment only when the link owner enabled
comments. Guest permission never grants drawing access. Guests can reply and
see moderation results, but they do not receive a private workspace inbox.
View links that allow comments assign the fallback display name **Guest** when
the visitor does not enter one, so an anonymous read-only visitor can still
start or reply to a discussion.

Authors can redact their own messages. Workspace owners can redact any message;
members with edit access to the current scene can resolve or reopen threads.
A workspace editor whose scene override is view-only cannot moderate that scene.
Redacted text is replaced by
a visible removal marker. When another participant changes a thread while you
are writing, the panel reports the conflict and keeps your draft so you can
reload and submit again.

## Use the inbox

Signed-in members are subscribed when they create or reply to a thread. The
dashboard **Notifications** control lists comment changes from other people.
Choosing an item marks it read and opens the exact scene and thread. The server
checks your current scene access both when it creates an inbox item and when you
read or open it; revoked scenes disappear from results.

Choose **Load more** to continue from the server cursor. The inbox may skip
revoked items while it searches for the next accessible result, so a page can
span more stored rows than the number shown.

Notifications are in-app only. Delivery retries are safe and create at most one
row per user and source event.
