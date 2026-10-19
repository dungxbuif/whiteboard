import {afterEach,expect,it} from 'vitest';
import {once} from 'node:events';
import {encodeFrame} from '@collab/contracts';
import {roomFixture} from './test-fixtures/presence.js';
let fixture:Awaited<ReturnType<typeof roomFixture>>;
afterEach(async()=>{await fixture?.close();});
it('keeps roster and awareness connection-scoped while ignoring claimed identity',async()=>{
 fixture=await roomFixture();const a=await fixture.connect(),b=await fixture.connect();
 expect(a.connectionId).toEqual(expect.any(String));expect(a.connectionId).not.toBe(b.connectionId);
 const id=a.send('presence',{displayName:'Alice',connectionId:b.connectionId,principalId:'forged',access:'owner'});
 const roster=await a.wait('roster',id);expect(roster.payload.participants.find((p:any)=>p.connectionId===a.connectionId)).toMatchObject({principalId:'user',displayName:'Alice',access:'edit'});
 a.socket.send(encodeFrame({channel:2,updateId:'cursor',payload:new TextEncoder().encode(JSON.stringify({cursor:{x:10,y:20},connectionId:b.connectionId,principalId:'forged'}))}));
 expect((await b.wait('awareness','cursor')).payload).toMatchObject({connectionId:a.connectionId,principalId:'user',cursor:{x:10,y:20}});
});
it('rejects self and cycles, tells targets their followers and clears follows when targets exit',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a'),b=await fixture.connect('b'),c=await fixture.connect('c');
 const self=a.send('follow',{targetConnectionId:a.connectionId});expect((await a.wait('error',self)).payload.code).toBe('INVALID_FOLLOW');
 const ab=a.send('follow',{targetConnectionId:b.connectionId});expect((await a.wait('follow',ab)).payload.targetConnectionId).toBe(b.connectionId);
 const bc=b.send('follow',{targetConnectionId:c.connectionId});await b.wait('follow',bc);
 const ca=c.send('follow',{targetConnectionId:a.connectionId});expect((await c.wait('error',ca)).payload.code).toBe('INVALID_FOLLOW');
 expect((await b.wait('roster',ab)).payload.participants.find((p:any)=>p.connectionId===a.connectionId).following).toBe(b.connectionId);
 const closed=once(b.socket,'close');b.socket.close();await closed;const reset=await a.wait('follow','server');expect(reset.payload).toMatchObject({connectionId:a.connectionId,targetConnectionId:null});
});
it('delivers bounded viewport only to followers and never persists ephemeral changes',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a'),b=await fixture.connect('b'),c=await fixture.connect('c');
 await b.wait('follow',b.send('follow',{targetConnectionId:a.connectionId}));
 const id=a.send('viewport',{scrollX:10,scrollY:20,zoom:1});expect((await b.wait('viewport',id)).payload).toEqual({connectionId:a.connectionId,scrollX:10,scrollY:20,zoom:1});
 await c.wait('roster',c.send('presence',{displayName:'C'}));expect(c.messages.some(m=>m.type==='viewport')).toBe(false);expect(fixture.writes).toHaveLength(0);
 const bad=a.send('viewport',{scrollX:1e100,scrollY:0,zoom:1});expect((await a.wait('error',bad)).payload.code).toBe('INVALID_PRESENCE');expect(a.socket.readyState).toBe(1);
});

it('roster reflects current effective access and keeps viewport limits independent from voice controls',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a');fixture.hooks.access='view';
 const id=a.send('presence',{displayName:' Viewer\u202e '});expect((await a.wait('roster',id)).payload.participants[0]).toMatchObject({access:'view',displayName:'Viewer'});
 // Cursor/camera bursts must not consume the minute-wide voice-control bucket.
 for(let i=0;i<31;i++)a.send('viewport',{scrollX:i,scrollY:0,zoom:1},'camera-'+i);
 expect((await a.wait('error','camera-30')).payload.code).toBe('PRESENCE_RATE_LIMITED');
 const join=a.send('voice_join');expect((await a.wait('voice_admission',join)).payload.admitted).toBe(true);
});
