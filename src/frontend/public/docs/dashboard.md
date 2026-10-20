# Dashboard

The prior dashboard baseline is deployed on VM100. Its public desktop, mobile,
keyboard, deep-link and access-role checks pass.

The management controls below are implemented and verified locally; their
production rollout remains pending.

The dashboard follows the original devtool interface, including its original colors, sidebar, scene cards, typography and spacing. Excalidraw keeps its native canvas appearance.

Use the workspace switcher to choose where scenes and folders are saved. Use search, All files, Starred and Trash to find content. The sidebar can collapse on desktop; on mobile it opens as a keyboard-accessible drawer.

## Manage workspaces, folders, and scenes

Workspace owners can choose **Rename workspace** in the workspace switcher.
Enter a name of 3–100 characters; existing workspace links keep working. Owners
and editors can choose **Rename or move** in a folder's Actions menu, change its
name (1–100 characters), and select a parent folder or the root. A folder cannot
move into itself, a descendant, or a location beyond the nesting depth limit.
Its subfolders move with it.

Owners and editors with scene edit access can use a scene's menu to **Move** it
to a folder or **Duplicate** it with a new name and destination. The copy opens
as a new scene with the latest durable drawing, including acknowledged
collaboration edits and uploaded images. Large scenes can be copied; a failed
copy does not appear as a partial scene in the dashboard or Trash. Edit access
is checked again before the copy becomes visible. If the selected destination
folder is moved or trashed while a copy is in progress, the copy fails; choose
its current location and try again. Legacy copies keep durable canvas settings
and supported scene-owned images; inline-only legacy images, broken image
references, or malformed saved fields need repair before copying. The copy
excludes collaborator and session state. Viewers do not
see these edit controls. If the follow-up list reload fails, the current folder
still shows only the scenes that belong there; a later reload refreshes the list.

When enabled by workspace policy, Collections appears beside folder navigation.
Workspace and creator-private containers use canonical reloadable routes and
organize links without moving scenes or changing scene access. See
[Collections](collections.md).
When workspace policy disables Collections, the sidebar shows **Collections are
unavailable for this workspace** and hides creation. A temporary failure keeps
a **Retry** action to reload the list.

Owners publish reusable native diagram bundles in [Libraries](libraries.md);
members browse them and scene editors insert fresh copies.

New scene opens an empty canvas. Import scenes accepts an Excalidraw JSON file up to 25 MiB. Invalid files show an error without creating a scene. Imported content is submitted in the scene creation request, preserving native editable elements.

Theme and navigation preferences are local to this browser. The account menu exposes theme switching and sign out. Workspace owners also have Members and Agent keys settings links.

## Import a drawing

1. Choose the workspace and folder.
2. Select **Import scenes**, then choose a version 2 `.excalidraw` JSON file.
3. The drawing opens after the server creates the scene. Use the normal canvas tools to edit it.

The maximum file size is 25 MiB. Embedded PNG, JPEG, GIF and WebP images are accepted; SVG images and remote image URLs are currently rejected. Invalid geometry, malformed bindings, duplicate IDs and invalid legacy font fields show an error before any scene is created. Selection, collaborators and other session-only state are not imported.

An import uses the existing scene-create API and the current signed-in workspace. Browser checks help detect malformed files; server authorization remains responsible for access control.

## Appearance and navigation

Use the account menu at the bottom of the sidebar to switch light/dark theme or sign out. The original style includes colored focus and status accents. Theme choices apply to dashboard dialogs and menus, while the canvas uses Excalidraw's native styling.

On desktop the header button hides or shows the sidebar. On mobile, **Open navigation** opens a drawer; Escape closes it and returns focus to the trigger. Long workspace names are shortened in the sidebar and shown in the page heading.

## Access and settings

Owners can open **Members** and **Agent keys** from the sidebar. Settings stay inside the dashboard shell. Use **Dashboard** to return to your scenes. Members supports invitation links, member role changes, removal and explicit scene overrides. Agent keys supports limited scopes, selected scenes and one-time key disclosure.

Editors see content creation controls. Viewers see reading controls; owner management actions remain hidden. These controls use server-computed permissions and every request is still authorized by the server.

Scene links accept existing UUIDs and public aliases. Reload, back and forward restore the selected workspace and scene. Returning to the dashboard closes the scene, and late scene-open responses cannot replace a newer selection. Search results open a scene route too.

Management and navigation have browser regression coverage and deployed public
surface checks. Public guest mutation/revocation and the roles matrix pass
production canaries. See [collaboration](collaboration.md) for invitation and
sharing steps.

## Move between settings and your files

Choosing a workspace opens that workspace's dashboard at `/w/:id`. You can reload that route, or use browser back/forward to return to the previous workspace settings. UUIDs and public aliases both work.

From Members or Agent keys, **All files**, a folder, **Starred**, owner-only **Trash**, and the search box open their destination in the workspace dashboard. The selected folder, view or search text is retained. **Dashboard** opens All files and clears the previous search.
