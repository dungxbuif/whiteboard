import {describe,it,expect} from 'vitest';
import {randomUUID} from 'node:crypto';
import {encodeFrame} from '@collab/contracts';
import {createEncoder,writeVarUint,writeVarUint8Array,toUint8Array} from 'lib0/encoding';
import {roomFixture} from './test-fixtures/presence.js';

describe('presentation room isolation',()=>{
 it('sends a waiting client only its status and rejects scene/media traffic',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});
   expect(waiting.messages.map(m=>m.type)).toEqual(['presentation_waiting']);
   expect(()=>fixture.rooms.resolvePresentationPending({workspaceId:'workspace',sceneId:'scene'},sessionId,'other-editor',waiting.connectionId,1)).toThrow(/unavailable/i);
   waiting.socket.send(encodeFrame({channel:1,updateId:'pending-sync',payload:new Uint8Array()}));await waiting.wait('error','pending-sync');
   expect(waiting.messages.some(m=>['ready','sync_chunk','roster','voice_admission','screen_admission'].includes(m.type))).toBe(false);
   waiting.socket.close();
   const changed=await host.wait('presentation_state','server');expect(changed.payload.pending).toEqual([]);expect(changed.payload.revision).toBe(2);
  }finally{await fixture.close();}
 });
 it('refreshes an admitted audience through presentation authority',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});
   const audienceToken=JSON.stringify({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',principalType:'public',purpose:'presentation-audience',grantVersion:'1',access:'view',aclVersion:'1',exp:Math.floor(Date.now()/1000)+600});
   const admitted=host.send('presentation_admit',{connectionId:waiting.connectionId,expectedRevision:'1',token:audienceToken});
   const credential=await waiting.wait('presentation_admitted',admitted);expect(credential.payload).toMatchObject({sessionId,token:audienceToken});expect(host.messages.some(message=>JSON.stringify(message).includes(audienceToken))).toBe(false);
   await waiting.wait('ready',admitted);await waiting.wait('sync_chunk',admitted);
   const presence=waiting.send('presence',{displayName:'Guest'});await waiting.wait('roster',presence);
   const encoder=createEncoder();writeVarUint(encoder,0);writeVarUint8Array(encoder,new Uint8Array([0]));waiting.socket.send(encodeFrame({channel:1,updateId:'audience-resync',payload:toUint8Array(encoder)}));await waiting.wait('sync_chunk','audience-resync');
   expect(fixture.hooks.readKinds.filter(kind=>kind==='presentation:presentation:guest').length).toBeGreaterThanOrEqual(2);
   expect(fixture.hooks.readKinds).not.toContain('room:presentation:guest');
   host.socket.close();const stopped=await waiting.wait('presentation_stop','server');expect(stopped.payload.reason).toBe('HOST_DISCONNECTED');
  }finally{await fixture.close();}
 });
 it('rejects an audience grant minted for a stale session revision',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});
   const id=host.send('presentation_admit',{connectionId:waiting.connectionId,expectedRevision:'1',token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',principalType:'public',purpose:'presentation-audience',grantVersion:'2',access:'view',aclVersion:'1',exp:Math.floor(Date.now()/1000)+600})});
   const error=await host.wait('error',id);expect(error.payload.code).toBe('PRESENTATION_CONFLICT');expect(waiting.messages.some(message=>message.type==='ready'||message.type==='presentation_admitted')).toBe(false);
  }finally{await fixture.close();}
 });
 it('ends active audience state on feature invalidation',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});
   const id=host.send('presentation_admit',{connectionId:waiting.connectionId,expectedRevision:'1',token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',principalType:'public',purpose:'presentation-audience',grantVersion:'1',access:'view',aclVersion:'1',exp:Math.floor(Date.now()/1000)+600})});await waiting.wait('ready',id);
   fixture.rooms.featureChanged('workspace','scene','1');const stopped=await waiting.wait('presentation_stop','server');expect(stopped.payload.reason).toBe('AUTHORITY_CHANGED');expect(()=>fixture.rooms.resolvePresentationAudience({workspaceId:'workspace',sceneId:'scene'},sessionId,'presentation:guest')).toThrow();
  }finally{await fixture.close();}
 });
});
