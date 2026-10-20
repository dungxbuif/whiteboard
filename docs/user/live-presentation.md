# Join or host a live presentation

Editors can start a live presentation for a scene and copy a presentation
invite. The invite is shown once and expires within 24 hours. Revoking it stops
new joins without changing anyone's workspace membership or ordinary share
links.

The host controls the live session from a floating panel over the canvas. They
can admit or deny people in the waiting room, remove an admitted audience
member, spotlight a participant, lock the audience camera to the current frame,
and transfer hosting to another signed-in editor who is currently in the scene.
Invite delivery is manual: **Copy invite** writes the link to the clipboard and
the app does not email it.

Guests open the invite, enter a display name and wait for the host. While
waiting, the app shows only waiting status. The scene, participants, images,
voice and screen sharing become available only after the host admits that exact
connection. A pending credential expires after five minutes, so reload the
invite if the waiting page expires.

After admission, the selected browser receives its own short-lived audience
credential and uses it for view-only scene sync and images. It is never shared
with the host or other people in the waiting room. Audience access cannot
upload or change scene assets.

Once admitted, the audience can pan and zoom normally until the host enables
**Lock view**. A locked view follows the host's selected Excalidraw frame by its
stable element ID. If that frame is deleted, the audience sees a removed-frame
message instead of jumping to a different frame. Locking affects only the
camera. It does not change scene permissions, and **Leave** plus the Escape key
remain available.

Audience members can send 👏, ❤️, 👍, 🎉 or 😂. Each browser may send one
reaction per second; reactions disappear after three seconds. Removing a
spotlighted person also clears the spotlight. The current host remains the only
authority when simultaneous host or frame changes occur.

A live session ends when the host leaves or goes offline, an administrator
disables live presentation for that scene, or the collaboration service
restarts. The host can then start a new session and copy a new invite. Up to
100 guests may wait and up to 100 may be admitted at once. Repeated join
attempts are limited to 60 per minute per IP.

If starting does not finish within ten seconds, the app closes and reconnects
the presentation connection before offering a retry. Wait for collaboration to
reconnect, then start again; the previous attempt cannot remain as a hidden
active session.

If an audience credential expires or is revoked, the canvas closes and the app
offers **Rejoin waiting room**. Rejoining repeats host admission and never
restores audience authority from cached UI state. Voice, screen sharing and
drawing keep their existing permission and browser-autoplay behavior; starting
a live presentation does not start media automatically.
If the presentation ends while you are joining, the Joining screen changes to
an ended message and offers the same fresh rejoin action. Choosing **Leave** in
the waiting room removes your name from the host's queue immediately.
