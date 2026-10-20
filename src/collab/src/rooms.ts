import WebSocket from 'ws';
import {randomUUID} from 'node:crypto';
import {Awareness,fields,parseAwareness,parseViewport,roomKey} from './awareness.js';
import {Voice,parseSignal} from './voice.js';
import {MediaAdmission,parseScreenSignal,parseScreenStop,type ScreenSession} from './screen-share.js';
import {IngressLimit} from './ingress.js';
import * as Y from 'yjs';
import {createDecoder,readVarUint,readVarUint8Array,hasContent} from 'lib0/decoding';
import {MAX_FRAME_BYTES,decodeFrame,encodeFrame,type SceneRef,type MutationReceipt} from '@collab/contracts';
import {CollabError,authorizeSync,requestID,type Principal,type Decision,type PresentationDecision} from './auth.js';
import {MAX_SNAPSHOT_BYTES,SnapshotHistoryLimitError} from './document.js';
import type {Persistence,SceneState} from './persistence.js';
import {RateLimit} from './rate-limit.js';
import {requireFeature,requireScreenFeature} from './screen-authority.js';
import {PresentationSessions,parsePresentationAdmit,parsePresentationStart} from './presentation.js';

export const HEARTBEAT_INTERVAL_MS=25_000;
const EPHEMERAL_CONTROLS=['presence','follow','viewport','voice_join','voice_leave','voice_mute','voice_signal','screen_request','screen_stop','screen_signal','presentation_start','presentation_admit','presentation_stop'];
const MAX_CONTROL_BYTES=32768;
const AUTH_PER_MINUTE=60,EPHEMERAL_PER_MINUTE=120,AWARENESS_PER_SECOND=30;

