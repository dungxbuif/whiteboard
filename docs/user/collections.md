# Organize scenes with collections

Collections let you gather related scene links without changing folders or scene
permissions. Open **Collections** in the workspace navigation. Editors and owners
can choose **New collection**, name it, and select workspace or private
visibility.

- A workspace collection is visible to current workspace members.
- A private collection is visible only to the person who created it.
- Only the creator of a workspace collection can later make it private. Other
  editors can keep managing the shared collection but are not offered that
  visibility change.
- A collection shows only scenes you can currently access.
- Adding, removing, or reordering a scene changes only the collection. The scene
  stays in its folder and keeps its existing sharing rules.

Open a collection at `/w/{workspace}/collections/{collection}`. This URL can be
bookmarked and reloaded. Use **Add scene** to include an accessible scene, arrow
buttons to adjust the stable order, and **Remove** to remove only the collection
membership. Deleting a collection leaves every scene intact.

If a scene is trashed or your access is removed, it disappears from the visible
collection and its count. Reordering the remaining visible scenes safely removes
those stale hidden links. Removing a single scene link always checks that you can
still access that scene first.

If another edit changed the collection first, the app reloads the current
revision and asks you to try the action again. Viewers can open workspace
collections and their authorized scene links but do not see management actions.
