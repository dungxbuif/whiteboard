import { FolderHeart, LockKeyhole, Plus } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { CollectionDialog } from './collection-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { collectionClient, collectionPath, type Collection, type CollectionClient } from '@/lib/collection-api';
import { navigate } from '@/lib/router';

const featureDisabled = (cause: unknown) => {
  const response = (cause as { response?: { status?: number; data?: { code?: string } } } | null)?.response;
  return response?.status === 403 && response.data?.code === 'FEATURE_DISABLED';
};

export function CollectionList({ workspaceId, workspaceSlug, canManage, api = collectionClient, onNavigate }: {
  workspaceId: string;
  workspaceSlug: string;
  canManage: boolean;
  api?: CollectionClient;
  onNavigate?: () => void;
}) {
  const [items, setItems] = useState<Collection[]>([]);
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [disabled, setDisabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState('');
  const [nextCursor, setNextCursor] = useState('');
  const requestVersion = useRef(0);
  const workspaceVersion = useRef(0);
  const load = useCallback(async (cursor = '') => {
    const version = ++requestVersion.current;
    setLoading(true);
    setError('');
    setDisabled(false);
    if (!cursor) {
      setItems([]);
      setNextCursor('');
    }
    try {
      const page = await api.list(workspaceId, cursor);
      if (version !== requestVersion.current) return;
      setItems(current => cursor ? [...current, ...page.items] : page.items);
      setNextCursor(page.nextCursor ?? '');
    }
    catch (cause) {
      if (version !== requestVersion.current) return;
      if (featureDisabled(cause)) {
        setDisabled(true);
        setItems([]);
        setNextCursor('');
      } else setError('Collections could not be loaded.');
    } finally { if (version === requestVersion.current) setLoading(false); }
  }, [api, workspaceId]);
  useEffect(() => { void load(''); return () => { requestVersion.current += 1; }; }, [load]);
  useEffect(() => {
    workspaceVersion.current += 1;
    setOpen(false);
    setPending(false);
    return () => { workspaceVersion.current += 1; };
  }, [workspaceId]);
  const create = async (draft: Parameters<CollectionClient['create']>[1]) => {
    const version = workspaceVersion.current;
    setPending(true);
    try {
      await api.create(workspaceId, draft);
      if (version !== workspaceVersion.current) return;
      setOpen(false);
      await load('');
    } catch (cause) {
      if (version !== workspaceVersion.current) return;
      if (featureDisabled(cause)) {
        setError('');
        setDisabled(true);
        setOpen(false);
        setItems([]);
        setNextCursor('');
      } else setError('Collection could not be created.');
    } finally { if (version === workspaceVersion.current) setPending(false); }
  };
  return <section aria-labelledby="collection-sidebar-heading" className="mt-4 grid gap-1 border-t pt-4">
    <div className="flex items-center justify-between px-2">
      <p className="text-xs text-muted-foreground" id="collection-sidebar-heading">Collections</p>
      {canManage && !disabled ? <Button aria-label="New collection" className="h-7 w-7" onClick={() => setOpen(true)} size="icon" type="button" variant="ghost"><Plus /></Button> : null}
    </div>
    {items.length ? <Input aria-label="Filter collections" className="mx-2 h-8 w-[calc(100%-1rem)]" onChange={event => setFilter(event.target.value)} placeholder="Filter collections" value={filter} /> : null}
    {error ? <p className="px-2 text-xs text-destructive" role="alert">{error}</p> : null}
    {disabled ? <p className="px-2 py-1 text-xs text-muted-foreground">Collections are unavailable for this workspace</p> : null}
    {error ? <Button className="justify-start" disabled={loading} onClick={() => void load('')} size="sm" type="button" variant="ghost">Retry collections</Button> : null}
    {items.filter(item => item.name.toLocaleLowerCase().includes(filter.trim().toLocaleLowerCase())).map(item => {
      const href = collectionPath(workspaceSlug, item.id);
      return <a className="flex min-w-0 items-center gap-2 rounded-md px-2 py-2 text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" href={href} key={item.id} onClick={event => { event.preventDefault(); navigate(href); onNavigate?.(); }}>
        {item.visibility === 'private' ? <LockKeyhole className="h-4 w-4 text-muted-foreground" /> : <FolderHeart className="h-4 w-4 text-primary" />}
        <span className="min-w-0 flex-1 truncate">{item.name}</span><span className="text-xs text-muted-foreground">{item.sceneCount}</span>
      </a>;
    })}
    {!items.length && !error && !disabled && !loading ? <p className="px-2 py-1 text-xs text-muted-foreground">No collections yet.</p> : null}
    {nextCursor && !filter.trim() ? <Button className="justify-start" onClick={() => void load(nextCursor)} size="sm" type="button" variant="ghost">Load more collections</Button> : null}
    <CollectionDialog open={open} title="Create collection" pending={pending} onOpenChange={value => { setOpen(value); if (!value) api.cancelPending(); }} onSubmit={create} />
  </section>;
}
