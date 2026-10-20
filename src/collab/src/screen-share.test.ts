import {afterEach,expect,it,vi} from 'vitest';
import {encodeFrame} from '@collab/contracts';
import {MediaAdmission} from './screen-share.js';
import {roomFixture} from './test-fixtures/presence.js';

let fixture:Awaited<ReturnType<typeof roomFixture>>|undefined;
afterEach(async()=>{vi.useRealTimers();await fixture?.close();fixture=undefined;});

it('screenIndependentOfMic',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher');
 const request=publisher.send('screen_request');const admission=await publisher.wait('screen_admission',request);
 expect(admission.payload).toMatchObject({admitted:true,publisher:publisher.connectionId,receiving:false});
 const activated=publisher.send('screen_request',{sessionId:admission.payload.sessionId});expect((await publisher.wait('screen_admission',activated)).payload).toMatchObject({admitted:true,publisher:publisher.connectionId,sessionId:admission.payload.sessionId});
 expect(publisher.messages.some(message=>message.type==='voice_admission')).toBe(false);
});

it('voiceScreenCountOnceAtTen',()=>{
 const media=new MediaAdmission();const room='workspace:scene';
 for(let index=0;index<10;index++)expect(media.join(room,`voice-${index}`,'voice').admitted).toBe(true);
 expect(media.join(room,'voice-0','screen')).toMatchObject({admitted:true,publisher:'voice-0',receiving:false});
 expect(media.join(room,'eleventh','screen')).toMatchObject({admitted:false,reason:'MEDIA_FULL'});
 media.close();
});

it('eleventhMediaDeniedButDrawingWorks',async()=>{
 fixture=await roomFixture();const clients=[];for(let index=0;index<11;index++)clients.push(await fixture.connect(`person-${index}`));
 for(const client of clients.slice(0,10))expect((await client.wait('voice_admission',client.send('voice_join'))).payload.admitted).toBe(true);
 const last=clients[10],request=last.send('screen_request');expect((await last.wait('screen_admission',request)).payload).toMatchObject({admitted:false,reason:'MEDIA_FULL'});
 last.socket.send(encodeFrame({channel:3,updateId:'still-drawing',payload:new Uint8Array([0,0])}));
 expect((await last.wait('ack','still-drawing')).payload.sequence).toBe('1');
});

it('publisherLeaseExpires',()=>{
 vi.useFakeTimers();const expired=vi.fn();const media=new MediaAdmission(expired);const admission=media.join('workspace:scene','publisher','screen');
 expect(admission.admitted).toBe(true);vi.advanceTimersByTime(30_001);
 expect(media.publisher('workspace:scene')).toBeUndefined();expect(expired).toHaveBeenCalledWith(expect.objectContaining({publisher:'publisher',sessionId:admission.sessionId}));
 const active=media.join('workspace:scene','active-publisher','screen');expect(media.activate('workspace:scene','active-publisher',active.sessionId!)).toBe(true);vi.advanceTimersByTime(30_001);expect(media.publisher('workspace:scene')).toBe('active-publisher');media.close();
});

it('signalCannotCrossRoom',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher','scene'),other=await fixture.connect('other','another');
 const request=publisher.send('screen_request'),admission=await publisher.wait('screen_admission',request);
 await other.wait('screen_admission',other.send('screen_request'));
 const signal=publisher.send('screen_signal',{to:other.connectionId,sessionId:admission.payload.sessionId,description:{type:'offer',sdp:'v=0'}});
 expect((await publisher.wait('error',signal)).payload.code).toBe('SCREEN_TARGET');expect(other.messages.some(message=>message.type==='screen_signal')).toBe(false);
});

it('publisher authority loss ends screen media before routing',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher'),viewer=await fixture.connect('viewer');
 const admission=await publisher.wait('screen_admission',publisher.send('screen_request'));await viewer.wait('screen_admission',viewer.send('screen_request'));
 fixture.hooks.access='view';const signal=publisher.send('screen_signal',{to:viewer.connectionId,sessionId:admission.payload.sessionId,description:{type:'offer',sdp:'v=0'}});
 expect((await publisher.wait('error',signal)).payload.code).toBe('SCREEN_FORBIDDEN');expect((await publisher.wait('screen_stop')).payload.sessionId).toBe(admission.payload.sessionId);expect(viewer.messages.some(message=>message.type==='screen_signal')).toBe(false);
});
