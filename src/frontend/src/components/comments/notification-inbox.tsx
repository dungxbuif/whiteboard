import { useEffect, useState } from 'react';
import { Bell } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { listNotifications, markNotificationRead, type Notification } from '@/lib/comment-api';
import { navigate } from '@/lib/router';

export function NotificationInbox() {
  const [items, setItems] = useState<Notification[]>([]);
  const [open, setOpen] = useState(false);
  const [nextCursor, setNextCursor] = useState('');
  useEffect(() => { void listNotifications().then(page => { setItems(page.items); setNextCursor(page.nextCursor ?? ''); }).catch(() => undefined); }, []);
  const loadMore = async () => { const page = await listNotifications(nextCursor); setItems(current => [...current, ...page.items.filter(item => !current.some(existing => existing.id === item.id))]); setNextCursor(page.nextCursor ?? ''); };
  const activate = async (item: Notification) => {
    await markNotificationRead(item.id);
    setItems(current => current.map(value => value.id === item.id ? { ...value, readAt: new Date().toISOString() } : value));
    navigate(item.deepLink);
  };
  return <div className="relative">
    <Button aria-label="Notifications" className="w-full justify-start" onClick={() => setOpen(value => !value)} variant="ghost"><Bell aria-hidden="true" size={16} />Notifications{items.some(item => !item.readAt) ? <span className="ml-auto h-2 w-2 rounded-full bg-[#6965db]" /> : null}</Button>
    {open ? <div className="absolute bottom-full left-0 z-50 mb-2 grid max-h-80 w-72 gap-1 overflow-auto rounded-lg border bg-popover p-2 shadow-xl">{items.length ? items.map(item => <button className="rounded-md p-2 text-left text-sm hover:bg-accent" key={item.id} onClick={() => void activate(item)} type="button"><strong>{item.actorName}</strong> updated a discussion</button>) : <p className="p-2 text-sm text-muted-foreground">No notifications</p>}{nextCursor ? <Button onClick={() => void loadMore()} size="sm" variant="ghost">Load more</Button> : null}</div> : null}
  </div>;
}
