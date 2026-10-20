import {EventEmitter} from 'node:events';
import {expect,it,vi} from 'vitest';
import {VersionCheckpointer} from './version-checkpointer.js';

class Store extends EventEmitter {captureAutomaticVersion=vi.fn(async()=>undefined);listDueVersionScenes=vi.fn(async()=>[]);}
const ref={workspaceId:'10000000-0000-0000-0000-000000000001',sceneId:'20000000-0000-0000-0000-000000000002'};

it('waits five dirty minutes and coalesces later edits',async()=>{
 let now=0;const store=new Store(),fail=vi.fn(),scheduler=new VersionCheckpointer(store,fail,300_000,()=>now);
 store.emit('committed',{ref,receipt:{requestId:'a',sequence:'2',stateVector:''}});
 now=299_999;await scheduler.flushDue();expect(store.captureAutomaticVersion).not.toHaveBeenCalled();
 store.emit('committed',{ref,receipt:{requestId:'b',sequence:'3',stateVector:''}});
 now=300_000;await scheduler.flushDue();expect(store.captureAutomaticVersion).toHaveBeenCalledOnce();
 await scheduler.close();expect(fail).not.toHaveBeenCalled();
});

it('keeps a scene dirty when a newer edit arrives during capture',async()=>{
 let release!:()=>void;const pending=new Promise<void>(resolve=>{release=resolve;});
 let now=0;const store=new Store();store.captureAutomaticVersion=vi.fn(async()=>{await pending;return undefined;});
 const scheduler=new VersionCheckpointer(store,()=>{},300_000,()=>now);
 store.emit('committed',{ref,receipt:{requestId:'a',sequence:'2',stateVector:''}});now=300_000;
 const flush=scheduler.flushDue();store.emit('committed',{ref,receipt:{requestId:'b',sequence:'3',stateVector:''}});release();await flush;
 now=600_000;await scheduler.flushDue();expect(store.captureAutomaticVersion).toHaveBeenCalledTimes(2);await scheduler.close();
});

it('recovers durable dirty scenes that were not held in memory',async()=>{
 let now=300_000;const store=new Store();store.listDueVersionScenes=vi.fn(async()=>[{ref,sequence:'9'}]);
 const scheduler=new VersionCheckpointer(store,()=>{},300_000,()=>now);
 await scheduler.scanDue();await scheduler.flushDue();expect(store.captureAutomaticVersion).toHaveBeenCalledWith(ref);await scheduler.close();
});
