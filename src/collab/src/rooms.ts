import WebSocket from 'ws';
import * as Y from 'yjs';
import {createDecoder,readVarUint,readVarUint8Array,hasContent} from 'lib0/decoding';
import {decodeFrame,encodeFrame,MAX_FRAME_BYTES,type SceneRef,type MutationReceipt} from '@collab/contracts';
import {CollabError,authorizeSync,requestID,type Principal,type Decision} from './auth.js';
import {MAX_SNAPSHOT_BYTES,SnapshotHistoryLimitError} from './document.js';
import type {Persistence,SceneState} from './persistence.js';
import {RateLimit} from './rate-limit.js';

function invalidInput<T>(parse:()=>T):T {try{return parse();}catch(error){if(error instanceof CollabError)throw error;throw new CollabError('INVALID_UPDATE','Malformed transport input',400);}}
type Session={socket:WebSocket;ref?:SceneRef;principal?:Principal;claims?:Decision;access?:string;expires?:NodeJS.Timeout;authTimer:NodeJS.Timeout;pending:number;chain:Promise<void>;ip:string;invalid:boolean};
export class Rooms {
 private sessions=new Set<Session>();private rates=new RateLimit();private tasks=new Set<Promise<void>>();
 constructor(private store:Persistence,private options:{updatesPerMinute:number;maxBufferedBytes:number}){store.on('committed',this.committed);}
 attach(socket:WebSocket,ip:string){
  const session:Session={socket,pending:0,chain:Promise.resolve(),ip,invalid:false,authTimer:setTimeout(()=>socket.close(4401,'Authentication required'),5000)};
  this.sessions.add(session);socket.on('error',()=>{});
  socket.on('close',()=>{session.invalid=true;clearTimeout(session.authTimer);clearTimeout(session.expires);this.sessions.delete(session);});
  socket.on('message',(data,binary)=>{
   if(session.invalid)return;
   if(++session.pending>32){this.invalidate(session,'BACKPRESSURE');return;}
   const bytes=Buffer.isBuffer(data)?new Uint8Array(data):new Uint8Array(data as ArrayBuffer);
   session.chain=session.chain.then(async()=>{if(session.invalid)return;let id='unknown';try{
    this.rates.take('ip:'+ip,600);
    if(binary){const frame=invalidInput(()=>decodeFrame(bytes));id=frame.updateId;requestID(id);await this.binary(session,frame.channel,id,frame.payload);}
    else{if(bytes.length>32768)throw new CollabError('INVALID_UPDATE','Control too large',400);const control=invalidInput(()=>JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes)));id=control.requestId;requestID(id);if(!control||Object.keys(control).some(k=>!['type','requestId','payload'].includes(k))||!['auth','refresh'].includes(control.type)||!control.payload||Object.keys(control.payload).some(k=>k!=='token'))throw new CollabError('INVALID_UPDATE','Invalid control',400);await this.authenticate(session,id,control.payload.token,control.type);}
   }catch(error){await this.error(session,id,error);}finally{session.pending--;}});
  });
 }
 private async authenticate(session:Session,id:string,token:string,type:string){
  if(type==='auth'&&session.principal||type==='refresh'&&!session.principal)throw new CollabError('UNAUTHENTICATED','Invalid authentication transition',401);
  const claims=await this.store.authority.verifyRoom(token),ref={workspaceId:claims.workspaceId,sceneId:claims.sceneId};
  if(session.ref&&(session.ref.workspaceId!==ref.workspaceId||session.ref.sceneId!==ref.sceneId||session.principal!.principalId!==claims.principalId||session.principal!.principalType!==claims.principalType))throw new CollabError('FORBIDDEN','Refresh cannot change room or principal');
  const principal={roomToken:token,principalId:claims.principalId,principalType:claims.principalType};
  const state=await this.store.read(ref,principal);
  try{
   session.ref=ref;session.principal=principal;session.claims=claims;session.access=state.access;
   clearTimeout(session.authTimer);clearTimeout(session.expires);session.expires=setTimeout(()=>this.invalidate(session,'TOKEN_EXPIRED'),Math.max(0,claims.exp*1000-Date.now()));
   await this.control(session,'ready',id,{...ref,principalId:principal.principalId,access:state.access,documentId:state.documentId,sequence:state.sequence,stateVector:state.stateVector});
   await this.sync(session,id,state);
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
   this.rates.take('awareness:'+session.principal.principalId,30,1000);
   if(payload.length>8192)throw new CollabError('INVALID_UPDATE','Awareness too large',400);
   const value=invalidInput(()=>JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(payload)));
   if(!value||Object.keys(value).some(k=>!['cursor','selectedIds'].includes(k))||(value.cursor!==undefined&&(!value.cursor||Object.keys(value.cursor).some(k=>!['x','y'].includes(k))||!Number.isFinite(value.cursor.x)||!Number.isFinite(value.cursor.y)))||(value.selectedIds!==undefined&&(!Array.isArray(value.selectedIds)||value.selectedIds.length>256||value.selectedIds.some((v:unknown)=>typeof v!=='string'||v.length>256))))throw new CollabError('INVALID_UPDATE','Invalid awareness',400);
   const state=await this.store.read(session.ref,session.principal);state.doc.destroy();
   for(const other of this.sessions)if(other!==session&&other.ref?.sceneId===session.ref.sceneId)await this.control(other,'awareness',id,{principalId:session.principal.principalId,...value});return;
  }
  if(session.access!=='edit')throw new CollabError('FORBIDDEN');
  this.rates.take('principal:'+session.principal.principalType+':'+session.principal.principalId,this.options.updatesPerMinute);
  const receipt=await this.store.appendUpdate(session.ref,session.principal,id,update);
  await this.control(session,'ack',id,receipt);
 }
 private async sync(session:Session,id:string,state:SceneState,vector?:Uint8Array){
  const bytes=Y.encodeStateAsUpdate(state.doc,vector);if(bytes.length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  const chunk=512*1024,count=Math.max(1,Math.ceil(bytes.length/chunk));
  for(let index=0;index<count;index++)await this.control(session,'sync_chunk',id,{documentId:state.documentId,sequence:state.sequence,index,count,totalBytes:bytes.length,data:Buffer.from(bytes.subarray(index*chunk,(index+1)*chunk)).toString('base64')});
 }
 private committed=({ref,receipt,update}:{ref:SceneRef;receipt:MutationReceipt;update:Uint8Array})=>{
  const task=(async()=>{for(const session of this.sessions){if(session.ref?.sceneId!==ref.sceneId||!session.principal||session.invalid)continue;try{
   // Fresh read authority also protects recipients; cached room access is never sufficient.
   const state=await this.store.read(ref,session.principal);state.doc.destroy();
   await this.send(session,encodeFrame({channel:3,updateId:receipt.requestId,payload:update}));
  }catch{this.invalidate(session,'AUTHORITY_CHANGED');}}})();this.tasks.add(task);void task.finally(()=>this.tasks.delete(task));
 };
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
 private invalidate(session:Session,reason:string){if(session.invalid)return;session.invalid=true;if(session.socket.readyState===WebSocket.OPEN){session.socket.send(JSON.stringify({type:'resync_required',requestId:'server',payload:{reason}}));session.socket.close(1013,reason);const timer=setTimeout(()=>session.socket.terminate(),1000);timer.unref();}}
 invalidateWorkspace(workspaceId?:string,reason='ACL_CHANGED'){for(const session of this.sessions)if(!workspaceId||session.ref?.workspaceId===workspaceId)this.invalidate(session,reason);}
 async close(){this.store.off('committed',this.committed);for(const session of this.sessions){session.invalid=true;session.socket.terminate();clearTimeout(session.authTimer);clearTimeout(session.expires);}await Promise.allSettled([...this.tasks,...[...this.sessions].map(s=>s.chain)]);this.sessions.clear();}
}
