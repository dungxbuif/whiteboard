import { useState } from 'react';
import { Button } from '@/components/ui/button';
import type { RestoreVersionResult, SceneVersion, VersionClient, VersionSceneRef } from '@/lib/version-api';

export function RestoreDialog({ api, currentSequence, onCancel, onRestored, sceneRef, version }: { api: Pick<VersionClient, 'restore'>; currentSequence: string; onCancel: () => void; onRestored: (result: RestoreVersionResult) => void; sceneRef: VersionSceneRef; version: SceneVersion }) {
  const [busy, setBusy] = useState(false), [error, setError] = useState('');
  const restore = async () => {
    setBusy(true); setError('');
    try {
      const result = await api.restore(sceneRef, version.id, { requestId: crypto.randomUUID(), expectedSequence: currentSequence });
      onRestored(result);
    } catch (cause) {
      setError((cause as { response?: { status?: number } }).response?.status === 409 ? 'The scene changed. Reload history and review the latest changes before restoring.' : 'Restore failed. Your current scene and recovery data are unchanged.');
      setBusy(false);
    }
  };
  return <section className="grid gap-3 rounded-lg border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-950 dark:bg-amber-950/30 dark:text-amber-100" role="dialog" aria-label="Confirm version restore">
    <strong>Restore this checkpoint?</strong>
    <p>A pinned recovery checkpoint of the current scene is created first. Offline edits can replay after the restore when those devices reconnect.</p>
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    <div className="flex justify-end gap-2"><Button disabled={busy} onClick={onCancel} type="button" variant="ghost">Cancel</Button><Button aria-label="Confirm restore" disabled={busy} onClick={() => void restore()} type="button">{busy ? 'Restoring…' : 'Confirm restore'}</Button></div>
  </section>;
}
