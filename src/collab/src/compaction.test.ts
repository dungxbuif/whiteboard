import {it,expect,vi} from 'vitest';
import {EventEmitter} from 'node:events';
import {Compactor} from './compaction.js';
import type {Persistence} from './persistence.js';
const ref={workspaceId:'workspace',sceneId:'scene'};
function deferred(){let resolve!:()=>void;const promise=new Promise<void>(r=>resolve=r);return {promise,resolve};}
it('only one due scan can wait through multiple ticks and shutdown drains it',async()=>{
 vi.useFakeTimers();const pending=deferred();let active=0,max=0,started=0;
 const scan=async()=>{started++;active++;max=Math.max(max,active);await pending.promise;active--;return [];};const store=Object.assign(new EventEmitter(),{compactDue:scan,listDueScenes:scan,compactScene:async()=>{}});
 const compactor=new Compactor(store as unknown as Persistence,()=>{});
 try{await vi.advanceTimersByTimeAsync(3150);expect(started).toBe(1);expect(max).toBe(1);const closing=compactor.close();let closed=false;void closing.then(()=>closed=true);await Promise.resolve();expect(closed).toBe(false);pending.resolve();await closing;expect(active).toBe(0);}finally{pending.resolve();await compactor.close();vi.useRealTimers();}
});
it('timer, count and scans deduplicate one running scene and bound other scene jobs',async()=>{
 vi.useFakeTimers();const first=deferred();let active=0,max=0;const calls:string[]=[];
 const store=Object.assign(new EventEmitter(),{listDueScenes:async()=>[ref],compactDue:async()=>{await store.compactScene(ref);},compactScene:async(r:typeof ref)=>{calls.push(r.sceneId);active++;max=Math.max(max,active);if(calls.length===1)await first.promise;active--;}});
 const compactor=new Compactor(store as unknown as Persistence,()=>{});
 try{for(let i=0;i<100;i++)store.emit('committed',{ref});await vi.advanceTimersByTimeAsync(6000);for(let i=0;i<100;i++)store.emit('committed',{ref});for(let i=0;i<500;i++)store.emit('committed',{ref:{...ref,sceneId:'scene-'+i}});await vi.advanceTimersByTimeAsync(6000);expect(max).toBe(1);expect(calls).toEqual(['scene']);first.resolve();await vi.advanceTimersByTimeAsync(1000);expect(calls.length).toBeLessThanOrEqual(67);expect(calls.filter(id=>id==='scene').length).toBeLessThanOrEqual(2);await compactor.close();expect(active).toBe(0);}finally{first.resolve();await compactor.close();vi.useRealTimers();}
});

it('bounded pages rediscover jobs deferred by full timer and queue capacity',async()=>{
 vi.useFakeTimers();const first=deferred();const pending=new Map(Array.from({length:150},(_,i)=>{const r={workspaceId:'workspace',sceneId:String(i).padStart(3,'0')};return [r.sceneId,r];}));let active=0,max=0,scans=0;const completed=new Set<string>();
 const store=Object.assign(new EventEmitter(),{listDueScenes:async(limit:number,after?:typeof ref)=>{scans++;expect(limit).toBeLessThanOrEqual(64);return [...pending.values()].filter(r=>!after||r.sceneId>after.sceneId).slice(0,limit);},compactScene:async(r:typeof ref)=>{active++;max=Math.max(max,active);if(r.sceneId==='000')await first.promise;pending.delete(r.sceneId);completed.add(r.sceneId);active--;}});
 const compactor=new Compactor(store as unknown as Persistence,()=>{});
 try{for(const r of pending.values())for(let i=0;i<100;i++)store.emit('committed',{ref:r});await vi.advanceTimersByTimeAsync(5000);expect(max).toBe(1);expect(completed.size).toBe(0);first.resolve();await vi.advanceTimersByTimeAsync(6000);expect(completed.size).toBe(150);expect(scans).toBeGreaterThan(1);expect(max).toBe(1);}finally{first.resolve();await compactor.close();vi.useRealTimers();}
});
