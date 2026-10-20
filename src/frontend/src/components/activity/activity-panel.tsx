import { useCallback, useEffect, useState } from 'react';

import { Button } from '@/components/ui/button';
import { activityClient, type ActivityClient, type ActivityItem } from '@/lib/activity-api';

type RefetchRegistration = { ready: Promise<void>; unregister: () => void };

function occurredLabel(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'Unknown time' : date.toLocaleString();
}

export function ActivityPanel({ workspaceId, sceneId, api = activityClient, registerRefetch }: { workspaceId: string; sceneId?: string; api?: ActivityClient; registerRefetch?: (key: string, refetch: () => Promise<void>) => RefetchRegistration }) {
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [retainedSince, setRetainedSince] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const clearForError = useCallback((reason: unknown) => {
    setItems([]); setNextCursor(''); setRetainedSince('');
    setError(reason instanceof Error ? reason.message : 'Activity unavailable');
  }, []);

  const refresh = useCallback(async () => {
    setError('');
    try {
      const page = await api.list(workspaceId, sceneId);
      setItems(page.items); setNextCursor(page.nextCursor ?? ''); setRetainedSince(page.retainedSince);
    } catch (reason) {
      clearForError(reason);
      throw reason;
    } finally { setLoading(false); }
  }, [api, clearForError, workspaceId, sceneId]);

  useEffect(() => { setLoading(true); void refresh().catch(() => undefined); }, [refresh]);
  useEffect(() => {
    if (!registerRefetch) return;
    const registration = registerRefetch(`activity:${workspaceId}:${sceneId ?? 'workspace'}`, refresh);
    void registration.ready.catch(clearForError);
    return registration.unregister;
  }, [clearForError, registerRefetch, refresh, workspaceId, sceneId]);

  const older = async () => {
    if (!nextCursor) return;
    setError('');
    try {
      const page = await api.list(workspaceId, sceneId, nextCursor);
      setItems(current => [...current, ...page.items]); setNextCursor(page.nextCursor ?? ''); setRetainedSince(page.retainedSince);
    } catch (reason) { clearForError(reason); }
  };

  if (loading) return <p className="p-2 text-sm text-muted-foreground">Loading activity...</p>;
  return <div className="grid gap-3" data-testid="activity-panel">
    {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
    {!items.length && !error ? <p className="text-sm text-muted-foreground">No activity yet.</p> : null}
    <ol className="grid gap-2">
      {items.map(item => <li className="rounded-lg border bg-background/80 p-3" key={item.id}>
        <div className="flex flex-wrap items-baseline justify-between gap-2"><strong className="break-words text-sm">{item.actorName}</strong><time className="text-xs text-muted-foreground" dateTime={item.occurredAt}>{occurredLabel(item.occurredAt)}</time></div>
        <p className="break-words text-sm">{item.kind}</p>
      </li>)}
    </ol>
    {nextCursor ? <Button onClick={() => void older()} type="button" variant="outline">Load older activity</Button> : null}
    {retainedSince ? <p className="text-xs text-muted-foreground">Available since {occurredLabel(retainedSince)}</p> : null}
  </div>;
}

