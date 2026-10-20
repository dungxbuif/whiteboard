import { useCallback, useEffect, useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { createCommentClient, type CommentClient, type CommentPrincipal, type CommentThread, type SceneRef } from '@/lib/comment-api';

type RefetchRegistration = { ready: Promise<void>; unregister: () => void };

export function CommentPanel({ sceneRef, principal, api, anchor = { x: 0, y: 0 }, registerRefetch, onThreadsChange }: { sceneRef: SceneRef; principal: CommentPrincipal; api?: CommentClient; anchor?: { elementId?: string; x: number; y: number }; registerRefetch?: (key: string, refetch: () => Promise<void>) => RefetchRegistration; onThreadsChange?: (threads: CommentThread[]) => void }) {
  const client = useMemo(() => api ?? createCommentClient(principal), [api, principal.kind, principal.id, principal.token]);
  const [threads, setThreads] = useState<CommentThread[]>([]);
  const [selected, setSelected] = useState('');
  const [draft, setDraft] = useState('');
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const active = threads.find(thread => thread.id === selected) ?? threads[0];
  const refresh = useCallback(async () => {
    const page = await client.list(sceneRef);
    setThreads(page.items);
    setSelected(current => current || page.items[0]?.id || '');
  }, [client, sceneRef.workspaceId, sceneRef.sceneId]);
  useEffect(() => { void refresh().catch(reason => setError(reason instanceof Error ? reason.message : 'Comments unavailable')); }, [client, sceneRef.workspaceId, sceneRef.sceneId]);
  useEffect(() => { onThreadsChange?.(threads); }, [onThreadsChange, threads]);
  useEffect(() => {
    if (!registerRefetch) return;
    const registration = registerRefetch(`comments:${sceneRef.workspaceId}:${sceneRef.sceneId}`, refresh);
    void registration.ready.catch(reason => setError(reason instanceof Error ? reason.message : 'Comments unavailable'));
    return registration.unregister;
  }, [registerRefetch, refresh, sceneRef.workspaceId, sceneRef.sceneId]);
  const replace = (thread: CommentThread) => setThreads(current => current.map(item => item.id === thread.id ? thread : item));
  const submit = async () => {
    if (!active || !draft.trim()) return;
    setError('');
    try {
      await client.reply(sceneRef, active.id, draft);
      setDraft('');
      replace(await client.get(sceneRef, active.id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Reply failed');
    }
  };
  const redact = async (messageId: string, revision: string) => {
    if (!active) return;
    await client.deleteMessage(sceneRef, active.id, messageId, revision);
    replace(await client.get(sceneRef, active.id));
  };
  const create = async () => {
    if (!draft.trim()) return;
    setError('');
    try {
      const thread = await client.create(sceneRef, anchor, draft);
      setThreads(current => [thread, ...current]); setSelected(thread.id); setDraft(''); setCreating(false);
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Discussion could not be created'); }
  };
  if (!active) return <div className="grid gap-3"><p className="text-sm text-muted-foreground">No discussions yet.</p><label className="grid gap-1 text-sm">New discussion<textarea aria-label="New discussion" className="min-h-24 rounded-md border bg-background p-2" maxLength={4000} onChange={event => setDraft(event.target.value)} value={draft} /></label>{error && <p role="alert">{error}</p>}<Button disabled={!draft.trim()} onClick={() => void create()}>Create discussion</Button></div>;
  return <div className="grid gap-4" data-testid="comment-panel">
    <div className="flex flex-wrap gap-2">{threads.map((thread, index) => <Button key={thread.id} onClick={() => { setCreating(false); setSelected(thread.id); }} size="sm" variant={!creating && thread.id === active.id ? 'default' : 'outline'}>Thread {index + 1}</Button>)}<Button onClick={() => setCreating(true)} size="sm" variant={creating ? 'default' : 'outline'}>New</Button><Button onClick={() => void refresh()} size="sm" variant="ghost">Refresh</Button></div>
    {creating ? <><label className="grid gap-1 text-sm">New discussion<textarea aria-label="New discussion" className="min-h-24 rounded-md border bg-background p-2" maxLength={4000} onChange={event => setDraft(event.target.value)} value={draft} /></label><Button disabled={!draft.trim()} onClick={() => void create()}>Create discussion</Button></> : <>
    <div className="grid gap-3">
      <div className="flex items-center justify-between gap-2"><span className="text-sm text-muted-foreground">{active.anchor.elementId ? 'Anchored discussion' : 'Canvas discussion'}</span><Button onClick={async () => replace(await client.updateThread(sceneRef, active, { resolved: active.status !== 'resolved' }))} size="sm" variant="outline">{active.status === 'resolved' ? 'Reopen' : 'Resolve'}</Button></div>
      {active.messages.map(message => <article className="rounded-lg border p-3" key={message.id}>
        <strong className="text-sm">{message.authorName}</strong>
        {message.deletedAt ? <p className="text-sm italic text-muted-foreground">Message removed</p> : <p className="whitespace-pre-wrap break-words text-sm">{message.body}</p>}
        {!message.deletedAt && (message.mine || principal.owner) ? <Button aria-label="Redact message" onClick={() => void redact(message.id, message.revision)} size="sm" variant="ghost">Redact</Button> : null}
      </article>)}
    </div>
    <label className="grid gap-1 text-sm">Reply<textarea aria-label="Reply" className="min-h-24 rounded-md border bg-background p-2" maxLength={4000} onChange={event => setDraft(event.target.value)} value={draft} /></label>
    {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
    <Button disabled={!draft.trim()} onClick={() => void submit()}>Reply</Button>
    </>}
  </div>;
}
