import { useSyncExternalStore } from 'react';
import { Mic, MicOff, PhoneOff, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCollab } from '@/collab/provider';
import type { Presence } from '@/collab/presence';
import type { Voice } from '@/collab/voice';
export function CollaborationToolbar() {
 const {presence,voice}=useCollab();
 return presence&&voice?<LiveToolbar presence={presence} voice={voice}/>:null;
}
function LiveToolbar({presence,voice}:{presence:Presence;voice:Voice}) {
 const room=useSyncExternalStore(presence.subscribe,presence.getSnapshot),call=useSyncExternalStore(voice.subscribe,voice.getSnapshot);
 const connected=!!room.connectionId,joined=call.status==='connected',waiting=['admitting','permission'].includes(call.status);
 const voiceCount=room.participants.filter(p=>p.voiceJoined).length;
 return <div className="flex max-w-full flex-wrap items-center gap-2" aria-label="Collaboration controls">
  <details className="relative">
   <summary className="flex min-h-10 cursor-pointer list-none items-center gap-2 rounded-md border bg-card px-3 text-sm focus-visible:outline focus-visible:outline-2" aria-label={`People (${room.participants.length})`}><Users size={16} aria-hidden="true"/>People ({room.participants.length})</summary>
   <div className="absolute left-0 top-full z-50 mt-2 max-h-[60vh] w-80 max-w-[85vw] overflow-auto rounded-md border bg-popover p-3 text-popover-foreground shadow-md" aria-label="Room participants">
    {!connected&&<p className="text-sm text-muted-foreground">Participants appear when connected.</p>}
    <ul className="space-y-3">{room.participants.map(p=><li key={p.connectionId} className="flex items-center justify-between gap-3">
     <div className="min-w-0"><p className="break-words text-sm font-medium">{p.displayName}{p.connectionId===room.connectionId?' (you)':''}</p><p className="text-xs text-muted-foreground">{p.access==='edit'?'Editor':'Viewer'}{p.voiceJoined?` · Voice${p.voiceMuted?' muted':''}`:''}</p></div>
     {p.connectionId!==room.connectionId&&<Button className="min-h-10 shrink-0" size="sm" variant={room.following===p.connectionId?'secondary':'outline'} onClick={()=>presence.follow(room.following===p.connectionId?null:p.connectionId)} aria-label={`${room.following===p.connectionId?'Stop following':'Follow'} ${p.displayName}`}>{room.following===p.connectionId?'Following':'Follow'}</Button>}
    </li>)}</ul>
   </div>
  </details>
  {room.following&&<Button variant="secondary" onClick={()=>presence.follow(null)}>Stop following</Button>}
  <span className="whitespace-nowrap text-xs text-muted-foreground">Voice {voiceCount}/10</span>
  {!joined&&!waiting&&<Button variant="outline" disabled={!connected} onClick={()=>voice.join()}><Mic size={15} aria-hidden="true"/>Join voice</Button>}
  {waiting&&<span className="text-xs" aria-live="polite">{call.status==='admitting'?'Joining voice…':'Allow microphone to join…'}</span>}
  {joined&&<Button variant="outline" onClick={()=>voice.toggleMute()} aria-label={call.muted?'Unmute microphone':'Mute microphone'}>{call.muted?<MicOff size={15} aria-hidden="true"/>:<Mic size={15} aria-hidden="true"/>}{call.muted?'Unmute':'Mute'}</Button>}
  {(joined||waiting)&&<Button variant="ghost" onClick={()=>voice.leave()} aria-label="Leave voice"><PhoneOff size={15} aria-hidden="true"/>Leave voice</Button>}
  {call.audioBlocked&&<Button variant="outline" onClick={()=>void voice.resumeAudio()}>Play voice audio</Button>}
  {(call.error||room.error)&&<p role="alert" className="w-full text-sm text-destructive">{call.error||room.error}</p>}
 </div>;
}
