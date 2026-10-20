import { ArrowDown, ArrowLeft, ArrowUp, ExternalLink, LockKeyhole, Pencil, Plus, Trash2, X } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { CollectionDialog } from './collection-dialog';
import { Button } from '@/components/ui/button';
import { collectionClient, type Collection, type CollectionClient } from '@/lib/collection-api';
import { navigate } from '@/lib/router';

type AvailableScene = { id: string; publicId?: string; title: string };
const sceneHref = (workspaceSlug: string, scene: { id: string; publicId?: string }) => `/w/${encodeURIComponent(workspaceSlug)}/scenes/${encodeURIComponent(scene.publicId || scene.id)}`;

export function CollectionView({ workspaceId, workspaceSlug, collectionId, canManage, availableScenes, api = collectionClient }: {
  workspaceId: string;
  workspaceSlug: string;
  collectionId: string;
  canManage: boolean;
  availableScenes: AvailableScene[];
  api?: CollectionClient;
}) {
  const [value, setValue] = useState<Collection | null>(null);
  const [selectedScene, setSelectedScene] = useState('');
  const [editing, setEditing] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    try { setError(''); setValue(await api.get(workspaceId, collectionId)); }
    catch { setError('This collection is unavailable or you do not have access to it.'); }
  }, [api, collectionId, workspaceId]);
  useEffect(() => { void load(); }, [load]);
  const scenes = useMemo(() => [...(value?.scenes ?? [])].sort((a, b) => a.position - b.position), [value?.scenes]);
  const candidates = useMemo(() => availableScenes.filter(scene => !scenes.some(item => item.id === scene.id)), [availableScenes, scenes]);
  async function mutate(work: () => Promise<Collection>) {
    setPending(true); setError('');
    try { setValue(await work()); }
    catch (cause) {
      const status = (cause as { response?: { status?: number } }).response?.status;
      setError(status === 409 ? 'Collection changed. Reloaded the latest version; try again.' : 'Collection could not be updated.');
      if (status === 409) await load();
    } finally { setPending(false); }
  }
  if (error && !value) return <p role="alert" className="text-destructive">{error}</p>;
  if (!value) return <p>Loading collection…</p>;
  return <section aria-labelledby="collection-heading" className="grid gap-6">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="grid gap-1"><button className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground" onClick={() => navigate(`/w/${encodeURIComponent(workspaceSlug)}`)} type="button"><ArrowLeft className="h-4 w-4" /> Dashboard</button><h1 className="flex items-center gap-2 text-2xl font-semibold" id="collection-heading">{value.name}{value.visibility === 'private' ? <LockKeyhole aria-label="Private collection" className="h-4 w-4 text-muted-foreground" /> : null}</h1><p className="text-sm text-muted-foreground">{value.sceneCount} {value.sceneCount === 1 ? 'scene' : 'scenes'}</p></div>
      {canManage ? <div className="flex gap-2"><Button aria-label="Edit collection" onClick={() => setEditing(true)} size="sm" type="button" variant="outline"><Pencil /> Edit</Button><Button aria-label="Delete collection" disabled={pending} onClick={async () => { if (!window.confirm(`Delete ${value.name}? Scenes will remain in the workspace.`)) return; setPending(true); try { await api.delete(workspaceId, collectionId, value.revision); navigate(`/w/${encodeURIComponent(workspaceSlug)}`); } catch { setError('Collection could not be deleted.'); setPending(false); } }} size="sm" type="button" variant="destructive"><Trash2 /> Delete</Button></div> : null}
    </div>
    {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
    {canManage ? <div className="flex flex-wrap items-end gap-2 rounded-lg border bg-card p-3"><label className="grid min-w-56 flex-1 gap-1 text-sm">Add scene<select aria-label="Scene to add" className="h-9 rounded-md border border-input bg-background px-3 text-sm" onChange={event => setSelectedScene(event.target.value)} value={selectedScene}><option value="">Choose a scene</option>{candidates.map(scene => <option key={scene.id} value={scene.id}>{scene.title}</option>)}</select></label><Button disabled={!selectedScene || pending} onClick={() => void mutate(async () => { const next = await api.addScene(workspaceId, collectionId, value.revision, selectedScene); setSelectedScene(''); return next; })} type="button"><Plus /> Add</Button></div> : null}
    {scenes.length ? <div className="grid gap-2">{scenes.map((scene, index) => <article className="flex items-center gap-2 rounded-md border bg-card p-3 shadow-sm" key={scene.id}><div className="min-w-0 flex-1"><p className="truncate font-medium">{scene.title}</p><p className="text-xs text-muted-foreground">Updated {new Date(scene.updatedAt).toLocaleDateString()}</p></div><a aria-label={`Open ${scene.title}`} className="inline-flex h-8 items-center gap-1 rounded-md px-2 text-xs font-medium hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" href={sceneHref(workspaceSlug, scene)} onClick={event => { event.preventDefault(); navigate(event.currentTarget.getAttribute('href')!); }}><ExternalLink className="h-4 w-4" /> Open</a>{canManage ? <><Button aria-label={`Move ${scene.title} up`} disabled={pending || index === 0} onClick={() => { const ordered = scenes.map(item => item.id); [ordered[index - 1], ordered[index]] = [ordered[index], ordered[index - 1]]; void mutate(() => api.reorder(workspaceId, collectionId, value.revision, ordered)); }} size="icon" type="button" variant="ghost"><ArrowUp /></Button><Button aria-label={`Move ${scene.title} down`} disabled={pending || index === scenes.length - 1} onClick={() => { const ordered = scenes.map(item => item.id); [ordered[index], ordered[index + 1]] = [ordered[index + 1], ordered[index]]; void mutate(() => api.reorder(workspaceId, collectionId, value.revision, ordered)); }} size="icon" type="button" variant="ghost"><ArrowDown /></Button><Button aria-label={`Remove ${scene.title}`} disabled={pending} onClick={() => void mutate(() => api.removeScene(workspaceId, collectionId, value.revision, scene.id))} size="icon" type="button" variant="ghost"><X /></Button></> : null}</article>)}</div> : <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">No accessible scenes in this collection.</div>}
    <CollectionDialog allowPrivate={value.canSetPrivate} initial={{ name: value.name, visibility: value.visibility }} open={editing} pending={pending} title="Edit collection" onOpenChange={setEditing} onSubmit={async draft => { await mutate(() => api.update(workspaceId, collectionId, value.revision, draft)); setEditing(false); }} />
  </section>;
}
