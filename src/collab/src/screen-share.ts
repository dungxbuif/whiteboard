import {randomUUID} from 'node:crypto';
import {CollabError} from './auth.js';
import {fields} from './awareness.js';

export const MAX_MEDIA_CONNECTIONS=10;
export const SCREEN_RESERVATION_MS=30_000;
export type MediaKind='voice'|'screen';
export type MediaAdmissionResult={admitted:boolean;connectionId:string;kind:MediaKind;peers:string[];publisher?:string;receiving?:boolean;sessionId?:string;reason?:'MEDIA_FULL'|'SCREEN_BUSY'};
export type ScreenSession={room:string;publisher:string;sessionId:string;active:boolean};
type Member={room:string;kinds:Set<MediaKind>};
type ScreenRecord=ScreenSession&{timer:NodeJS.Timeout;receivers:Set<string>};

export class MediaAdmission {
 private members=new Map<string,Member>();
 private screens=new Map<string,ScreenRecord>();
 constructor(private expired:(session:ScreenSession)=>void=()=>{}){}
 private roomMembers(room:string){return [...this.members].filter(([,member])=>member.room===room);}
 join(room:string,connectionId:string,kind:MediaKind):MediaAdmissionResult {
  const existing=this.members.get(connectionId);
  if(existing&&existing.room!==room)return {admitted:false,connectionId,kind,peers:[],reason:'MEDIA_FULL'};
  if(!existing&&this.roomMembers(room).length>=MAX_MEDIA_CONNECTIONS)return {admitted:false,connectionId,kind,peers:[],reason:'MEDIA_FULL'};
  const member=existing??{room,kinds:new Set<MediaKind>()};
  if(kind==='voice'){
   const peers=this.roomMembers(room).filter(([id,value])=>id!==connectionId&&value.kinds.has('voice')).map(([id])=>id);
   member.kinds.add('voice');this.members.set(connectionId,member);return {admitted:true,connectionId,kind,peers};
  }
  const current=this.screens.get(room);
  if(!current){
   const sessionId=randomUUID();const record:ScreenRecord={room,publisher:connectionId,sessionId,active:false,receivers:new Set(),timer:setTimeout(()=>this.expire(room,sessionId),SCREEN_RESERVATION_MS)};record.timer.unref();
   this.screens.set(room,record);member.kinds.add('screen');this.members.set(connectionId,member);
   return {admitted:true,connectionId,kind,peers:[],publisher:connectionId,receiving:false,sessionId};
  }
  member.kinds.add('screen');this.members.set(connectionId,member);if(connectionId!==current.publisher)current.receivers.add(connectionId);
  return {admitted:true,connectionId,kind,peers:connectionId===current.publisher?[...current.receivers]:[current.publisher],publisher:current.publisher,receiving:connectionId!==current.publisher,sessionId:current.sessionId};
 }
 publisher(room:string){return this.screens.get(room)?.publisher;}
 screen(room:string):ScreenSession|undefined {const value=this.screens.get(room);return value?{room:value.room,publisher:value.publisher,sessionId:value.sessionId,active:value.active}:undefined;}
 state(connectionId:string){const member=this.members.get(connectionId),screen=member?this.screens.get(member.room):undefined;return {screenJoined:member?.kinds.has('screen')??false,screenPublisher:screen?.publisher===connectionId,screenSessionId:screen&&member?.kinds.has('screen')?screen.sessionId:undefined};}
 activate(room:string,connectionId:string,sessionId:string){const screen=this.screens.get(room);if(!screen||screen.publisher!==connectionId||screen.sessionId!==sessionId)return false;screen.active=true;clearTimeout(screen.timer);return true;}
 canSignal(room:string,from:string,to:string,kind:MediaKind,sessionId?:string){
  if(from===to)return false;const source=this.members.get(from),target=this.members.get(to);if(source?.room!==room||target?.room!==room||!source.kinds.has(kind)||!target.kinds.has(kind))return false;
  if(kind==='screen'){const screen=this.screens.get(room);if(!screen||screen.sessionId!==sessionId||(from!==screen.publisher&&to!==screen.publisher))return false;}
  return true;
 }
 leave(room:string,connectionId:string,kind:MediaKind):ScreenSession|undefined {
  const member=this.members.get(connectionId);if(!member||member.room!==room||!member.kinds.has(kind))return;
  if(kind==='screen'){
   const screen=this.screens.get(room);if(screen?.publisher===connectionId)return this.endScreen(room,screen.sessionId);
   screen?.receivers.delete(connectionId);
  }
  member.kinds.delete(kind);if(!member.kinds.size)this.members.delete(connectionId);return;
 }
 leaveAll(connectionId:string):ScreenSession|undefined {const member=this.members.get(connectionId);if(!member)return;const ended=member.kinds.has('screen')?this.leave(member.room,connectionId,'screen'):undefined;this.members.delete(connectionId);return ended;}
 endScreen(room:string,sessionId:string):ScreenSession|undefined {
  const screen=this.screens.get(room);if(!screen||screen.sessionId!==sessionId)return;clearTimeout(screen.timer);this.screens.delete(room);
  for(const id of [screen.publisher,...screen.receivers]){const member=this.members.get(id);member?.kinds.delete('screen');if(member&&!member.kinds.size)this.members.delete(id);}
  return {room:screen.room,publisher:screen.publisher,sessionId:screen.sessionId,active:screen.active};
 }
 private expire(room:string,sessionId:string){const ended=this.endScreen(room,sessionId);if(ended)this.expired(ended);}
 close(){for(const screen of this.screens.values())clearTimeout(screen.timer);this.screens.clear();this.members.clear();}
}

