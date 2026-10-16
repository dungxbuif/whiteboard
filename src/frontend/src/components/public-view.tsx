import { useEffect, useState, type ComponentType } from 'react';
import '@excalidraw/excalidraw/index.css';
import { readPublicScene, type PublicScene } from '@/lib/public-api';
import { createPublicSession, accessError, type SceneSession } from '@/lib/collaboration-api';
import { normalizeSceneContent } from '@/lib/scene-content';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import type { Scene } from '@/store/scene-store';
type ExcalidrawModule = typeof import('@excalidraw/excalidraw');

export type PublicSceneEditorProps = { scene: PublicScene; session: SceneSession; onSessionChange: (session: SceneSession) => void; onAccessLost: () => void };
// B3 replaces this guarded snapshot with the shared Yjs canvas after room admission.
export function PublicSceneEditor({ scene, session }: PublicSceneEditorProps) {
  return <div><p className="bg-muted px-4 py-2 text-sm" role="status">{session.displayName} · Edit access · Live editing is awaiting collaboration connection.</p><SceneReader scene={scene} /></div>;
}
export function PublicView({ token }: { token: string }) {
  const [scene, setScene] = useState<PublicScene | null>(null); const [session, setSession] = useState<SceneSession | null>(null); const [password, setPassword] = useState(''); const [name, setName] = useState(''); const [challenge, setChallenge] = useState<'password' | 'name' | null>(null); const [error, setError] = useState(''); const [busy, setBusy] = useState(false);
  useEffect(() => { let current = true; setBusy(true); createPublicSession({ token }).then(async credential => { const data = await readPublicScene(token); if (current) { setSession(credential); setScene(data); } }).catch(cause => { if (!current) return; const code = cause.response?.data?.code; if (code === 'SHARE_PASSWORD_REQUIRED') setChallenge('password'); else if (code === 'INVALID_INPUT') setChallenge('name'); else setError('This shared scene is unavailable. Ask the owner for a current link.'); }).finally(() => { if (current) setBusy(false); }); return () => { current = false; }; }, [token]);
  async function open(event: React.FormEvent) { event.preventDefault(); setBusy(true); setError(''); try { const credential = await createPublicSession({ token, ...(password ? { password } : {}), ...(name ? { displayName: name.trim() } : {}) }); const data = await readPublicScene(token, password, name.trim()); setSession(credential); setScene(data); setPassword(''); setChallenge(null); } catch (cause) { const code = (cause as {response?: {data?: {code?: string}}}).response?.data?.code; if (code === 'SHARE_PASSWORD_REQUIRED') { setChallenge('password'); setError(accessError(cause, 'Enter the share password.')); } else if (code === 'INVALID_INPUT' && !name.trim()) { setChallenge('name'); setError('Enter a display name to join an edit link.'); } else setError('This shared scene is unavailable. Ask the owner for a current link.'); } finally { setBusy(false); } }
  if (scene && session) return session.access === 'edit' ? <PublicSceneEditor scene={scene} session={session} onSessionChange={setSession} onAccessLost={() => { setSession(null); setScene(null); setError('Access to this shared scene ended. Ask the owner for a current link.'); }} /> : <section aria-label="Public Excalidraw canvas"><SceneReader scene={scene} /></section>;
  if (challenge) return <main className="grid min-h-screen place-items-center bg-muted/30 p-6"><Card className="w-full max-w-sm"><CardHeader><CardTitle>{challenge === 'password' ? 'Password required' : 'Join shared scene'}</CardTitle></CardHeader><CardContent><form className="grid gap-4" onSubmit={event => void open(event)}><label className="grid gap-2 text-sm">Share password<Input autoFocus={challenge === 'password'} autoComplete="off" type="password" placeholder="Share password" value={password} onChange={event => setPassword(event.target.value)} /></label>{challenge === 'name' && <label className="grid gap-2 text-sm">Display name<Input autoFocus required maxLength={100} value={name} onChange={event => setName(event.target.value)} /></label>}{error && <p role="alert" className="text-sm text-destructive">{error}</p>}<Button disabled={busy}>{busy ? 'Opening…' : 'Open shared scene'}</Button></form></CardContent></Card></main>;
  if (error) return <main className="grid min-h-screen place-items-center p-6"><p role="alert" className="text-destructive">{error}</p></main>;
  return <main className="grid min-h-screen place-items-center">Loading shared scene...</main>;
}

export function SceneReader({ scene, onBack }: { scene: Pick<Scene, 'title' | 'content'>; onBack?: () => void }) {
  const [module, setModule] = useState<ExcalidrawModule | null>(null);
  useEffect(() => { void import('@excalidraw/excalidraw').then(setModule); }, []);
  if (!module) return <main className="grid min-h-screen place-items-center">Loading scene...</main>;
  const Excalidraw = module.Excalidraw as ComponentType<Record<string, unknown>>;
  return <main className="grid h-screen grid-rows-[56px_minmax(0,1fr)] bg-background"><header className="flex items-center gap-3 border-b bg-card px-4">{onBack && <Button aria-label="Back to dashboard" variant="ghost" onClick={onBack}>Back</Button>}<h1 className="truncate font-semibold">{scene.title}</h1><span className="ml-auto text-sm text-muted-foreground">Read-only</span></header><section className="min-h-0" aria-label="Excalidraw canvas"><Excalidraw initialData={normalizeSceneContent(scene.content)} name={scene.title} viewModeEnabled /></section></main>;
}
