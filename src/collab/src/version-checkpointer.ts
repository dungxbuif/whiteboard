import type {SceneRef,MutationReceipt} from '@collab/contracts';
import type {Persistence} from './persistence.js';

const CAPACITY=64;

/** Bounded in-memory dirty timers; durable document sequences are rechecked under the writer lock. */
export class VersionCheckpointer {
 private dirty=new Map<string,{ref:SceneRef;sequence:string;dirtyAt:number}>();
 private running:Promise<void>|undefined;
 private closed=false;
 private timer:NodeJS.Timeout;
 constructor(private store:Pick<Persistence,'on'|'off'|'captureAutomaticVersion'>,private onFailure:(error:unknown,ref:SceneRef)=>void,private delayMs=300_000,private now=()=>Date.now()){
  store.on('committed',this.committed);
  this.timer=setInterval(()=>void this.flushDue(),1000);this.timer.unref();
 }
 private committed=({ref,receipt}:{ref:SceneRef;receipt:MutationReceipt})=>{
  if(this.closed)return;
  const current=this.dirty.get(ref.sceneId);
  if(current){current.sequence=receipt.sequence;return;}
  if(this.dirty.size<CAPACITY)this.dirty.set(ref.sceneId,{ref,sequence:receipt.sequence,dirtyAt:this.now()});
 };
 async flushDue(at=this.now()):Promise<void>{
  if(this.closed||this.running)return this.running;
  const due=[...this.dirty.entries()].find(([,entry])=>at-entry.dirtyAt>=this.delayMs);
  if(!due)return;
  const [key,entry]=due;
  const scheduledSequence=entry.sequence;
  this.running=(async()=>{
   try{
    await this.store.captureAutomaticVersion(entry.ref);
    if(this.dirty.get(key)?.sequence===scheduledSequence)this.dirty.delete(key);
   }catch(error){this.onFailure(error,entry.ref);}
  })().finally(()=>{this.running=undefined;});
  return this.running;
 }
 async close():Promise<void>{
  this.closed=true;clearInterval(this.timer);this.store.off('committed',this.committed);this.dirty.clear();await this.running;
 }
}
