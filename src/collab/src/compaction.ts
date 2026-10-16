import type {SceneRef} from '@collab/contracts';
import type {Persistence} from './persistence.js';
/** Each dirty scene is scheduled once; count threshold runs immediately. Startup scans recover missed timers. */
export class Compactor {
 private dirty=new Map<string,{ref:SceneRef;count:number;timer:NodeJS.Timeout}>();
 private tasks=new Set<Promise<void>>();private closed=false;
 private scan:NodeJS.Timeout;
 constructor(private store:Persistence,private onFailure:(error:unknown)=>void){
  store.on('committed',this.committed);this.scan=setInterval(()=>this.run(store.compactDue()),1000);this.scan.unref();
 }
 private run(task:Promise<void>){this.tasks.add(task);void task.catch(this.onFailure).finally(()=>this.tasks.delete(task));}
 private flush=(key:string)=>{const entry=this.dirty.get(key);if(!entry)return;clearTimeout(entry.timer);this.dirty.delete(key);if(!this.closed)this.run(this.store.compactScene(entry.ref));};
 private committed=({ref}:{ref:SceneRef})=>{if(this.closed)return;let entry=this.dirty.get(ref.sceneId);if(!entry){entry={ref,count:0,timer:setTimeout(()=>this.flush(ref.sceneId),5000)};entry.timer.unref();this.dirty.set(ref.sceneId,entry);}if(++entry.count>=100)this.flush(ref.sceneId);};
 async close(){this.closed=true;clearInterval(this.scan);this.store.off('committed',this.committed);for(const entry of this.dirty.values())clearTimeout(entry.timer);this.dirty.clear();await Promise.allSettled(this.tasks);}
}
