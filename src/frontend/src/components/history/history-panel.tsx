import { useCallback, useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { versionClient, type RestoreVersionResult, type SceneVersion, type SceneVersionDiff, type VersionClient, type VersionSceneRef } from '@/lib/version-api';
import { VersionDiff } from './version-diff';
import { RestoreDialog } from './restore-dialog';

export function HistoryPanel({ api = versionClient, canRestore, currentSequence, onRestored, registerRefetch, sceneRef }: { api?: Pick<VersionClient, 'list' | 'diff' | 'restore'> & Partial<Pick<VersionClient, 'setMetadata'>>; canRestore: boolean; currentSequence: string; onRestored: (result: RestoreVersionResult) => void; registerRefetch?: (key: string, refetch: () => Promise<void>) => { ready: Promise<void>; unregister: () => void }; sceneRef: VersionSceneRef }) {
  const [versions, setVersions] = useState<SceneVersion[]>([]), [restoreTarget, setRestoreTarget] = useState<SceneVersion | null>(null), [compared, setCompared] = useState<string | null>(null), [diff, setDiff] = useState<SceneVersionDiff | null>(null), [filter, setFilter] = useState(''), [error, setError] = useState(''), [loading, setLoading] = useState(true);
  const load = useCallback(async () => { setLoading(true); setError(''); try { setVersions((await api.list(sceneRef)).items); } catch { setError('Version history could not be loaded.'); } finally { setLoading(false); } }, [api, sceneRef.workspaceId, sceneRef.sceneId]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => { if (!registerRefetch) return; const registration = registerRefetch('versions', load); return registration.unregister; }, [registerRefetch, load]);
  const compare = async (version: SceneVersion) => { setError(''); try { setDiff(await api.diff(sceneRef, version.id, currentSequence)); setCompared(version.id); } catch { setError('The scene changed or the comparison is unavailable. Reload history and try again.'); } };
  const metadata = async (version: SceneVersion, update: { label?: string; pinned?: boolean }) => { setError(''); try { if (!api.setMetadata) throw new Error('Version metadata unavailable'); const next = await api.setMetadata(sceneRef, version.id, { requestId: crypto.randomUUID(), expectedRevision: version.metadataRevision, ...update }); setVersions(items => items.map(item => item.id === next.id ? next : item)); } catch { setError('Version metadata changed. Reload history and try again.'); } };
  const visible = versions.filter(version => !filter.trim() || (version.label ?? '').toLowerCase().includes(filter.trim().toLowerCase()) || version.kind.toLowerCase().includes(filter.trim().toLowerCase()));
  if (loading) return <p role="status">Loading history…</p>;
  return <div className="grid gap-3" data-testid="history-panel">
    <label className="grid gap-1 text-sm">Filter checkpoints<input aria-label="Filter checkpoints" className="h-9 rounded-md border bg-background px-3" onChange={event => setFilter(event.target.value)} placeholder="Label or type" value={filter} /></label>
    {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
    {!visible.length ? <p className="text-sm text-muted-foreground">No checkpoints yet.</p> : visible.map(version => <article className="grid gap-2 rounded-lg border p-3" key={version.id}>
      <div><strong className="block">{version.label ?? (version.pinned ? 'Pinned checkpoint' : 'Checkpoint')}</strong><span className="text-xs text-muted-foreground">Sequence {version.sequence} · {new Date(version.createdAt).toLocaleString()}</span></div>
      {compared === version.id && diff ? <VersionDiff diff={diff} /> : null}
      {restoreTarget?.id === version.id && canRestore ? <RestoreDialog api={api} currentSequence={currentSequence} onCancel={() => setRestoreTarget(null)} onRestored={onRestored} sceneRef={sceneRef} version={version} /> : <div className="flex flex-wrap gap-2"><Button aria-label="Compare version" onClick={() => void compare(version)} size="sm" type="button" variant="outline">Compare</Button>{canRestore ? <><Button aria-label="Restore version" onClick={() => setRestoreTarget(version)} size="sm" type="button">Restore</Button><Button onClick={() => void metadata(version, { pinned: !version.pinned })} size="sm" type="button" variant="ghost">{version.pinned ? 'Unpin' : 'Pin'}</Button><Button onClick={() => { const label = window.prompt('Checkpoint label', version.label ?? '')?.trim(); if (label) void metadata(version, { label }); }} size="sm" type="button" variant="ghost">Label</Button></> : null}</div>}
    </article>)}
  </div>;
}
