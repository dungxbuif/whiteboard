import {useSyncExternalStore} from 'react';
import {MonitorUp,Square} from 'lucide-react';
import {Button} from '@/components/ui/button';
import type {ScreenShare} from '@/collab/screen-share';

export function ScreenSharePanel({screenShare,canPublish}:{screenShare:ScreenShare;canPublish:boolean}){
 const state=useSyncExternalStore(screenShare.subscribe,screenShare.getSnapshot);
 return <>
  {state.status==='idle'&&canPublish?<Button className="min-h-11" variant="outline" onClick={()=>void screenShare.join().catch(()=>{})}><MonitorUp aria-hidden="true" size={15}/>Share screen</Button>:null}
  {state.status==='admitting'?<span className="text-xs" aria-live="polite">Requesting screen…</span>:null}
  {state.status==='ready'?<Button className="min-h-11" variant="outline" onClick={()=>void screenShare.startSharing()}><MonitorUp aria-hidden="true" size={15}/>Start sharing</Button>:null}
  {state.status==='permission'?<span className="text-xs" aria-live="polite">Choose a screen or window…</span>:null}
  {state.status==='sharing'?<Button className="min-h-11" variant="outline" onClick={()=>screenShare.leave()}><Square aria-hidden="true" size={14}/>Stop sharing</Button>:null}
  {state.status==='receiving'?<div className="fixed bottom-[max(0.75rem,env(safe-area-inset-bottom))] left-[max(0.75rem,env(safe-area-inset-left))] z-30 grid w-[min(32rem,calc(100vw-1.5rem))] gap-2 rounded-lg border bg-card/95 p-2 shadow-lg backdrop-blur"><video aria-label="Shared screen" className="max-h-[45dvh] w-full rounded bg-black object-contain" ref={element=>screenShare.attachVideo(element)}/><Button size="sm" variant="ghost" onClick={()=>screenShare.leave()}>Stop watching</Button></div>:null}
  {state.error?<p role="alert" className="max-w-64 text-xs text-destructive">{state.error}</p>:null}
 </>;
}
