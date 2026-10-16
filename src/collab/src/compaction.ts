import type {SceneRef} from '@collab/contracts';
import type {Persistence} from './persistence.js';
const CAPACITY=64;
/** One bounded scan and one scene job; durable rows recover work deferred by memory limits. */
export class Compactor {
 private dirty=new Map<string,{ref:SceneRef;count:number;timer:NodeJS.Timeout}>();
 private queued=new Map<string,SceneRef>();
 private running:{ref:SceneRef;again:boolean}|undefined;
 private sceneTask:Promise<void>|undefined;
 private scanTask:Promise<void>|undefined;
 private cursor:SceneRef|undefined;
 private closed=false;
 private scan:NodeJS.Timeout;
 constructor(private store:Persistence,private onFailure:(error:unknown,ref?:SceneRef)=>void){
  store.on('committed',this.committed);
  this.scan=setInterval(()=>this.scanDue(),1000);this.scan.unref();
 }
 private scanDue(){
  if(this.closed||this.scanTask||this.queued.size>=CAPACITY)return;
  const limit=CAPACITY-this.queued.size;
  this.scanTask=(async()=>{
   try{
    const due=await this.store.listDueScenes(limit,this.cursor);
    this.cursor=due.length===limit?due[due.length-1]:undefined;
    if(!this.closed)for(const ref of due)this.enqueue(ref,false);
   }catch(error){this.onFailure(error);}
  })().finally(()=>{this.scanTask=undefined;});
 }
 private enqueue(ref:SceneRef,newMutation:boolean){
  if(this.closed)return;
  if(this.running?.ref.sceneId===ref.sceneId){if(newMutation)this.running.again=true;return;}
  if(this.queued.has(ref.sceneId)||this.queued.size>=CAPACITY)return;
  this.queued.set(ref.sceneId,ref);this.pump();
 }
 private pump(){
  if(this.closed||this.sceneTask||this.queued.size===0)return;
  const ref=this.queued.values().next().value!;this.queued.delete(ref.sceneId);
  this.running={ref,again:false};
  this.sceneTask=(async()=>{
   try{await this.store.compactScene(ref);}catch(error){this.onFailure(error,ref);}
  })().finally(()=>{
   const again=this.running?.again;this.running=undefined;this.sceneTask=undefined;
   if(again&&!this.closed&&this.queued.size<CAPACITY)this.queued.set(ref.sceneId,ref);
   this.pump();
  });
 }
 private flush=(key:string)=>{
  const entry=this.dirty.get(key);if(!entry)return;
  clearTimeout(entry.timer);this.dirty.delete(key);this.enqueue(entry.ref,true);
 };
 private committed=({ref}:{ref:SceneRef})=>{
  if(this.closed)return;
  if(this.running?.ref.sceneId===ref.sceneId){this.running.again=true;return;}
  if(this.queued.has(ref.sceneId))return;
  let entry=this.dirty.get(ref.sceneId);
  if(!entry){
   if(this.dirty.size>=CAPACITY)return;
   entry={ref,count:0,timer:setTimeout(()=>this.flush(ref.sceneId),5000)};
   entry.timer.unref();this.dirty.set(ref.sceneId,entry);
  }
  if(++entry.count>=100)this.flush(ref.sceneId);
 };
 async close(){
  this.closed=true;clearInterval(this.scan);this.store.off('committed',this.committed);
  for(const entry of this.dirty.values())clearTimeout(entry.timer);
  this.dirty.clear();this.queued.clear();
  await Promise.allSettled([this.scanTask,this.sceneTask].filter((task):task is Promise<void>=>task!==undefined));
 }
}
