import {describe,it,expect} from 'vitest';
import {randomUUID} from 'node:crypto';
import {encodeFrame} from '@collab/contracts';
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
   const admitted=host.send('presentation_admit',{connectionId:waiting.connectionId,expectedRevision:'1',token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',principalType:'public',purpose:'presentation-audience',access:'view',aclVersion:'1',exp:Math.floor(Date.now()/1000)+600})});
   await waiting.wait('ready',admitted);await waiting.wait('sync_chunk',admitted);
   const presence=waiting.send('presence',{displayName:'Guest'});await waiting.wait('roster',presence);
   expect(fixture.hooks.readKinds.filter(kind=>kind==='presentation:presentation:guest').length).toBeGreaterThanOrEqual(2);
   expect(fixture.hooks.readKinds).not.toContain('room:presentation:guest');
   host.socket.close();const stopped=await waiting.wait('presentation_stop','server');expect(stopped.payload.reason).toBe('HOST_DISCONNECTED');
  }finally{await fixture.close();}
 });
});
