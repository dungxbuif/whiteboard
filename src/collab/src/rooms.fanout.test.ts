import {it,expect} from 'vitest';
import {EventEmitter} from 'node:events';
import * as Y from 'yjs';
import {Rooms} from './rooms.js';
import {CollabError} from './auth.js';
import {decodeFrame,MAX_FRAME_BYTES} from '@collab/contracts';
const wait=(ms:number)=>new Promise(r=>setTimeout(r,ms));
function fixture(fail?:Error){
 const store=new EventEmitter() as any;let reads=0;store.read=async()=>{reads++;if(fail)throw fail;return {doc:new Y.Doc(),access:'edit'};};
 const rooms=new Rooms(store,{updatesPerMinute:120,maxBufferedBytes:1024*1024});
 const sent:any[]=[];const socket={readyState:1,bufferedAmount:0,send:(v:any,cb?:()=>void)=>{sent.push(v);cb?.();},close:()=>{},terminate:()=>{}};
 const ref={workspaceId:'workspace',sceneId:'scene'};const session={socket,ref,principal:{principalId:'user'},claims:{exp:Date.now()/1000+60},invalid:false,chain:Promise.resolve()};
 (rooms as any).sessions.add(session);return {store,rooms,sent,ref,reads:()=>reads};
}
it('coalesces a burst into one authorized delivery without losing independent fields',async()=>{
 const f=fixture(),doc=new Y.Doc();
 try{for(let i=0;i<100;i++){let update!:Uint8Array;doc.once('update',u=>update=u);doc.getMap('values').set('field'+i,i);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'update'+i},update});}
 await wait(250);expect(f.reads()).toBe(1);expect(f.sent).toHaveLength(1);const replica=new Y.Doc();Y.applyUpdate(replica,decodeFrame(f.sent[0]).payload);expect(replica.getMap('values').toJSON()).toEqual(doc.getMap('values').toJSON());replica.destroy();
 }finally{doc.destroy();await f.rooms.close();}
});
it.each([[new CollabError('FORBIDDEN'),'AUTHORITY_CHANGED'],[new CollabError('UNAVAILABLE','offline',503,true),'UNAVAILABLE']])('does not deliver after failed fresh authorization (%s)',async(error,reason)=>{
 const f=fixture(error as Error),doc=new Y.Doc();
 try{doc.getMap('values').set('x',1);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'one'},update:Y.encodeStateAsUpdate(doc)});await wait(250);expect(f.sent.every(s=>typeof s==='string')).toBe(true);expect(f.sent.map(s=>JSON.parse(s)).some(m=>m.type==='resync_required'&&m.payload.reason===reason)).toBe(true);}
 finally{doc.destroy();await f.rooms.close();}
});

it('serializes later commits behind the current authority check and bounds pending tiny updates',async()=>{
 const f=fixture(),doc=new Y.Doc();let release!:()=>void;let calls=0;
 f.store.read=async()=>{calls++;if(calls===1)await new Promise<void>(r=>release=r);return {doc:new Y.Doc(),access:'edit'};};
 try{
  doc.getMap('values').set('first',1);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'first'},update:Y.encodeStateAsUpdate(doc)});await wait(150);expect(calls).toBe(1);expect(f.sent).toHaveLength(0);
  doc.getMap('values').set('second',2);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'second'},update:Y.encodeStateAsUpdate(doc)});await wait(150);expect(calls).toBe(1);release();await wait(250);expect(calls).toBe(2);const replica=new Y.Doc();for(const frame of f.sent)Y.applyUpdate(replica,decodeFrame(frame).payload);expect(replica.getMap('values').toJSON()).toEqual({first:1,second:2});replica.destroy();
 }finally{release?.();doc.destroy();await f.rooms.close();}
 const blocked=fixture(),seed=new Y.Doc();
 try{for(let i=0;i<1025;i++)blocked.store.emit('committed',{ref:blocked.ref,receipt:{requestId:'many'},update:Y.encodeStateAsUpdate(seed)});await wait(150);expect(blocked.reads()).toBe(0);expect(blocked.sent.map(s=>JSON.parse(s)).some(m=>m.payload.reason==='BACKPRESSURE')).toBe(true);}
 finally{seed.destroy();await blocked.rooms.close();}
});

it('cancels queued broadcasts when workspace authority is invalidated',async()=>{
 const f=fixture(),doc=new Y.Doc();try{doc.getMap('values').set('x',1);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'one'},update:Y.encodeStateAsUpdate(doc)});f.rooms.invalidateWorkspace(f.ref.workspaceId);await wait(150);expect(f.reads()).toBe(0);expect(f.sent.every(s=>typeof s==='string')).toBe(true);expect((f.rooms as any).fanouts.size).toBe(0);}finally{doc.destroy();await f.rooms.close();}
});

it('announces the highest durable sequence after update frames',async()=>{
 const f=fixture(),doc=new Y.Doc();try{doc.getMap('values').set('x',1);f.store.emit('committed',{ref:f.ref,receipt:{requestId:'one',sequence:'9'},update:Y.encodeStateAsUpdate(doc)});await wait(250);
 const control=f.sent.filter(value=>typeof value==='string').map(value=>JSON.parse(value)).find(value=>value.type==='committed');expect(control?.payload.sequence).toBe('9');
 expect(typeof f.sent[0]).not.toBe('string');
 }finally{doc.destroy();await f.rooms.close();}
});
it('fans a trusted restore larger than the retained fanout budget as bounded sync chunks before its sequence',async()=>{
 const f=fixture(),doc=new Y.Doc();doc.getMap('values').set('large','x'.repeat(4*MAX_FRAME_BYTES+64));f.store.read=async()=>({doc,access:'edit',documentId:'document',sequence:'12',stateVector:''});
 try{f.store.emit('committed',{ref:f.ref,receipt:{requestId:'restore',sequence:'12'},update:Y.encodeStateAsUpdate(doc)});const queued=[...(f.rooms as any).fanouts.values()][0];expect(queued).toMatchObject({bytes:0,resync:true,updates:[{sequence:'12'}]});expect(queued.updates[0].bytes).toBeUndefined();await wait(250);
  const controls=f.sent.map(value=>typeof value==='string'?JSON.parse(value):null).filter(Boolean);const chunks=controls.filter(value=>value.type==='sync_chunk');expect(chunks.length).toBeGreaterThan(1);expect(chunks.every(value=>value.payload.data.length<700000)).toBe(true);expect(controls.at(-1)).toMatchObject({type:'committed',payload:{sequence:'12'}});expect(f.sent.every(value=>typeof value==='string')).toBe(true);
 }finally{doc.destroy();await f.rooms.close();}
});