function invalidInput<T>(parse:()=>T):T {try{return parse();}catch(error){if(error instanceof CollabError)throw error;throw new CollabError('INVALID_UPDATE','Malformed transport input',400);}}
type Session={connectionId:string;alive:boolean;socket:WebSocket;ref?:SceneRef;principal?:Principal;claims?:Decision;access?:string;presentation?:{kind:'pending'|'audience';sessionId:string;room:string;credential?:PresentationDecision};expires?:NodeJS.Timeout;authTimer:NodeJS.Timeout;pending:number;chain:Promise<void>;ip:string;invalid:boolean};
type Fanout={ref:SceneRef;updates:Array<{bytes:Uint8Array;sequence?:string}>;bytes:number;overflow:boolean};
export class Rooms {
 private sessions=new Set<Session>();private rates=new RateLimit();private ingress=new IngressLimit();private tasks=new Set<Promise<void>>();
 private fanouts=new Map<string,Fanout>();
 private awareness=new Awareness();private media:MediaAdmission;private voice:Voice;private presentations=new PresentationSessions();private heartbeat:NodeJS.Timeout;private closing=false;
 constructor(private store:Persistence,private options:{updatesPerMinute:number;maxBufferedBytes:number;heartbeatIntervalMs?:number}){
  this.media=new MediaAdmission(session=>this.screenExpired(session));this.voice=new Voice(this.media);
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
     if(['auth','refresh','presentation_auth'].includes(control.type)){
      this.rates.take('auth:'+ip,AUTH_PER_MINUTE);
      if(!control.payload||Object.keys(control.payload).some(k=>k!=='token'))throw new CollabError('INVALID_UPDATE','Invalid auth control',400);
      if(control.type==='presentation_auth')await this.authenticatePresentation(session,id,control.payload.token);else await this.authenticate(session,id,control.payload.token,control.type);
     }else if(EPHEMERAL_CONTROLS.includes(control.type))await this.ephemeral(session,id,control.type,control.payload);
     else throw new CollabError('INVALID_UPDATE','Unknown control',400);
    }
   }catch(error){await this.error(session,id,error);}finally{session.pending--;}});
   this.track(session.chain);
  });
 }
 private async authenticatePresentation(session:Session,id:string,token:string){
  if(session.principal||session.presentation)throw new CollabError('UNAUTHENTICATED','Invalid authentication transition',401);
  const claims=await this.store.authority.verifyPresentation(token,'presentation-pending'),ref={workspaceId:claims.workspaceId,sceneId:claims.sceneId},room=roomKey(ref);
  const status=this.presentations.wait(room,claims.sessionId,{connectionId:session.connectionId,principalId:claims.principalId,displayName:claims.displayName});
  session.presentation={kind:'pending',sessionId:claims.sessionId,room};clearTimeout(session.authTimer);clearTimeout(session.expires);session.expires=setTimeout(()=>this.invalidate(session,'TOKEN_EXPIRED'),Math.max(0,claims.exp*1000-Date.now()));
  await this.control(session,'presentation_waiting',id,status);
  const state=this.presentations.state(room),host=[...this.sessions].find(value=>value.connectionId===state?.hostConnectionId);if(host)await this.control(host,'presentation_waiting','server',{sessionId:claims.sessionId,revision:state!.revision,participant:{connectionId:session.connectionId,displayName:claims.displayName}});
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
   if(subtype===0){this.rates.take('read:'+session.principal.principalId,120);if(bytes.length>65536)throw new CollabError('INVALID_UPDATE','State vector too large',400);const state=await this.readCurrent(session);try{session.access=state.access;await this.sync(session,id,state,bytes);}finally{state.doc.destroy();}return;}
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
  const state=await this.readCurrent(session);
  try{if(session.invalid)throw new CollabError('UNAUTHENTICATED');session.access=state.access;this.awareness.register(session.ref,{connectionId:session.connectionId,principalId:session.principal.principalId,principalType:session.principal.principalType,access:state.access});}finally{state.doc.destroy();}
 }
 private async readCurrent(session:Session):Promise<SceneState>{
  if(session.presentation?.kind==='audience'&&session.presentation.credential){await requireFeature(this.store.authority,session.principal!.roomToken,session.ref!,'presentation:host');this.resolvePresentationAudience(session.ref!,session.presentation.sessionId,session.principal!.principalId);return this.store.readPresentation(session.ref!,session.principal!,session.presentation.credential);}
  return this.store.read(session.ref!,session.principal!);
 }
 private matching(ref:SceneRef){return [...this.sessions].filter(s=>!s.invalid&&s.ref&&roomKey(s.ref)===roomKey(ref));}
 private async recipient(session:Session){try{await this.fresh(session);return true;}catch(error){this.invalidate(session,error instanceof CollabError&&['UNAUTHENTICATED','FORBIDDEN','STALE_AUTHORITY'].includes(error.code)?'AUTHORITY_CHANGED':'UNAVAILABLE');return false;}}
 private async broadcast(ref:SceneRef,type:string,id:string,payload:unknown,filter:(s:Session)=>boolean=()=>true){
  for(const other of this.matching(ref))if(filter(other)&&await this.recipient(other))await this.control(other,type,id,payload);
 }
 private async roster(ref:SceneRef,id='server'){
  const recipients=this.matching(ref);await Promise.all(recipients.map(s=>this.recipient(s)));
  for(const session of recipients){const participants=this.awareness.list(ref).map(p=>({...p,...this.voice.state(p.connectionId),...this.media.state(p.connectionId)}));await this.control(session,'roster',id,{participants});}
 }
 private async ephemeral(session:Session,id:string,type:string,payload:unknown){
  const voiceControl=type.startsWith('voice_');
  const screenControl=type.startsWith('screen_');
  try{this.rates.take((type==='viewport'?'awareness:':'control:')+session.connectionId,type==='viewport'?AWARENESS_PER_SECOND:EPHEMERAL_PER_MINUTE,type==='viewport'?1000:60_000);}catch{throw new CollabError(voiceControl?'VOICE_RATE_LIMITED':screenControl?'SCREEN_RATE_LIMITED':'PRESENCE_RATE_LIMITED','Ephemeral control rate exceeded',429);}
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
  if(type.startsWith('presentation_')){await this.presentation(session,id,type,payload);return;}
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
  if(screenControl){await this.screen(session,id,type,payload);return;}
  const signal=parseSignal(payload);const target=this.matching(ref).find(s=>s.connectionId===signal.to);
  if(!target||!this.voice.canSignal(roomKey(ref),session.connectionId,signal.to)||!await this.recipient(target))throw new CollabError('VOICE_TARGET','Voice peer is not available',400);
  // Admission can end while the authority request is in flight.
  if(session.invalid||target.invalid||!this.voice.canSignal(roomKey(ref),session.connectionId,signal.to))throw new CollabError('VOICE_TARGET','Voice peer is not available',400);
  await this.control(target,'voice_signal',id,{...signal,from:session.connectionId});
 }
 private async presentation(session:Session,id:string,type:string,payload:unknown){
  const ref=session.ref!,room=roomKey(ref);
  if(session.principal?.principalType!=='user'||session.access!=='edit')throw new CollabError('PRESENTATION_FORBIDDEN','Only signed-in editors can host',403);
  try{await requireFeature(this.store.authority,session.principal.roomToken,ref,'presentation:host');}catch(error){const state=this.presentations.state(room);if(state?.hostConnectionId===session.connectionId){this.presentations.disconnect(session.connectionId);for(const target of this.sessions)if(target.presentation?.sessionId===state.sessionId){await this.control(target,'presentation_stop','server',{sessionId:state.sessionId,reason:'AUTHORITY_CHANGED'});this.invalidate(target,'PRESENTATION_ENDED');}}if(error instanceof CollabError&&error.code==='UNAVAILABLE')throw new CollabError('PRESENTATION_AUTHORITY_UNAVAILABLE','Presentation authority unavailable',503,true);throw new CollabError('PRESENTATION_FORBIDDEN','Presentation is disabled or no longer permitted',403);}
  if(type==='presentation_start'){
   const input=parsePresentationStart(payload),claims=await this.store.authority.verifyPresentation(input.token,'presentation-host');
   if(claims.workspaceId!==ref.workspaceId||claims.sceneId!==ref.sceneId||claims.principalId!==session.principal.principalId)throw new CollabError('PRESENTATION_FORBIDDEN','Host credential binding mismatch',403);
   const state=this.presentations.start(room,claims.sessionId,session.connectionId,session.principal.principalId);await this.control(session,'presentation_state',id,state);return;
  }
  if(type==='presentation_admit'){
   const input=parsePresentationAdmit(payload),claims=await this.store.authority.verifyPresentation(input.token,'presentation-audience'),state=this.presentations.state(room);
   if(!state||claims.workspaceId!==ref.workspaceId||claims.sceneId!==ref.sceneId||claims.sessionId!==state.sessionId||claims.principalId!==state.pending.find(p=>p.connectionId===input.connectionId)?.principalId)throw new CollabError('PRESENTATION_TARGET','Audience credential binding mismatch',403);
   if(claims.grantVersion!==String(input.expectedRevision))throw new CollabError('PRESENTATION_CONFLICT','Audience credential revision changed',409);
   const target=[...this.sessions].find(value=>value.connectionId===input.connectionId&&value.presentation?.kind==='pending'&&value.presentation.sessionId===claims.sessionId);if(!target)throw new CollabError('PRESENTATION_TARGET','Waiting participant disconnected',404);
   const principal={roomToken:input.token,principalId:claims.principalId,principalType:'public'},audienceRef={workspaceId:claims.workspaceId,sceneId:claims.sceneId},scene=await this.store.readPresentation(audienceRef,principal,claims);
   try{const admitted=this.presentations.admit(room,claims.sessionId,session.connectionId,input.connectionId,input.expectedRevision);target.ref=audienceRef;target.principal=principal;target.claims={...claims,purpose:'room',access:'view'} as Decision;target.access='view';target.presentation={kind:'audience',sessionId:claims.sessionId,room,credential:claims};clearTimeout(target.expires);target.expires=setTimeout(()=>this.invalidate(target,'TOKEN_EXPIRED'),Math.max(0,claims.exp*1000-Date.now()));this.awareness.register(audienceRef,{connectionId:target.connectionId,principalId:principal.principalId,principalType:'public',access:'view'});await this.control(target,'presentation_admitted',id,{sessionId:claims.sessionId,token:input.token,expiresAt:claims.exp});await this.control(target,'ready',id,{...audienceRef,connectionId:target.connectionId,principalId:principal.principalId,access:'view',documentId:scene.documentId,sequence:scene.sequence,stateVector:scene.stateVector,presentationSessionId:claims.sessionId});await this.sync(target,id,scene);await this.control(session,'presentation_state',id,admitted.state);await this.roster(ref,id);}finally{scene.doc.destroy();}return;
  }
  const input=fields(payload,['sessionId'],'PRESENTATION_INVALID');if(typeof input.sessionId!=='string')throw new CollabError('PRESENTATION_INVALID','Invalid presentation session',400);const ended=this.presentations.stop(room,input.sessionId,session.connectionId);for(const target of this.sessions)if(target.presentation?.sessionId===input.sessionId){await this.control(target,'presentation_stop','server',{sessionId:input.sessionId,reason:'HOST_ENDED'});this.invalidate(target,'PRESENTATION_ENDED');}await this.control(session,'presentation_state',id,ended);
 }
 private async screen(session:Session,id:string,type:string,payload:unknown){
  const ref=session.ref!,room=roomKey(ref);
  const current=this.media.screen(room);
  if(type!=='screen_stop'&&current?.publisher===session.connectionId&&(session.principal?.principalType!=='user'||session.access!=='edit')){
   const ended=this.media.endScreen(room,current.sessionId);if(ended){await this.broadcast(ref,'screen_stop','server',{sessionId:ended.sessionId,publisher:ended.publisher,reason:'AUTHORITY_CHANGED'});await this.roster(ref,id);}throw new CollabError('SCREEN_FORBIDDEN','Only signed-in editors can share a screen',403);
  }
  if(type==='screen_request'){
   const request=fields(payload,['sessionId'],'SCREEN_INVALID');
   if(request.sessionId!==undefined){
    const input=parseScreenStop(request);await this.requireScreenPublisher(session,ref,room,id);if(!this.media.activate(room,session.connectionId,input.sessionId))throw new CollabError('SCREEN_TARGET','Screen reservation is unavailable',400);
    const admission=this.media.join(room,session.connectionId,'screen');await this.control(session,'screen_admission',id,admission);await this.roster(ref,id);return;
   }
   const publisher=this.media.publisher(room);
   if(!publisher&&(session.principal?.principalType!=='user'||session.access!=='edit'))throw new CollabError('SCREEN_FORBIDDEN','Only signed-in editors can share a screen');
   if(!publisher)await this.requireScreenPublisher(session,ref,room,id);else await this.requireActiveScreenPublisher(ref,room,id);
   const admission=this.media.join(room,session.connectionId,'screen');await this.control(session,'screen_admission',id,admission);
   if(!admission.admitted)return;
   if(admission.receiving&&admission.publisher){const owner=this.matching(ref).find(value=>value.connectionId===admission.publisher);if(owner&&await this.recipient(owner))await this.control(owner,'screen_admission','server',admission);}
   await this.roster(ref,id);return;
  }
  if(type==='screen_stop'){
   const input=parseScreenStop(payload),current=this.media.screen(room);if(!current||current.sessionId!==input.sessionId)throw new CollabError('SCREEN_TARGET','Screen session is unavailable',400);
   await this.requireActiveScreenPublisher(ref,room,id);
   const ended=this.media.leave(room,session.connectionId,'screen');
   if(ended)await this.broadcast(ref,'screen_stop','server',{sessionId:ended.sessionId,publisher:ended.publisher});
   else {const publisher=this.matching(ref).find(value=>value.connectionId===current.publisher);if(publisher&&await this.recipient(publisher))await this.control(publisher,'screen_stop','server',{sessionId:current.sessionId,connectionId:session.connectionId});}
   await this.roster(ref,id);return;
  }
  const signal=parseScreenSignal(payload),target=this.matching(ref).find(value=>value.connectionId===signal.to);
  await this.requireActiveScreenPublisher(ref,room,id);
  if(!target||!this.media.canSignal(room,session.connectionId,signal.to,'screen',signal.sessionId)||!await this.recipient(target))throw new CollabError('SCREEN_TARGET','Screen peer is not available',400);
  if(session.invalid||target.invalid||!this.media.canSignal(room,session.connectionId,signal.to,'screen',signal.sessionId))throw new CollabError('SCREEN_TARGET','Screen peer is not available',400);
  await this.control(target,'screen_signal',id,{...signal,from:session.connectionId});
 }
 private async requireActiveScreenPublisher(ref:SceneRef,room:string,id:string){
  const current=this.media.screen(room);if(!current)throw new CollabError('SCREEN_TARGET','Screen session is unavailable',400);
  const publisher=this.matching(ref).find(value=>value.connectionId===current.publisher);if(!publisher)throw new CollabError('SCREEN_TARGET','Screen publisher is unavailable',400);
  await this.requireScreenPublisher(publisher,ref,room,id);
  const refreshed=this.media.screen(room);if(!refreshed||refreshed.sessionId!==current.sessionId||refreshed.publisher!==publisher.connectionId)throw new CollabError('SCREEN_TARGET','Screen session changed during authority check',400);
 }
 private async requireScreenPublisher(session:Session,ref:SceneRef,room:string,id:string){
  try{await requireScreenFeature(this.store.authority,session.principal!.roomToken,ref);}
  catch(error){
   const current=this.media.screen(room);if(current?.publisher===session.connectionId){const ended=this.media.endScreen(room,current.sessionId);if(ended){await this.broadcast(ref,'screen_stop','server',{sessionId:ended.sessionId,publisher:ended.publisher,reason:'FEATURE_REVOKED'});await this.roster(ref,id);}}
   if(error instanceof CollabError&&error.code==='UNAVAILABLE')throw new CollabError('SCREEN_AUTHORITY_UNAVAILABLE','Screen authority unavailable; keep drawing and retry',503,true);
   throw new CollabError('SCREEN_FORBIDDEN','Screen sharing is disabled or no longer permitted',403);
  }
 }
 private screenExpired(session:ScreenSession){
  if(this.closing)return;const owner=[...this.sessions].find(value=>value.ref&&roomKey(value.ref)===session.room);if(!owner?.ref)return;const ref=owner.ref;
  this.track((async()=>{await this.broadcast(ref,'screen_stop','server',{sessionId:session.sessionId,publisher:session.publisher,reason:'RESERVATION_EXPIRED'});await this.roster(ref);})());
 }
 private track(task:Promise<void>){this.tasks.add(task);void task.catch(()=>{}).finally(()=>this.tasks.delete(task));}
 private detach(session:Session){
  session.invalid=true;clearTimeout(session.authTimer);clearTimeout(session.expires);
  if(!this.sessions.delete(session))return;
  this.voice.leaveVoice(session.connectionId);const endedScreen=this.media.leaveAll(session.connectionId);const endedPresentation=this.presentations.disconnect(session.connectionId);const followers=this.awareness.remove(session.connectionId);
  if(endedPresentation&&!this.closing)this.track((async()=>{
   if(endedPresentation.ended){for(const target of this.sessions)if(target.presentation?.sessionId===endedPresentation.ended.sessionId){await this.control(target,'presentation_stop','server',{sessionId:endedPresentation.ended.sessionId,reason:'HOST_DISCONNECTED'});this.invalidate(target,'PRESENTATION_ENDED');}return;}
   if(endedPresentation.state){const host=[...this.sessions].find(value=>value.connectionId===endedPresentation.state.hostConnectionId);if(host)await this.control(host,'presentation_state','server',endedPresentation.state);}
  })());
  if(!session.ref||this.closing)return;
  const ref=session.ref;
  this.track((async()=>{
   if(endedScreen)await this.broadcast(ref,'screen_stop','server',{sessionId:endedScreen.sessionId,publisher:endedScreen.publisher});
   for(const other of this.matching(ref))if(followers.includes(other.connectionId)&&await this.recipient(other))await this.control(other,'follow','server',{connectionId:other.connectionId,targetConnectionId:null});
   await this.roster(ref);
  })());
 }
 private async sync(session:Session,id:string,state:SceneState,vector?:Uint8Array){
  const bytes=Y.encodeStateAsUpdate(state.doc,vector);if(bytes.length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  const chunk=512*1024,count=Math.max(1,Math.ceil(bytes.length/chunk));
  for(let index=0;index<count;index++)await this.control(session,'sync_chunk',id,{documentId:state.documentId,sequence:state.sequence,index,count,totalBytes:bytes.length,data:Buffer.from(bytes.subarray(index*chunk,(index+1)*chunk)).toString('base64')});
 }
 private committed=({ref,receipt,update}:{ref:SceneRef;receipt:MutationReceipt;update:Uint8Array})=>{
  if(this.closing)return;const key=roomKey(ref);let pending=this.fanouts.get(key);const created=!pending;
  if(!pending){pending={ref,updates:[],bytes:0,overflow:false};this.fanouts.set(key,pending);}
  if(pending.overflow)return;
  // Bound both retained bytes and tiny-update metadata; overflow recovers from durable sync.
  if(pending.bytes+update.length>4*MAX_FRAME_BYTES||pending.updates.length>=1024){pending.overflow=true;pending.updates=[];pending.bytes=0;this.invalidateScene(ref.sceneId,'BACKPRESSURE');return;}
  pending.updates.push({bytes:update,sequence:receipt.sequence});pending.bytes+=update.length;
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
    const oversized=updates.some(item=>item.bytes.length>MAX_FRAME_BYTES-256);
    const frames:Uint8Array[]=[];let group:Uint8Array[]=[];let bytes=0;
    const flush=()=>{if(group.length){frames.push(encodeFrame({channel:3,updateId:id,payload:group.length===1?group[0]:Y.mergeUpdates(group)}));group=[];bytes=0;}};
    if(!oversized){for(const item of updates){const update=item.bytes;if(bytes+update.length>MAX_FRAME_BYTES/2)flush();group.push(update);bytes+=update.length;}flush();}
    const sequence=updates.map(item=>item.sequence).filter((value):value is string=>typeof value==='string'&&/^\d+$/.test(value)).sort((a,b)=>BigInt(a)<BigInt(b)?-1:BigInt(a)>BigInt(b)?1:0).at(-1);
    for(const session of this.matching(pending.ref)){
     // No cached authorization: the fresh read keeps current ACL and ownership
     // fences and reconstructs current durable state before any batch is delivered.
     if(!await this.recipient(session))continue;
     if(oversized){const state=await this.readCurrent(session);try{await this.sync(session,'server',state);}finally{state.doc.destroy();}}
     else for(const frame of frames)await this.send(session,frame);
     if(sequence)await this.control(session,'committed','server',{sequence});
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
   const rooms=new Set(recipients.map(session=>roomKey(session.ref!)));for(const room of rooms){const state=this.presentations.state(room);if(!state)continue;this.presentations.disconnect(state.hostConnectionId);for(const target of this.sessions)if(target.presentation?.sessionId===state.sessionId){await this.control(target,'presentation_stop','server',{sessionId:state.sessionId,reason:'AUTHORITY_CHANGED'});this.invalidate(target,'PRESENTATION_ENDED');}}
   for(const session of recipients)if(await this.recipient(session))await this.control(session,'feature_changed','server',{eventCursor});
  })());
 }
 resolvePresentationHost(ref:SceneRef,sessionId:string,principalId:string){const state=this.presentations.state(roomKey(ref));if(!state||state.sessionId!==sessionId||state.hostPrincipalId!==principalId)throw new CollabError('PRESENTATION_ENDED','Presentation host is not active',409);return {active:true,revision:state.revision};}
 resolvePresentationPending(ref:SceneRef,sessionId:string,hostPrincipalId:string,connectionId:string,expectedRevision:number){const state=this.presentations.state(roomKey(ref)),participant=state?.pending.find(value=>value.connectionId===connectionId);if(!state||state.sessionId!==sessionId||state.hostPrincipalId!==hostPrincipalId||state.revision!==expectedRevision||!participant)throw new CollabError('PRESENTATION_TARGET','Waiting participant is unavailable',409);return {sessionId,...ref,principalId:participant.principalId,displayName:participant.displayName};}
 resolvePresentationAudience(ref:SceneRef,sessionId:string,principalId:string){const state=this.presentations.state(roomKey(ref));if(!state||state.sessionId!==sessionId||!state.admitted.some(value=>value.principalId===principalId))throw new CollabError('PRESENTATION_TARGET','Audience participant is unavailable',409);return {active:true,hostPrincipalId:state.hostPrincipalId};}
 async close(){this.closing=true;clearInterval(this.heartbeat);this.media.close();this.store.off('committed',this.committed);const sessions=[...this.sessions];for(const session of sessions){this.detach(session);session.socket.terminate();}await Promise.allSettled([...this.tasks,...sessions.map(s=>s.chain)]);}
}
