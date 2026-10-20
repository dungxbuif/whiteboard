import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import type { CollectionDraft, CollectionVisibility } from '@/lib/collection-api';

export function CollectionDialog({ open, title, initial, pending = false, allowPrivate = true, onOpenChange, onSubmit }: {
  open: boolean;
  title: string;
  initial?: CollectionDraft;
  pending?: boolean;
  allowPrivate?: boolean;
  onOpenChange(value: boolean): void;
  onSubmit(value: CollectionDraft): Promise<void> | void;
}) {
  const [name, setName] = useState(initial?.name ?? '');
  const [visibility, setVisibility] = useState<CollectionVisibility>(initial?.visibility ?? 'workspace');
  useEffect(() => {
    if (open) { setName(initial?.name ?? ''); setVisibility(initial?.visibility ?? 'workspace'); }
  }, [open, initial?.name, initial?.visibility]);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || pending) return;
    await onSubmit({ name: name.trim(), visibility });
  }
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent>
      <form aria-label={title} className="grid gap-4" onSubmit={submit}>
        <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>Collections organize scene links and do not change scene access.</DialogDescription></DialogHeader>
        <label className="grid gap-1 text-sm">Name<Input autoFocus aria-label="Collection name" value={name} onChange={event => setName(event.target.value)} /></label>
        <label className="grid gap-1 text-sm">Visibility<select aria-label="Collection visibility" className="h-9 rounded-md border border-input bg-background px-3 text-sm" value={visibility} onChange={event => setVisibility(event.target.value as CollectionVisibility)}><option value="workspace">Workspace</option>{allowPrivate ? <option value="private">Private</option> : null}</select></label>
        <DialogFooter><Button disabled={pending || !name.trim()} type="submit">{pending ? 'Saving…' : 'Save'}</Button></DialogFooter>
      </form>
    </DialogContent>
  </Dialog>;
}
