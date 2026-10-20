import { useState, useSyncExternalStore } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { CollabProvider, useCollab } from '@/collab/provider';
import { joinPresentation, type PendingPresentationGrant } from '@/lib/presentation-api';
import { navigate } from '@/lib/router';
import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
import { WaitingRoom } from './waiting-room';

function ActivePresentation({onRejoin}:{onRejoin:()=>void}){
 const collab=useCollab(),live=useSyncExternalStore(collab.livePresentation.subscribe,collab.livePresentation.getSnapshot);
 if(live.status==='ended'||live.status==='error')return <WaitingRoom error={live.error} onLeave={()=>navigate('/')} onRejoin={onRejoin} status={live.reason==='REMOVED'?'removed':'ended'}/>;
 if(live.status!=='audience'||!['readonly','synced','pending'].includes(collab.state.status))return <WaitingRoom onLeave={()=>{collab.livePresentation.leave();navigate('/')}} status={live.status==='waiting'?'waiting':'joining'}/>;
 return <CollaborativeCanvas displayName={collab.principal.current.kind==='presentation'?collab.principal.current.credential.displayName:undefined} title="Live presentation"/>;
}

export function PresentationView({token}:{token:string}){
 const [name,setName]=useState(''),[credential,setCredential]=useState<PendingPresentationGrant|null>(null),[error,setError]=useState(''),[busy,setBusy]=useState(false);
 async function join(event:React.FormEvent){event.preventDefault();setBusy(true);setError('');try{setCredential(await joinPresentation(token,name.trim()));}catch(cause){const code=(cause as {response?:{data?:{code?:string}}}).response?.data?.code;setError(code==='RATE_LIMITED'?'Too many join attempts. Try again in a minute.':code==='PRESENTATION_ENDED'?'This presentation has ended.':'This invite is unavailable. Ask the host for a current link.');}finally{setBusy(false)}}
 if(credential)return <CollabProvider principal={{kind:'presentation',id:credential.principalId,credential,onCredentialChange:setCredential}} sceneRef={{workspaceId:credential.workspaceId,sceneId:credential.sceneId}}><ActivePresentation onRejoin={()=>setCredential(null)}/></CollabProvider>;
 return <main className="grid min-h-[100dvh] place-items-center bg-muted/30 p-4"><Card className="w-full max-w-sm"><CardHeader><CardTitle>Join live presentation</CardTitle></CardHeader><CardContent><form className="grid gap-4" onSubmit={event=>void join(event)}><label className="grid gap-2 text-sm font-medium">Display name<Input autoFocus maxLength={100} onChange={event=>setName(event.target.value)} required value={name}/></label>{error?<p className="text-sm text-destructive" role="alert">{error}</p>:null}<Button className="min-h-11" disabled={busy||!name.trim()}>{busy?'Joining…':'Join and wait'}</Button></form></CardContent></Card></main>;
}
