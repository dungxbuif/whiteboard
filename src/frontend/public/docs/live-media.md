# Screen sharing

Screen sharing uses the existing authenticated collaboration socket for
admission and WebRTC signaling. It is independent of microphone voice.

Signed-in editors first send `screen_request`. A successful
`screen_admission` creates a 30-second reservation and exposes a separate
**Start sharing** action. That click invokes `getDisplayMedia` with
`audio:false` and maximum video constraints of 1280×720 at 15 fps. Successful
capture confirms the server-issued session and clears the reservation.

`screen_signal` accepts only current same-room admitted connection and session
IDs. `screen_stop`, authority loss, disconnect, picker cancellation, late
capture completion, or track end closes peers and stops tracks. Capacity is ten
distinct media connections across voice and screen, counted once per
connection, with one screen publisher per room.

The workspace `screenShareEnabled` policy is disabled by default. Go evaluates
the current `screen:publish` decision through the private authority before
reservation, activation and every publisher control. Disabling the feature ends
the active share before further signaling; drawing remains available.
