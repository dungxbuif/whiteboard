import WebSocket from 'ws';
import {randomUUID} from 'node:crypto';
import {Awareness,fields,parseAwareness,parseViewport,roomKey} from './awareness.js';
import {Voice,parseSignal} from './voice.js';
import {IngressLimit} from './ingress.js';
import * as Y from 'yjs';
import {createDecoder,readVarUint,readVarUint8Array,hasContent} from 'lib0/decoding';
import {MAX_FRAME_BYTES,decodeFrame,encodeFrame,type SceneRef,type MutationReceipt} from '@collab/contracts';
import {CollabError,authorizeSync,requestID,type Principal,type Decision} from './auth.js';
import {MAX_SNAPSHOT_BYTES,SnapshotHistoryLimitError} from './document.js';
import type {Persistence,SceneState} from './persistence.js';
import {RateLimit} from './rate-limit.js';

export const HEARTBEAT_INTERVAL_MS=25_000;
const EPHEMERAL_CONTROLS=['presence','follow','viewport','voice_join','voice_leave','voice_mute','voice_signal'];
const MAX_CONTROL_BYTES=32768;
const AUTH_PER_MINUTE=60,EPHEMERAL_PER_MINUTE=120,AWARENESS_PER_SECOND=30;

function invalidInput<T>(parse:()=>T):T {try{return parse();}catch(error){if(error instanceof CollabError)throw error;throw new CollabError('INVALID_UPDATE','Malformed transport input',400);}}
type Session={connectionId:string;alive:boolean;socket:WebSocket;ref?:SceneRef;principal?:Principal;claims?:Decision;access?:string;expires?:NodeJS.Timeout;authTimer:NodeJS.Timeout;pending:number;chain:Promise<void>;ip:string;invalid:boolean};
type Fanout={ref:SceneRef;updates:Uint8Array[];bytes:number;overflow:boolean};
export class Rooms {
 private sessions=new Set<Session>();private rates=new RateLimit();private ingress=new IngressLimit();private tasks=new Set<Promise<void>>();
 private fanouts=new Map<string,Fanout>();
 private awareness=new Awareness();private voice=new Voice();private heartbeat:NodeJS.Timeout;private closing=false;
 constructor(private store:Persistence,private options:{updatesPerMinute:number;maxBufferedBytes:number;heartbeatIntervalMs?:number}){
  const interval=options.heartbeatIntervalMs??HEARTBEAT_INTERVAL_MS;
  if(!Number.isInteger(interval)||interval<10||interval>HEARTBEAT_INTERVAL_MS)throw new Error('Heartbeat interval must be 10..25000ms');
  store.on('committed',this.committed);
  this.heartbeat=setInterval(()=>{for(const session of this.sessions){if(session.invalid)continue;if(!session.alive){this.detach(session);session.socket.terminate();continue;}session.alive=false;if(session.socket.readyState===WebSocket.OPEN)session.socket.ping();}},interval);this.heartbeat.unref();
 }
 attach(socket:WebSocket,ip:string){
  const session:Session={connectionId:randomUUID(),alive:true,socket,pending:0,chain:Promise.resolve(),ip,invalid:false,authTimer:setTimeout(()=>socket.close(4401,'Authentication required'),5000)};
  this.sessions.add(session);socket.on('error',()=>{});
  socket.on('pong',()=>{session.alive=true;});
  socket.on('close',()=>this.detach(session));
  socket.on('message',(data,binary)=>{
   if(session.invalid)return;
   const bytes=Buffer.isBuffer(data)?new Uint8Array(data):new Uint8Array(data as ArrayBuffer);
   try{this.ingress.take(ip,bytes.byteLength);}catch{this.invalidate(session,'INGRESS_RATE_LIMITED');return;}
   if(++session.pending>32){this.invalidate(session,'BACKPRESSURE');return;}
   session.chain=session.chain.then(async()=>{if(session.invalid)return;let id='unknown';try{
    if(binary){const frame=invalidInput(()=>decodeFrame(bytes));id=frame.updateId;requestID(id);await this.binary(session,frame.channel,id,frame.payload);}
    else{
     if(bytes.length>MAX_CONTROL_BYTES)throw new CollabError('INVALID_UPDATE','Control too large',400);
     const control=invalidInput(()=>JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes)));
     if(!control||typeof control!=='object'||Array.isArray(control)||Object.keys(control).some(k=>!['type','requestId','payload'].includes(k)))throw new CollabError('INVALID_UPDATE','Invalid control',400);
     id=control.requestId;requestID(id);
     if(['auth','refresh'].includes(control.type)){
      this.rates.take('auth:'+ip,AUTH_PER_MINUTE);
      if(!control.payload||Object.keys(control.payload).some(k=>k!=='token'))throw new CollabError('INVALID_UPDATE','Invalid auth control',400);
      await this.authenticate(session,id,control.payload.token,control.type);
     }else if(EPHEMERAL_CONTROLS.includes(control.type))await this.ephemeral(session,id,control.type,control.payload);
     else throw new CollabError('INVALID_UPDATE','Unknown control',400);
    }
   }catch(error){await this.error(session,id,error);}finally{session.pending--;}});
   this.track(session.chain);
  });
 }
 private async authenticate(session:Session,id:string,token:string,type:string){
  if(type==='auth'&&session.principal||type==='refresh'&&!session.principal)throw new CollabError('UNAUTHENTICATED','Invalid authentication transition',401);
  const claims=await this.store.authority.verifyRoom(token),ref={workspaceId:claims.workspaceId,sceneId:claims.sceneId};
  if(session.ref&&(session.ref.workspaceId!==ref.workspaceId||session.ref.sceneId!==ref.sceneId||session.principal!.principalId!==claims.principalId||session.principal!.principalType!==claims.principalType))throw new CollabError('FORBIDDEN','Refresh cannot change room or principal');
  const principal={roomToken:token,principalId:claims.principalId,principalType:claims.principalType};
  const state=await this.store.read(ref,principal);
  try{
   if(session.invalid)return;
   session.ref=ref;session.principal=principal;session.claims=claims;session.access=state.access;
   clearTimeout(session.authTimer);clearTimeout(session.expires);session.expires=setTimeout(()=>this.invalidate(session,'TOKEN_EXPIRED'),Math.max(0,claims.exp*1000-Date.now()));
   this.awareness.register(ref,{connectionId:session.connectionId,principalId:principal.principalId,principalType:principal.principalType,access:state.access});
   await this.control(session,'ready',id,{...ref,connectionId:session.connectionId,principalId:principal.principalId,access:state.access,documentId:state.documentId,sequence:state.sequence,stateVector:state.stateVector});
   await this.sync(session,id,state);
   await this.roster(ref,id);
  }finally{state.doc.destroy();}
 }
 private async binary(session:Session,channel:number,id:string,payload:Uint8Array){
  if(!session.principal||!session.ref||!session.claims||Date.now()>=session.claims.exp*1000)throw new CollabError('UNAUTHENTICATED','Fresh room credential required',401);
  let update=payload;
  if(channel===1){
   const {subtype,bytes}=invalidInput(()=>{const decoder=createDecoder(payload);const subtype=readVarUint(decoder);const bytes=readVarUint8Array(decoder);if(hasContent(decoder))throw new CollabError('INVALID_UPDATE','Trailing sync bytes',400);return {subtype,bytes};});authorizeSync(session.access!,subtype);
   if(subtype===0){this.rates.take('read:'+session.principal.principalId,120);if(bytes.length>65536)throw new CollabError('INVALID_UPDATE','State vector too large',400);const state=await this.store.read(session.ref,session.principal);try{session.access=state.access;await this.sync(session,id,state,bytes);}finally{state.doc.destroy();}return;}
   update=bytes;
  }
  if(channel===2){
   try{this.rates.take('awareness:'+session.connectionId,AWARENESS_PER_SECOND,1000);}catch{throw new CollabError('PRESENCE_RATE_LIMITED','Awareness rate exceeded',429);}
   if(payload.length>8192)throw new CollabError('INVALID_PRESENCE','Awareness too large',400);
   const value=parseAwareness(invalidInput(()=>JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(payload))));
   await this.fresh(session);
   await this.broadcast(session.ref,'awareness',id,{...value,connectionId:session.connectionId,principalId:session.principal.principalId,principalType:session.principal.principalType},other=>other!==session);return;
  }
  if(session.access!=='edit')throw new CollabError('FORBIDDEN');
  this.rates.take('principal:'+session.principal.principalType+':'+session.principal.principalId,this.options.updatesPerMinute);
  const receipt=await this.store.appendUpdate(session.ref,session.principal,id,update);
  await this.control(session,'ack',id,receipt);
 }
 private async fresh(session:Session){
  if(session.invalid||!session.principal||!session.ref||!session.claims||Date.now()>=session.claims.exp*1000)throw new CollabError('UNAUTHENTICATED','Fresh room credential required',401);
  const state=await this.store.read(session.ref,session.principal);
  try{if(session.invalid)throw new CollabError('UNAUTHENTICATED');session.access=state.access;this.awareness.register(session.ref,{connectionId:session.connectionId,principalId:session.principal.principalId,principalType:session.principal.principalType,access:state.access});}finally{state.doc.destroy();}
 }
 private matching(ref:SceneRef){return [...this.sessions].filter(s=>!s.invalid&&s.ref&&roomKey(s.ref)===roomKey(ref));}
 private async recipient(session:Session){try{await this.fresh(session);return true;}catch(error){this.invalidate(session,error instanceof CollabError&&['UNAUTHENTICATED','FORBIDDEN','STALE_AUTHORITY'].includes(error.code)?'AUTHORITY_CHANGED':'UNAVAILABLE');return false;}}
 private async broadcast(ref:SceneRef,type:string,id:string,payload:unknown,filter:(s:Session)=>boolean=()=>true){
  for(const other of this.matching(ref))if(filter(other)&&await this.recipient(other))await this.control(other,type,id,payload);
 }
 private async roster(ref:SceneRef,id='server'){
  const recipients=this.matching(ref);await Promise.all(recipients.map(s=>this.recipient(s)));
  for(const session of recipients){const participants=this.awareness.list(ref).map(p=>({...p,...this.voice.state(p.connectionId)}));await this.control(session,'roster',id,{participants});}
 }
 private async ephemeral(session:Session,id:string,type:string,payload:unknown){
  const voiceControl=type.startsWith('voice_');
  try{this.rates.take((type==='viewport'?'awareness:':'control:')+session.connectionId,type==='viewport'?AWARENESS_PER_SECOND:EPHEMERAL_PER_MINUTE,type==='viewport'?1000:60_000);}catch{throw new CollabError(voiceControl?'VOICE_RATE_LIMITED':'PRESENCE_RATE_LIMITED','Ephemeral control rate exceeded',429);}
  await this.fresh(session);const ref=session.ref!;
  if(type==='presence'){this.awareness.update(session.connectionId,payload);await this.roster(ref,id);return;}
  if(type==='follow'){
   const input=fields(payload,['targetConnectionId'],'INVALID_FOLLOW');
   if(input.targetConnectionId!==null){const target=this.matching(ref).find(s=>s.connectionId===input.targetConnectionId);if(!target||!await this.recipient(target))throw new CollabError('INVALID_FOLLOW','Target is not available',400);}
   if(session.invalid)return;this.awareness.follow(session.connectionId,input.targetConnectionId);
   await this.control(session,'follow',id,{connectionId:session.connectionId,targetConnectionId:input.targetConnectionId});await this.roster(ref,id);return;
  }
  if(type==='viewport'){
   const viewport=parseViewport(payload);const followers=new Set(this.awareness.list(ref).filter(p=>p.following===session.connectionId).map(p=>p.connectionId));
   await this.broadcast(ref,'viewport',id,{...viewport,connectionId:session.connectionId},other=>followers.has(other.connectionId));return;
  }
  if(type==='voice_join'){
   fields(payload,[],'VOICE_INVALID');const admission=this.voice.joinVoice(roomKey(ref),session.connectionId);
   await this.control(session,'voice_admission',id,admission);if(admission.admitted)await this.roster(ref,id);return;
  }
  if(type==='voice_leave'){
   fields(payload,[],'VOICE_INVALID');this.voice.leaveVoice(session.connectionId);await this.control(session,'voice_left',id,{connectionId:session.connectionId});await this.roster(ref,id);return;
  }
  if(type==='voice_mute'){
   const input=fields(payload,['muted'],'VOICE_INVALID');this.voice.mute(session.connectionId,input.muted);await this.roster(ref,id);return;
  }
  const signal=parseSignal(payload);const target=this.matching(ref).find(s=>s.connectionId===signal.to);
  if(!target||!this.voice.canSignal(roomKey(ref),session.connectionId,signal.to)||!await this.recipient(target))throw new CollabError('VOICE_TARGET','Voice peer is not available',400);
  // Admission can end while the authority request is in flight.
  if(session.invalid||target.invalid||!this.voice.canSignal(roomKey(ref),session.connectionId,signal.to))throw new CollabError('VOICE_TARGET','Voice peer is not available',400);
  await this.control(target,'voice_signal',id,{...signal,from:session.connectionId});
 }
 private track(task:Promise<void>){this.tasks.add(task);void task.catch(()=>{}).finally(()=>this.tasks.delete(task));}
 private detach(session:Session){
  session.invalid=true;clearTimeout(session.authTimer);clearTimeout(session.expires);
  if(!this.sessions.delete(session))return;
  this.voice.leaveVoice(session.connectionId);const followers=this.awareness.remove(session.connectionId);
  if(!session.ref||this.closing)return;
  const ref=session.ref;
  this.track((async()=>{
   for(const other of this.matching(ref))if(followers.includes(other.connectionId)&&await this.recipient(other))await this.control(other,'follow','server',{connectionId:other.connectionId,targetConnectionId:null});
   await this.roster(ref);
  })());
 }
 private async sync(session:Session,id:string,state:SceneState,vector?:Uint8Array){
  const bytes=Y.encodeStateAsUpdate(state.doc,vector);if(bytes.length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  const chunk=512*1024,count=Math.max(1,Math.ceil(bytes.length/chunk));
  for(let index=0;index<count;index++)await this.control(session,'sync_chunk',id,{documentId:state.documentId,sequence:state.sequence,index,count,totalBytes:bytes.length,data:Buffer.from(bytes.subarray(index*chunk,(index+1)*chunk)).toString('base64')});
 }
 private committed=({ref,update}:{ref:SceneRef;receipt:MutationReceipt;update:Uint8Array})=>{
  if(this.closing)return;const key=roomKey(ref);let pending=this.fanouts.get(key);const created=!pending;
  if(!pending){pending={ref,updates:[],bytes:0,overflow:false};this.fanouts.set(key,pending);}
  if(pending.overflow)return;
  // Bound both retained bytes and tiny-update metadata; overflow recovers from durable sync.
  if(pending.bytes+update.length>4*MAX_FRAME_BYTES||pending.updates.length>=1024){pending.overflow=true;pending.updates=[];pending.bytes=0;this.invalidateScene(ref.sceneId,'BACKPRESSURE');return;}
  pending.updates.push(update);pending.bytes+=update.length;
  if(created)this.track(this.flushFanout(key,pending));
 };
 private async flushFanout(key:string,pending:Fanout){
  try{
   while(!this.closing&&!pending.overflow&&pending.updates.length){
    await new Promise(resolve=>setTimeout(resolve,100));
    if(this.closing||pending.overflow)break;
    const updates=pending.updates;pending.updates=[];pending.bytes=0;const id='s'; // Broadcast identity is not an ACK; durable receipts retain original request IDs.
    // Avoid a merged frame exceeding the existing transport ceiling. Each accepted
    // update fits already; small updates share a frame, large updates stand alone.
    const frames:Uint8Array[]=[];let group:Uint8Array[]=[];let bytes=0;
    const flush=()=>{if(group.length){frames.push(encodeFrame({channel:3,updateId:id,payload:group.length===1?group[0]:Y.mergeUpdates(group)}));group=[];bytes=0;}};
    for(const update of updates){if(bytes+update.length>MAX_FRAME_BYTES/2)flush();group.push(update);bytes+=update.length;}flush();
    for(const session of this.matching(pending.ref)){
     // No cached authorization: the fresh read keeps current ACL and ownership
     // fences and reconstructs current durable state before any batch is delivered.
     if(!await this.recipient(session))continue;
     for(const frame of frames)await this.send(session,frame);
    }
   }
  }catch{this.invalidateScene(pending.ref.sceneId,'UNAVAILABLE');}finally{if(this.fanouts.get(key)===pending)this.fanouts.delete(key);}
 }
 private async send(session:Session,data:string|Uint8Array){
  if(session.invalid||session.socket.readyState!==WebSocket.OPEN)return;
  if(session.socket.bufferedAmount+Buffer.byteLength(data)>this.options.maxBufferedBytes){this.invalidate(session,'BACKPRESSURE');return;}
  await new Promise<void>(resolve=>session.socket.send(data,()=>resolve()));
 }
 private control(session:Session,type:string,requestId:string,payload:unknown){return this.send(session,JSON.stringify({type,requestId,payload}));}
 private async error(session:Session,id:string,error:unknown){
  const fault=error instanceof SnapshotHistoryLimitError?new CollabError(error.code,error.message,503,false):error instanceof CollabError?error:new CollabError('UNAVAILABLE','Operation failed; retain pending updates',503,true);
  await this.control(session,'error',id,{code:fault.code,message:fault.message,retryable:fault.retryable});
  if(fault.code==='UNAUTHENTICATED'||fault.code==='FORBIDDEN')this.invalidate(session,fault.code);
 }
 private invalidate(session:Session,reason:string){if(session.invalid)return;this.detach(session);if(session.socket.readyState===WebSocket.OPEN){session.socket.send(JSON.stringify({type:'resync_required',requestId:'server',payload:{reason}}));session.socket.close(1013,reason);const timer=setTimeout(()=>session.socket.terminate(),1000);timer.unref();}}
 private cancelFanouts(matches:(ref:SceneRef)=>boolean){for(const [key,pending] of this.fanouts)if(matches(pending.ref)){pending.overflow=true;pending.updates=[];pending.bytes=0;this.fanouts.delete(key);}}
 invalidateScene(sceneId:string,reason:string){this.cancelFanouts(ref=>ref.sceneId===sceneId);for(const session of this.sessions)if(session.ref?.sceneId===sceneId)this.invalidate(session,reason);}
 invalidateWorkspace(workspaceId?:string,reason='ACL_CHANGED'){this.cancelFanouts(ref=>!workspaceId||ref.workspaceId===workspaceId);for(const session of this.sessions)if(!workspaceId||session.ref?.workspaceId===workspaceId)this.invalidate(session,reason);}
 featureChanged(workspaceId:string,sceneId:string|undefined,eventCursor:string){
  this.track((async()=>{
   const recipients=[...this.sessions].filter(session=>!session.invalid&&session.principal?.principalType==='user'&&session.ref?.workspaceId===workspaceId&&(!sceneId||session.ref.sceneId===sceneId));
   for(const session of recipients)if(await this.recipient(session))await this.control(session,'feature_changed','server',{eventCursor});
  })());
 }
 async close(){this.closing=true;clearInterval(this.heartbeat);this.store.off('committed',this.committed);const sessions=[...this.sessions];for(const session of sessions){this.detach(session);session.socket.terminate();}await Promise.allSettled([...this.tasks,...sessions.map(s=>s.chain)]);}
}
