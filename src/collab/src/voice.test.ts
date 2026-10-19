import {afterEach,expect,it} from 'vitest';
import {once} from 'node:events';
import {encodeFrame} from '@collab/contracts';
import {roomFixture} from './test-fixtures/presence.js';
let fixture:Awaited<ReturnType<typeof roomFixture>>;
afterEach(async()=>{await fixture?.close();});
it('admits ten distinct connections, repeated join is idempotent and the eleventh still edits',async()=>{
 fixture=await roomFixture();const clients=[];for(let i=0;i<11;i++)clients.push(await fixture.connect('same-user'));
 expect(new Set(clients.map(c=>c.connectionId)).size).toBe(11);expect(clients[0].connectionId).not.toBe('same-user');
 for(const client of clients.slice(0,10)){const id=client.send('voice_join');expect((await client.wait('voice_admission',id)).payload.admitted).toBe(true);}
 const repeated=clients[0].send('voice_join');expect((await clients[0].wait('voice_admission',repeated)).payload.admitted).toBe(true);
 const last=clients[10],id=last.send('voice_join');expect((await last.wait('voice_admission',id)).payload).toMatchObject({admitted:false,reason:'VOICE_FULL'});
 last.socket.send(encodeFrame({channel:3,updateId:'still-editing',payload:new Uint8Array([0,0])}));expect((await last.wait('ack','still-editing')).payload.sequence).toBe('1');expect(fixture.writes).toHaveLength(1);
 const closed=once(clients[0].socket,'close');clients[0].socket.close();await closed;
 expect((await last.wait('roster','server')).payload.participants.some((p:any)=>p.connectionId===clients[0].connectionId)).toBe(false);
 const retry=last.send('voice_join');expect((await last.wait('voice_admission',retry)).payload.admitted).toBe(true);
});
it('signals only admitted same-room peers with server-owned sender identity',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a'),b=await fixture.connect('b'),other=await fixture.connect('other','another');
 for(const client of [a,b,other])await client.wait('voice_admission',client.send('voice_join'));
 const id=a.send('voice_signal',{to:b.connectionId,from:other.connectionId,principalId:'forged',description:{type:'offer',sdp:'v=0'}});
 expect((await b.wait('voice_signal',id)).payload).toEqual({from:a.connectionId,to:b.connectionId,description:{type:'offer',sdp:'v=0'}});
 const bad=a.send('voice_signal',{to:other.connectionId,candidate:{candidate:'candidate'}});expect((await a.wait('error',bad)).payload.code).toBe('VOICE_TARGET');expect(a.socket.readyState).toBe(1);
 b.send('voice_leave');await b.wait('voice_left');const left=a.send('voice_signal',{to:b.connectionId,candidate:null});expect((await a.wait('error',left)).payload.code).toBe('VOICE_TARGET');
});
it('rechecks current recipient authority before delivering signaling and frees revoked slots',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a'),b=await fixture.connect('b');
 for(const client of [a,b])await client.wait('voice_admission',client.send('voice_join'));
 fixture.denied.add('b');const id=a.send('voice_signal',{to:b.connectionId,description:{type:'offer',sdp:'v=0'}});
 expect((await a.wait('error',id)).payload.code).toBe('VOICE_TARGET');expect(b.messages.some(m=>m.type==='voice_signal')).toBe(false);expect(a.socket.readyState).toBe(1);
});
it('protocol ping preserves responsive peers and removes a dead voice participant',async()=>{
 fixture=await roomFixture({heartbeatIntervalMs:100});const live=await fixture.connect('live'),dead=await fixture.connect('dead','scene',false);
 const pings:Buffer[]=[];live.socket.on('ping',data=>pings.push(data));
 await dead.wait('voice_admission',dead.send('voice_join'));const closed=once(dead.socket,'close');await closed;
 await new Promise(resolve=>setTimeout(resolve,120));expect(pings.length).toBeGreaterThanOrEqual(2);expect(live.socket.readyState).toBe(1);
 expect(live.messages.filter(m=>m.type==='roster').at(-1)?.payload.participants.map((p:any)=>p.connectionId)).toEqual([live.connectionId]);
});

it('an admission whose authority read finishes after disconnect cannot leave a ghost slot',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('delayed');
 let release!:()=>void;let entered!:()=>void;const started=new Promise<void>(r=>entered=r);const gate=new Promise<void>(r=>release=r);
 fixture.hooks.beforeRead=async principal=>{if(principal==='delayed'){entered();await gate;}};
 a.send('voice_join');await started;const closed=once(a.socket,'close');a.socket.close();await closed;release();
 const b=await fixture.connect('b');const id=b.send('voice_join');expect((await b.wait('voice_admission',id)).payload.peers).toEqual([]);
 expect((await b.wait('roster',id)).payload.participants.map((p:any)=>p.connectionId)).toEqual([b.connectionId]);
});
it('invalid signals and repeated joins never close the data socket or duplicate voice membership',async()=>{
 fixture=await roomFixture();const a=await fixture.connect('a');
 const ids=Array.from({length:12},()=>a.send('voice_join'));
 for(const id of ids)expect((await a.wait('voice_admission',id)).payload).toMatchObject({admitted:true,peers:[]});
 for(const payload of [{to:'x',description:{type:'offer',sdp:'x'.repeat(24577)}},{to:'x',candidate:{candidate:'x'.repeat(4097)}},{to:'x',candidate:{candidate:'ok',sdpMLineIndex:-1}},{to:'x',candidate:null,description:{type:'offer',sdp:'v=0'}}]){
  const id=a.send('voice_signal',payload);expect((await a.wait('error',id)).payload.code).toBe('VOICE_INVALID');
 }
 expect(a.socket.readyState).toBe(1);
});
