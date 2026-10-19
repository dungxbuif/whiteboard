import { useEffect, useState } from 'react';
import api from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

const scopeOptions = [
  ['workspace:read', 'Discover this workspace'],
  ['scene:read', 'Read scenes'],
  ['scene:create', 'Create scenes'],
  ['scene:write', 'Edit scene elements'],
  ['scene:move', 'Move scenes'],
  ['scene:trash', 'Move scenes to trash'],
  ['scene:export', 'Export scenes'],
] as const;
type KeyMetadata = { id: string; scopes: string[]; sceneIds: string[]; createdAt: string; revokedAt: string | null };
type Props = { workspaceId: string; isOwner: boolean; scenes?: { id: string; title: string }[] };

export function AgentSettings({ workspaceId, isOwner, scenes = [] }: Props) {
  if (!isOwner) return <p className="text-sm text-muted-foreground">Only workspace owners can manage agent keys.</p>;
  return <AgentKeyEditor key={workspaceId} workspaceId={workspaceId} scenes={scenes} />;
}

function AgentKeyEditor({ workspaceId, scenes }: Omit<Props, 'isOwner'>) {
  const [keys, setKeys] = useState<KeyMetadata[]>([]);
  const [scopes, setScopes] = useState<string[]>(['workspace:read', 'scene:read']);
  const [sceneIds, setSceneIds] = useState<string[]>([]);
  const [secret, setSecret] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const path = `/workspaces/${encodeURIComponent(workspaceId)}/agent-keys`;

  useEffect(() => {
    let active = true;
    api.get<KeyMetadata[]>(path)
      .then(response => { if (active) setKeys(response.data); })
      .catch(() => { if (active) setError('Could not load agent keys. Check your workspace access and try again.'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [path]);

  async function create() {
    setBusy(true);
    setError('');
    setSecret('');
    try {
      const { data } = await api.post<KeyMetadata & { key: string }>(path, { scopes, sceneIds });
      const { key, ...metadata } = data;
      setKeys(previous => [...previous, metadata]);
      setSecret(key);
    } catch {
      setError('Could not create the key. Check your selected scopes and workspace access.');
    } finally {
      setBusy(false);
    }
  }

  async function revoke(id: string) {
    setBusy(true);
    setError('');
    try {
      await api.delete(`${path}/${encodeURIComponent(id)}`);
      setKeys(previous => previous.map(key => key.id === id ? { ...key, revokedAt: new Date().toISOString() } : key));
      setSecret('');
    } catch {
      setError('Could not revoke the key. Try again.');
    } finally {
      setBusy(false);
    }
  }

  function toggleScene(id: string) {
    setSceneIds(previous => previous.includes(id) ? previous.filter(value => value !== id) : [...previous, id]);
    setScopes(previous => previous.filter(scope => scope !== 'scene:create'));
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Agent keys</CardTitle>
        <CardDescription>Connect an agent to this workspace with only the access it needs. Keys cannot manage members, sharing, or permissions.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        <fieldset disabled={busy} className="space-y-3">
          <legend className="mb-2 text-sm font-medium">Allowed actions</legend>
          {scopeOptions.map(([scope, label]) => (
            <label key={scope} className="flex items-center gap-3 text-sm">
              <input type="checkbox" checked={scopes.includes(scope)} disabled={scope === 'scene:create' && sceneIds.length > 0}
                onChange={() => setScopes(previous => previous.includes(scope) ? previous.filter(value => value !== scope) : [...previous, scope])} />
              {label}
            </label>
          ))}
        </fieldset>
        {Boolean(scenes?.length) && (
          <fieldset disabled={busy} className="space-y-3">
            <legend className="mb-2 text-sm font-medium">Limit to selected scenes</legend>
            <p className="text-sm text-muted-foreground">Leave all unchecked to allow this workspace. Selecting scenes disables scene creation.</p>
            {scenes?.map(scene => (
              <label className="flex items-center gap-3 text-sm" key={scene.id}>
                <input type="checkbox" checked={sceneIds.includes(scene.id)} onChange={() => toggleScene(scene.id)} />
                <span className="min-w-0 break-words">{scene.title}</span>
              </label>
            ))}
          </fieldset>
        )}
        <Button disabled={busy || scopes.length === 0} onClick={() => void create()}>Create key</Button>
        {secret && (
          <div className="space-y-3 rounded-lg border bg-muted/30 p-4">
            <label htmlFor="agent-key-once" className="block text-sm font-medium">Copy this key now. It will not be shown again.</label>
            <Input id="agent-key-once" readOnly value={secret} autoComplete="off" spellCheck={false} onFocus={event => event.target.select()} />
            <Button variant="outline" onClick={() => setSecret('')}>I've saved this key</Button>
          </div>
        )}
        <div className="space-y-3">
          <h4 className="font-medium">Existing keys</h4>
          {loading ? <p role="status" className="text-sm text-muted-foreground">Loading keys…</p>
            : keys.length === 0 ? <p className="text-sm text-muted-foreground">No agent keys yet.</p>
              : keys.map(key => (
                <div key={key.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
                  <div className="min-w-0 space-y-1">
                    <p className="break-all font-mono text-xs">{key.id}</p>
                    <p className="break-words text-sm text-muted-foreground">{key.scopes.join(', ')}</p>
                    <p className="text-xs text-muted-foreground">{key.sceneIds.length ? `${key.sceneIds.length} selected scenes` : 'Entire workspace'} · {key.revokedAt ? 'Revoked' : 'Active'}</p>
                  </div>
                  {!key.revokedAt && <Button variant="destructive" size="sm" disabled={busy} onClick={() => void revoke(key.id)}>Revoke</Button>}
                </div>
              ))}
        </div>
        <p className="text-sm text-muted-foreground">
          Use a client that supports bearer headers, or the remote stdio adapter.{' '}
          <a className="text-primary underline" href="/docs/agents.md" target="_blank" rel="noreferrer">Read the connection guide</a>.
        </p>
      </CardContent>
    </Card>
  );
}
