import type {SceneRef,MutationReceipt} from '@collab/contracts';
import type {Persistence} from './persistence.js';

const CAPACITY=64;

/** Bounded in-memory dirty timers; durable document sequences are rechecked under the writer lock. */
export class VersionCheckpointer {
 private dirty=new Map<string,{ref:SceneRef;sequence:string;dirtyAt:number}>();
 private running:Promise<void>|undefined;
 private scanTask:Promise<void>|undefined;
 private cursor:SceneRef|undefined;
 private closed=false;
 private timer:NodeJS.Timeout;
 constructor(private store:Pick<Persistence,'on'|'off'|'captureAutomaticVersion'|'listDueVersionScenes'>,private onFailure:(error:unknown,ref?:SceneRef)=>void,private delayMs=300_000,private now=()=>Date.now()){
  store.on('committed',this.committed);
  this.timer=setInterval(()=>void this.tick(),1000);this.timer.unref();
 }
 private tick(){void this.scanDue();void this.flushDue();}
 scanDue():Promise<void>|undefined{
  if(this.closed||this.scanTask||this.dirty.size>=CAPACITY)return;
  const limit=CAPACITY-this.dirty.size;
  this.scanTask=this.store.listDueVersionScenes(limit,this.cursor).then(rows=>{
   this.cursor=rows.length===limit?rows[rows.length-1].ref:undefined;
   if(!this.closed)for(const row of rows)if(!this.dirty.has(row.ref.sceneId))this.dirty.set(row.ref.sceneId,{...row,dirtyAt:this.now()-this.delayMs});
  }).catch(error=>this.onFailure(error)).finally(()=>{this.scanTask=undefined;});
  return this.scanTask;
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
  this.closed=true;clearInterval(this.timer);this.store.off('committed',this.committed);this.dirty.clear();await Promise.allSettled([this.running,this.scanTask].filter((task):task is Promise<void>=>task!==undefined));
 }
}
