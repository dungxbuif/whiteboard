import { FolderHeart, LockKeyhole, Plus } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { CollectionDialog } from './collection-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { collectionClient, collectionPath, type Collection, type CollectionClient } from '@/lib/collection-api';
import { navigate } from '@/lib/router';

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
  const [filter, setFilter] = useState('');
  const [nextCursor, setNextCursor] = useState('');
  const load = useCallback(async (cursor = '') => {
    try {
      setError('');
      const page = await api.list(workspaceId, cursor);
      setItems(current => cursor ? [...current, ...page.items] : page.items);
      setNextCursor(page.nextCursor ?? '');
    }
    catch { setError('Collections could not be loaded.'); }
  }, [api, workspaceId]);
  useEffect(() => { void load(''); }, [load]);
  return <section aria-labelledby="collection-sidebar-heading" className="mt-4 grid gap-1 border-t pt-4">
    <div className="flex items-center justify-between px-2">
      <p className="text-xs text-muted-foreground" id="collection-sidebar-heading">Collections</p>
      {canManage ? <Button aria-label="New collection" className="h-7 w-7" onClick={() => setOpen(true)} size="icon" type="button" variant="ghost"><Plus /></Button> : null}
    </div>
    {items.length ? <Input aria-label="Filter collections" className="mx-2 h-8 w-[calc(100%-1rem)]" onChange={event => setFilter(event.target.value)} placeholder="Filter collections" value={filter} /> : null}
    {error ? <p className="px-2 text-xs text-destructive" role="alert">{error}</p> : null}
    {items.filter(item => item.name.toLocaleLowerCase().includes(filter.trim().toLocaleLowerCase())).map(item => {
      const href = collectionPath(workspaceSlug, item.id);
      return <a className="flex min-w-0 items-center gap-2 rounded-md px-2 py-2 text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" href={href} key={item.id} onClick={event => { event.preventDefault(); navigate(href); onNavigate?.(); }}>
        {item.visibility === 'private' ? <LockKeyhole className="h-4 w-4 text-muted-foreground" /> : <FolderHeart className="h-4 w-4 text-primary" />}
        <span className="min-w-0 flex-1 truncate">{item.name}</span><span className="text-xs text-muted-foreground">{item.sceneCount}</span>
      </a>;
    })}
    {!items.length && !error ? <p className="px-2 py-1 text-xs text-muted-foreground">No collections yet.</p> : null}
    {nextCursor && !filter.trim() ? <Button className="justify-start" onClick={() => void load(nextCursor)} size="sm" type="button" variant="ghost">Load more collections</Button> : null}
    <CollectionDialog open={open} title="Create collection" pending={pending} onOpenChange={setOpen} onSubmit={async draft => { setPending(true); try { await api.create(workspaceId, draft); setOpen(false); await load(''); } catch { setError('Collection could not be created.'); } finally { setPending(false); } }} />
  </section>;
}
