import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import type { RestoreVersionResult, SceneVersion, VersionClient, VersionSceneRef } from '@/lib/version-api';

export function RestoreDialog({ api, currentSequence, onCancel, onRestored, sceneRef, version }: { api: Pick<VersionClient, 'restore'>; currentSequence: string; onCancel: () => void; onRestored: (result: RestoreVersionResult) => void; sceneRef: VersionSceneRef; version: SceneVersion }) {
  const [busy, setBusy] = useState(false), [error, setError] = useState('');
  const requestId = useRef(crypto.randomUUID());
  const [conflictedSequence, setConflictedSequence] = useState<string | null>(null);
  useEffect(() => { if (conflictedSequence !== null && conflictedSequence !== currentSequence) { requestId.current = crypto.randomUUID(); setConflictedSequence(null); setError(''); } }, [currentSequence, conflictedSequence]);
  const restore = async () => {
    setBusy(true); setError('');
    try {
      const result = await api.restore(sceneRef, version.id, { requestId: requestId.current, expectedSequence: currentSequence });
      onRestored(result);
    } catch (cause) {
      const status=(cause as {response?:{status?:number}}).response?.status;
      if(status===409){setConflictedSequence(currentSequence);setError('The scene changed. Waiting for the latest collaboration sequence before this confirmation can be rebased.');}
      else if(status===undefined||status>=500)setError('Restore acknowledgement is uncertain. Retrying will safely reuse the same request.');
      else {requestId.current=crypto.randomUUID();setError('Restore was rejected. Review access and version availability before trying again.');}
      setBusy(false);
    }
  };
  return <section className="grid gap-3 rounded-lg border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-950 dark:bg-amber-950/30 dark:text-amber-100" role="dialog" aria-label="Confirm version restore">
    <strong>Restore this checkpoint?</strong>
    <p>A pinned recovery checkpoint of the current scene is created first. Offline edits can replay after the restore when those devices reconnect.</p>
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    <div className="flex justify-end gap-2"><Button disabled={busy} onClick={onCancel} type="button" variant="ghost">Cancel</Button><Button aria-label="Confirm restore" disabled={busy||conflictedSequence!==null} onClick={() => void restore()} type="button">{busy ? 'Restoring…' : 'Confirm restore'}</Button></div>
  </section>;
}