const sessionID=(value:unknown):value is string=>typeof value==='string'&&/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
export function parseScreenSignal(value:unknown){
 const input=fields(value,['to','sessionId','description','candidate'],'SCREEN_INVALID');
 if(typeof input.to!=='string'||input.to.length>128||!sessionID(input.sessionId)||(!('description'in input)===!('candidate'in input)))throw new CollabError('SCREEN_INVALID','Invalid screen signal',400);
 if('description'in input){const description=fields(input.description,['type','sdp'],'SCREEN_INVALID');if(!['offer','answer'].includes(String(description.type))||typeof description.sdp!=='string'||Buffer.byteLength(description.sdp)>24*1024)throw new CollabError('SCREEN_INVALID','Invalid screen description',400);return {to:input.to,sessionId:input.sessionId,description:{type:description.type as 'offer'|'answer',sdp:description.sdp}};}
 if(input.candidate===null)return {to:input.to,sessionId:input.sessionId,candidate:null};
 const candidate=fields(input.candidate,['candidate','sdpMid','sdpMLineIndex','usernameFragment'],'SCREEN_INVALID');
 if(typeof candidate.candidate!=='string'||candidate.candidate.length>4096||[candidate.sdpMid,candidate.usernameFragment].some(v=>v!==undefined&&v!==null&&(typeof v!=='string'||v.length>256))||(candidate.sdpMLineIndex!==undefined&&candidate.sdpMLineIndex!==null&&(!Number.isInteger(candidate.sdpMLineIndex)||Number(candidate.sdpMLineIndex)<0||Number(candidate.sdpMLineIndex)>65535)))throw new CollabError('SCREEN_INVALID','Invalid screen candidate',400);
 return {to:input.to,sessionId:input.sessionId,candidate:{candidate:candidate.candidate,...(candidate.sdpMid!==undefined?{sdpMid:candidate.sdpMid}:{}),...(candidate.sdpMLineIndex!==undefined?{sdpMLineIndex:candidate.sdpMLineIndex}:{}),...(candidate.usernameFragment!==undefined?{usernameFragment:candidate.usernameFragment}:{})}};
}
export function parseScreenStop(value:unknown){const input=fields(value,['sessionId'],'SCREEN_INVALID');if(!sessionID(input.sessionId))throw new CollabError('SCREEN_INVALID','Invalid screen session',400);return {sessionId:input.sessionId};}
