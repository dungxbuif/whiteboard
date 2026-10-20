import {describe,it,expect} from 'vitest';
import {randomUUID} from 'node:crypto';
import {once} from 'node:events';
import {encodeFrame} from '@collab/contracts';
import {createEncoder,writeVarUint,writeVarUint8Array,toUint8Array} from 'lib0/encoding';
import {roomFixture} from './test-fixtures/presence.js';

describe('presentation room isolation',()=>{
 it('sends a waiting client only its status and rejects scene/media traffic',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});
   expect(waiting.messages.map(m=>m.type)).toEqual(['presentation_waiting']);
   const hostQueue=await host.wait('presentation_state','server');expect(hostQueue.payload).toMatchObject({sessionId,revision:1,pending:[{connectionId:waiting.connectionId,displayName:'Guest'}]});host.messages.splice(host.messages.indexOf(hostQueue),1);
   expect(()=>fixture.rooms.resolvePresentationPending({workspaceId:'workspace',sceneId:'scene'},sessionId,'other-editor',waiting.connectionId,1)).toThrow(/unavailable/i);
   waiting.socket.send(encodeFrame({channel:1,updateId:'pending-sync',payload:new Uint8Array()}));await waiting.wait('error','pending-sync');
   expect(waiting.messages.some(m=>['ready','sync_chunk','roster','voice_admission','screen_admission'].includes(m.type))).toBe(false);
   waiting.socket.close();
   const changed=await host.wait('presentation_state','server');expect(changed.payload.pending).toEqual([]);expect(changed.payload.revision).toBe(2);
  }finally{await fixture.close();}
 });
 it('publishes queue removal when a waiting guest leaves explicitly',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('host'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'host',sessionId})});await host.wait('presentation_state',started);
   const waiting=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:guest',displayName:'Guest',exp:Math.floor(Date.now()/1000)+300});await host.wait('presentation_state','server');
   const leave=waiting.send('presentation_leave',{sessionId});expect((await waiting.wait('presentation_stop',leave)).payload.reason).toBe('LEFT');
   expect((await host.wait('presentation_state',leave)).payload).toMatchObject({sessionId,revision:2,pending:[]});
  }finally{await fixture.close();}
 });
 it('does not start after the exact host session disconnects during authority gating',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('host'),sessionId=randomUUID();let entered!:()=>void,release!:()=>void;const gated=new Promise<void>(resolve=>entered=resolve),gate=new Promise<void>(resolve=>release=resolve);
   fixture.hooks.beforeFeature=async principalId=>{if(principalId==='host'){entered();await gate;}};host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'host',sessionId})});await gated;
   const closed=once(host.socket,'close');host.socket.close();await closed;release();await new Promise(resolve=>setTimeout(resolve,20));
   expect(()=>fixture.rooms.resolvePresentationHost({workspaceId:'workspace',sceneId:'scene'},sessionId,'host')).toThrow(/not active/i);
  }finally{await fixture.close();}
 });
 it('does not publish stale state after a session ends during recipient authority refresh',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('host'),observer=await fixture.connect('observer'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'host',sessionId})});await host.wait('presentation_state',started);await observer.wait('presentation_state',started);
   let entered!:()=>void,release!:()=>void;const gated=new Promise<void>(resolve=>entered=resolve),gate=new Promise<void>(resolve=>release=resolve);fixture.hooks.beforeRead=async principalId=>{if(principalId==='host'){entered();await gate;}};
   const frame=host.send('presentation_set_frame',{sessionId,frameId:'frame-a',expectedRevision:'1'});await gated;const closed=once(host.socket,'close');host.socket.close();await closed;expect((await observer.wait('presentation_stop','server')).payload.reason).toBe('HOST_DISCONNECTED');release();await new Promise(resolve=>setTimeout(resolve,20));
   expect(observer.messages.some(message=>message.type==='presentation_state'&&message.requestId===frame)).toBe(false);
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
 it('ends the caller host when authority disappears during a control',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);
   fixture.hooks.featureEnabled=false;
   const frame=host.send('presentation_set_frame',{sessionId,frameId:'frame-a',expectedRevision:'0'});
   expect((await host.wait('presentation_stop','server')).payload).toEqual({sessionId,reason:'AUTHORITY_CHANGED'});
   expect((await host.wait('error',frame)).payload.code).toBe('PRESENTATION_FORBIDDEN');
  }finally{await fixture.close();}
 });
 it('hydrates a collaborator that joins while a presentation is active',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('host'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'host',sessionId})});await host.wait('presentation_state',started);
   const late=await fixture.connect('late-editor');
   expect((await late.wait('presentation_state','server')).payload).toMatchObject({sessionId,hostConnectionId:host.connectionId,pending:[]});
  }finally{await fixture.close();}
 });
 it('serializes host state, hides pending guests from audiences, and transfers only to current editors',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('host-user'),editor=await fixture.connect('editor-user'),sessionId=randomUUID();
   fixture.hooks.accessByPrincipal.set('viewer-user','view');const viewer=await fixture.connect('viewer-user');
   const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'host-user',sessionId})});await host.wait('presentation_state',started);
   const first=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:first',displayName:'First',exp:Math.floor(Date.now()/1000)+300});
   const second=await fixture.connectPresentation({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:second',displayName:'Second',exp:Math.floor(Date.now()/1000)+300});
   const admitted=host.send('presentation_admit',{connectionId:first.connectionId,expectedRevision:'1',token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',sessionId,principalId:'presentation:first',principalType:'public',purpose:'presentation-audience',grantVersion:'1',access:'view',aclVersion:'1',exp:Math.floor(Date.now()/1000)+600})});
   await first.wait('ready',admitted);const audienceState=await first.wait('presentation_state',admitted);expect(audienceState.payload.pending).toEqual([]);
   expect(second.messages.map(message=>message.type)).toEqual(['presentation_waiting']);
   const stale=host.send('presentation_set_frame',{sessionId,frameId:'frame-a',expectedRevision:'1'});expect((await host.wait('error',stale)).payload.code).toBe('PRESENTATION_CONFLICT');
   const frame=host.send('presentation_set_frame',{sessionId,frameId:'frame-a',expectedRevision:'2'});expect((await first.wait('presentation_state',frame)).payload).toMatchObject({frameId:'frame-a',revision:3,pending:[]});
   const denied=host.send('presentation_transfer',{sessionId,connectionId:viewer.connectionId,expectedRevision:'3'});expect((await host.wait('error',denied)).payload.code).toBe('PRESENTATION_TARGET');
   fixture.hooks.beforeFeature=async principalId=>{if(principalId==='editor-user')throw new Error('forbidden target');};
   const rejected=host.send('presentation_transfer',{sessionId,connectionId:editor.connectionId,expectedRevision:'3'});expect((await host.wait('error',rejected)).payload.code).toBe('PRESENTATION_TARGET');
   fixture.hooks.beforeFeature=undefined;const stillHost=host.send('presentation_lock',{sessionId,locked:true,expectedRevision:'3'});expect((await host.wait('presentation_state',stillHost)).payload).toMatchObject({hostConnectionId:host.connectionId,revision:4,locked:true});
   fixture.hooks.beforeFeature=async principalId=>{if(principalId==='editor-user')fixture.hooks.accessByPrincipal.set('editor-user','view');};
   const raced=host.send('presentation_transfer',{sessionId,connectionId:editor.connectionId,expectedRevision:'4'});expect((await host.wait('error',raced)).payload.code).toBe('PRESENTATION_TARGET');
   fixture.hooks.beforeFeature=undefined;fixture.hooks.accessByPrincipal.set('editor-user','edit');
   const transfer=host.send('presentation_transfer',{sessionId,connectionId:editor.connectionId,expectedRevision:'4'});expect((await editor.wait('presentation_state',transfer)).payload).toMatchObject({hostConnectionId:editor.connectionId,hostPrincipalId:'editor-user',revision:5});
  }finally{await fixture.close();}
 });
 it('keeps the host ended after ending the live session',async()=>{
  const fixture=await roomFixture();try{
   const host=await fixture.connect('user'),observer=await fixture.connect('editor'),sessionId=randomUUID();const started=host.send('presentation_start',{token:JSON.stringify({workspaceId:'workspace',sceneId:'scene',principalId:'user',sessionId})});await host.wait('presentation_state',started);await observer.wait('presentation_state',started);
   const leave=host.send('presentation_leave',{sessionId});const stopped=await host.wait('presentation_stop',leave);expect(stopped.payload).toEqual({sessionId,reason:'HOST_ENDED'});
   expect((await observer.wait('presentation_stop','server')).payload).toEqual({sessionId,reason:'HOST_ENDED'});
  }finally{await fixture.close();}
 });
});
