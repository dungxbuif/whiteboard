import type { SceneVersionDiff } from '@/lib/version-api';

export function VersionDiff({ diff }: { diff: SceneVersionDiff }) {
  return <div className="grid grid-cols-2 gap-2 rounded-md border bg-muted/40 p-3 text-xs" aria-label="Version changes">
    <span>{diff.addedIds.length} added</span><span>{diff.removedIds.length} removed</span>
    <span>{diff.changedIds.length} changed</span><span>{diff.changedAppStateFields.length} canvas settings</span>
  </div>;
}
