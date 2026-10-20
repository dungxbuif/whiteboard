import { useEffect, useSyncExternalStore } from 'react';
import type { NativeElement } from '@collab/contracts';
import { Button } from '@/components/ui/button';
import { presentationFrame, type LivePresentation } from '@/collab/live-presentation';

export function AudienceView({live,elements,onFocusFrame,showControls=true}:{live:LivePresentation;elements:NativeElement[];onFocusFrame?:(element:NativeElement)=>void;showControls?:boolean}){
 const state=useSyncExternalStore(live.subscribe,live.getSnapshot),target=presentationFrame(elements,state.frameId);
 const spotlight=state.spotlight?[...state.admitted,{connectionId:state.hostConnectionId??'',principalId:'',displayName:'Host'}].find(person=>person.connectionId===state.spotlight):undefined;
 useEffect(()=>{if(state.locked&&target.kind==='frame')onFocusFrame?.(target.element);},[state.locked,state.frameId,target.kind]);
 useEffect(()=>{if(!state.locked||target.kind!=='frame')return;let timer:ReturnType<typeof setTimeout>|undefined;const restore=()=>onFocusFrame?.(target.element),afterWheel=()=>{clearTimeout(timer);timer=setTimeout(restore,0);};window.addEventListener('pointerup',restore,true);window.addEventListener('keyup',restore,true);window.addEventListener('wheel',afterWheel,{capture:true,passive:true});return()=>{clearTimeout(timer);window.removeEventListener('pointerup',restore,true);window.removeEventListener('keyup',restore,true);window.removeEventListener('wheel',afterWheel,true);};},[state.locked,state.frameId,target.kind,onFocusFrame]);
 useEffect(()=>{const key=(event:KeyboardEvent)=>{if(event.key==='Escape'){event.preventDefault();live.leave();}};window.addEventListener('keydown',key,true);return()=>window.removeEventListener('keydown',key,true);},[live]);
 if(!['audience','host'].includes(state.status))return null;
 return <section aria-label="Live presentation audience controls" className="pointer-events-auto absolute bottom-5 left-1/2 z-30 flex max-w-[calc(100vw-1.5rem)] -translate-x-1/2 flex-wrap items-center justify-center gap-2 rounded-full border bg-card/95 px-3 py-2 shadow-lg backdrop-blur">
  {target.kind==='missing'?<span className="text-sm" role="status">The presented frame was removed.</span>:state.locked?<span className="text-sm">Host controls the view</span>:null}
  {spotlight?<span className="text-sm">Spotlight: {spotlight.displayName}</span>:null}
  {showControls&&(['👏','❤️','👍','🎉','😂'] as const).map(emoji=><Button aria-label={`React ${emoji}`} className="min-h-11 min-w-11" key={emoji} onClick={()=>live.react(emoji)} size="icon" variant="ghost">{emoji}</Button>)}
  {showControls&&<Button className="min-h-11" onClick={()=>live.leave()} variant="outline">Leave</Button>}
  <div aria-live="polite" className="pointer-events-none absolute bottom-full mb-2 flex gap-1">{state.reactions.map(item=><span className="animate-bounce text-2xl" key={item.id}>{item.emoji}</span>)}</div>
 </section>;
}
