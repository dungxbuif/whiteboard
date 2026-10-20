import {afterEach,expect,it,vi} from 'vitest';
import {encodeFrame} from '@collab/contracts';
import {CollabError} from './auth.js';
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

it('default-disabled screen policy denies publisher reservation without stopping drawing',async()=>{
 fixture=await roomFixture();fixture.hooks.featureEnabled=false;const publisher=await fixture.connect('publisher');const request=publisher.send('screen_request');
 expect((await publisher.wait('error',request)).payload.code).toBe('SCREEN_FORBIDDEN');expect(publisher.messages.some(message=>message.type==='screen_admission'&&message.requestId===request)).toBe(false);
 publisher.socket.send(encodeFrame({channel:3,updateId:'drawing-survives-feature-denial',payload:new Uint8Array([0,0])}));expect((await publisher.wait('ack','drawing-survives-feature-denial')).payload.sequence).toBe('1');
 expect(fixture.hooks.featureChecks).toEqual(['screen:publish']);
});

it('enabled-to-disabled policy stops publisher before subsequent signal routing',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher'),viewer=await fixture.connect('viewer');
 const admission=await publisher.wait('screen_admission',publisher.send('screen_request'));await publisher.wait('screen_admission',publisher.send('screen_request',{sessionId:admission.payload.sessionId}));await viewer.wait('screen_admission',viewer.send('screen_request'));
 fixture.hooks.featureEnabled=false;const signal=publisher.send('screen_signal',{to:viewer.connectionId,sessionId:admission.payload.sessionId,description:{type:'offer',sdp:'v=0'}});
 expect((await publisher.wait('error',signal)).payload.code).toBe('SCREEN_FORBIDDEN');expect((await viewer.wait('screen_stop')).payload).toMatchObject({sessionId:admission.payload.sessionId,reason:'FEATURE_REVOKED'});expect(viewer.messages.some(message=>message.type==='screen_signal')).toBe(false);
 expect(fixture.hooks.featureChecks).toEqual(['screen:publish','screen:publish','screen:publish','screen:publish']);
});

it('receiver join rechecks the active publisher policy before admission',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher'),viewer=await fixture.connect('viewer');
 const admission=await publisher.wait('screen_admission',publisher.send('screen_request'));await publisher.wait('screen_admission',publisher.send('screen_request',{sessionId:admission.payload.sessionId}));const publisherAdmissions=publisher.messages.filter(message=>message.type==='screen_admission').length;
 fixture.hooks.featureEnabled=false;const request=viewer.send('screen_request');expect((await viewer.wait('error',request)).payload.code).toBe('SCREEN_FORBIDDEN');expect((await viewer.wait('screen_stop')).payload).toMatchObject({sessionId:admission.payload.sessionId,reason:'FEATURE_REVOKED'});expect(publisher.messages.filter(message=>message.type==='screen_admission')).toHaveLength(publisherAdmissions);
 expect(fixture.hooks.featurePrincipals.at(-1)).toBe(publisher.principalId);
 viewer.socket.send(encodeFrame({channel:3,updateId:'drawing-after-receiver-denial',payload:new Uint8Array([0,0])}));expect((await viewer.wait('ack','drawing-after-receiver-denial')).payload.sequence).toBe('1');expect((await viewer.wait('voice_admission',viewer.send('voice_join'))).payload.admitted).toBe(true);
});

it('receiver signal fails closed when active publisher authority is unavailable',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher'),viewer=await fixture.connect('viewer');
 const admission=await publisher.wait('screen_admission',publisher.send('screen_request'));await publisher.wait('screen_admission',publisher.send('screen_request',{sessionId:admission.payload.sessionId}));await viewer.wait('screen_admission',viewer.send('screen_request'));
 fixture.hooks.featureError=new CollabError('UNAVAILABLE','authority unavailable',503,true);const signal=viewer.send('screen_signal',{to:publisher.connectionId,sessionId:admission.payload.sessionId,candidate:{candidate:'candidate'}});
 expect((await viewer.wait('error',signal)).payload).toMatchObject({code:'SCREEN_AUTHORITY_UNAVAILABLE',retryable:true});expect((await publisher.wait('screen_stop')).payload).toMatchObject({sessionId:admission.payload.sessionId,reason:'FEATURE_REVOKED'});expect(publisher.messages.some(message=>message.type==='screen_signal')).toBe(false);
 expect(fixture.hooks.featurePrincipals.at(-1)).toBe(publisher.principalId);
});

it('receiver stop rechecks publisher policy before forwarding stop control',async()=>{
 fixture=await roomFixture();const publisher=await fixture.connect('publisher'),viewer=await fixture.connect('viewer');
 const admission=await publisher.wait('screen_admission',publisher.send('screen_request'));await publisher.wait('screen_admission',publisher.send('screen_request',{sessionId:admission.payload.sessionId}));await viewer.wait('screen_admission',viewer.send('screen_request'));
 fixture.hooks.featureEnabled=false;const request=viewer.send('screen_stop',{sessionId:admission.payload.sessionId});expect((await viewer.wait('error',request)).payload.code).toBe('SCREEN_FORBIDDEN');expect((await publisher.wait('screen_stop')).payload).toMatchObject({sessionId:admission.payload.sessionId,reason:'FEATURE_REVOKED'});expect(publisher.messages.some(message=>message.type==='screen_stop'&&message.payload.connectionId===viewer.connectionId)).toBe(false);
 expect(fixture.hooks.featurePrincipals.at(-1)).toBe(publisher.principalId);
});
