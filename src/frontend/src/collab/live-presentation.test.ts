import { describe, expect, it, vi } from 'vitest';
import { LivePresentation, presentationFrame } from './live-presentation';
import type { Control, TransportState } from './transport';

class Transport {
  state: TransportState={status:'synced',canEdit:true,pending:0,recovery:false,connectionId:'host'};
  controls=new Set<(value:Control)=>void>();sent:{type:string;payload:Record<string,unknown>}[]=[];
  subscribeControl=(listener:(value:Control)=>void)=>{this.controls.add(listener);return()=>this.controls.delete(listener)};
  listeners=new Set<()=>void>();subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>this.listeners.delete(listener)};getSnapshot=()=>this.state;
  sendControl=(type:string,payload:Record<string,unknown>)=>{this.sent.push({type,payload});return crypto.randomUUID()};
  emit(type:string,payload:Record<string,unknown>){for(const listener of this.controls)listener({type,requestId:'server',payload})}
  update(status:TransportState['status']){this.state={...this.state,status};for(const listener of this.listeners)listener();}
}
const session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const state=(overrides:Record<string,unknown>={})=>({sessionId:session,hostConnectionId:'host',hostPrincipalId:'user',revision:4,pending:[],admitted:[],frameId:null,spotlight:null,locked:false,reactions:[],...overrides});

describe('live presentation controller',()=>{
 const api=()=>({create:vi.fn(async()=>({sessionId:session,token:'host-token',expiresAt:'2099-01-01T00:00:00Z'})),createLink:vi.fn(),admit:vi.fn()});
 it('sends revision-bound host controls and ephemeral reactions',()=>{
  const transport=new Transport(),live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state());
  live.setFrame('frame-a');live.spotlight('guest');live.setLocked(true);live.remove('guest');live.transfer('editor');live.react('👏');
  expect(transport.sent.map(item=>item.payload)).toEqual([
   {sessionId:session,frameId:'frame-a',expectedRevision:'4'},
   {sessionId:session,connectionId:'guest',expectedRevision:'4'},
   {sessionId:session,locked:true,expectedRevision:'4'},
   {sessionId:session,connectionId:'guest',expectedRevision:'4'},
   {sessionId:session,connectionId:'editor',expectedRevision:'4'},
   {sessionId:session,emoji:'👏'},
  ]);
 });
 it('keeps Leave available while locked and expires reactions after three seconds',()=>{
  vi.useFakeTimers();const transport=new Transport(),live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state({locked:true}));
  transport.emit('presentation_reaction',{sessionId:session,id:'r',connectionId:'guest',emoji:'👍',expiresAt:Date.now()+3000});expect(live.getSnapshot().reactions).toHaveLength(1);
  live.leave();expect(transport.sent.at(-1)).toMatchObject({type:'presentation_leave',payload:{sessionId:session}});
  vi.advanceTimersByTime(3001);expect(live.getSnapshot().reactions).toHaveLength(0);vi.useRealTimers();
 });
 it('does not silently jump when the selected frame was removed',()=>{
  expect(presentationFrame([{id:'other',type:'frame'}] as never,'missing')).toEqual({kind:'missing',frameId:'missing'});
  expect(presentationFrame([{id:'frame-a',type:'frame'}] as never,'frame-a')).toMatchObject({kind:'frame',frameId:'frame-a'});
  expect(presentationFrame([],null)).toEqual({kind:'none'});
 });
 it('ends host controls when the host socket goes offline',()=>{
  const transport=new Transport(),live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state());transport.update('offline');
  expect(live.getSnapshot()).toMatchObject({status:'ended',reason:'HOST_OFFLINE'});
 });
 it('requires a fresh waiting-room join after an admitted audience disconnects',()=>{
  const transport=new Transport();transport.state.connectionId='guest';const live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state({admitted:[{connectionId:'guest',principalId:'presentation:g',displayName:'Guest'}]}));transport.update('offline');
  expect(live.getSnapshot()).toMatchObject({status:'ended',reason:'AUDIENCE_OFFLINE'});
 });
 it('keeps ordinary room collaborators as observers until host transfer',()=>{
  const transport=new Transport();transport.state.connectionId='editor';const live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state());
  expect(live.getSnapshot().status).toBe('observer');expect(live.react('👍')).toBe(false);expect(live.leave()).toBe(false);
 });
 it('ends an observer view when its room socket goes offline',()=>{
  const transport=new Transport();transport.state.connectionId='editor';const live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state());transport.update('offline');
  expect(live.getSnapshot()).toMatchObject({status:'ended',reason:'OBSERVER_OFFLINE'});
 });
 it('exposes retry after the socket drops while a host is starting',async()=>{
  const transport=new Transport(),live=new LivePresentation(transport,api());live.start();await live.beginHost();transport.update('offline');
  expect(live.getSnapshot()).toMatchObject({status:'error',reason:'START_OFFLINE'});
 });
 it('exposes a runtime rejection while a host is starting',async()=>{
  const transport=new Transport(),live=new LivePresentation(transport,api());live.start();await live.beginHost();transport.emit('error',{code:'PRESENTATION_ACTIVE',message:'Another host is already presenting'});
  expect(live.getSnapshot()).toMatchObject({status:'error',error:'Another host is already presenting'});
 });
 it('exposes retry when the runtime never acknowledges a host start',async()=>{
  vi.useFakeTimers();const transport=new Transport(),live=new LivePresentation(transport,api());live.start();await live.beginHost();vi.advanceTimersByTime(10_001);
  expect(live.getSnapshot()).toMatchObject({status:'error',reason:'START_TIMEOUT'});vi.useRealTimers();
 });
 it('ignores lower revisions and cannot resurrect a terminal session',()=>{
  const transport=new Transport(),live=new LivePresentation(transport);live.start();transport.emit('presentation_state',state({revision:5,frameId:'current'}));transport.emit('presentation_state',state({revision:4,frameId:'stale'}));
  expect(live.getSnapshot()).toMatchObject({revision:5,frameId:'current'});transport.emit('presentation_stop',{sessionId:session,reason:'HOST_ENDED'});transport.emit('presentation_state',state({revision:6,frameId:'late'}));
  expect(live.getSnapshot()).toMatchObject({status:'ended',reason:'HOST_ENDED',revision:5,frameId:'current'});
 });
 it('turns a rejected pending authentication into a terminal rejoin state',()=>{
  const transport=new Transport();transport.state.connectionId=undefined;const live=new LivePresentation(transport);live.start();transport.emit('error',{code:'PRESENTATION_ENDED',message:'Presentation has ended'});
  expect(live.getSnapshot()).toMatchObject({status:'error',reason:'PRESENTATION_ENDED',error:'Presentation has ended'});
 });
});
