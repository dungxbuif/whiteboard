import {CollabError} from './auth.js';
import {fields} from './awareness.js';
export const MAX_VOICE_PARTICIPANTS=10;
const MAX_SDP_BYTES=24*1024;
const MAX_CANDIDATE_LENGTH=4096;
export type VoiceAdmission={admitted:boolean;connectionId:string;peers:string[];reason?:'VOICE_FULL'};
export class Voice {
 private members=new Map<string,{room:string;muted:boolean}>();
 joinVoice(room:string,connectionId:string):VoiceAdmission {
  const peers=[...this.members].filter(([id,member])=>member.room===room&&id!==connectionId).map(([id])=>id);
  if(!this.members.has(connectionId)&&peers.length>=MAX_VOICE_PARTICIPANTS)return {admitted:false,connectionId,peers:[],reason:'VOICE_FULL'};
  if(!this.members.has(connectionId))this.members.set(connectionId,{room,muted:false});
  return {admitted:true,connectionId,peers};
 }
 leaveVoice(connectionId:string){this.members.delete(connectionId);}
 state(connectionId:string){const member=this.members.get(connectionId);return {voiceJoined:!!member,voiceMuted:member?.muted??false};}
 mute(connectionId:string,muted:unknown){const member=this.members.get(connectionId);if(!member)throw new CollabError('VOICE_NOT_JOINED');if(typeof muted!=='boolean')throw new CollabError('VOICE_INVALID','Invalid mute state',400);member.muted=muted;}
 canSignal(room:string,from:string,to:string){return from!==to&&this.members.get(from)?.room===room&&this.members.get(to)?.room===room;}
}
export function parseSignal(value:unknown) {
 const input=fields(value,['to','description','candidate'],'VOICE_INVALID');
 if(typeof input.to!=='string'||input.to.length>128||(!('description'in input)===!('candidate'in input)))throw new CollabError('VOICE_INVALID','Invalid signal',400);
 if('description'in input){
  const description=fields(input.description,['type','sdp'],'VOICE_INVALID');
  if(!['offer','answer'].includes(String(description.type))||typeof description.sdp!=='string'||Buffer.byteLength(description.sdp)>MAX_SDP_BYTES)throw new CollabError('VOICE_INVALID','Invalid session description',400);
  return {to:input.to,description:{type:description.type as 'offer'|'answer',sdp:description.sdp}};
 }
 if(input.candidate===null)return {to:input.to,candidate:null};
 const candidate=fields(input.candidate,['candidate','sdpMid','sdpMLineIndex','usernameFragment'],'VOICE_INVALID');
 if(typeof candidate.candidate!=='string'||candidate.candidate.length>MAX_CANDIDATE_LENGTH||[candidate.sdpMid,candidate.usernameFragment].some(v=>v!==undefined&&v!==null&&(typeof v!=='string'||v.length>256))||(candidate.sdpMLineIndex!==undefined&&candidate.sdpMLineIndex!==null&&(!Number.isInteger(candidate.sdpMLineIndex)||Number(candidate.sdpMLineIndex)<0||Number(candidate.sdpMLineIndex)>65535)))throw new CollabError('VOICE_INVALID','Invalid ICE candidate',400);
 return {to:input.to,candidate:{candidate:candidate.candidate,...(candidate.sdpMid!==undefined?{sdpMid:candidate.sdpMid}:{}),...(candidate.sdpMLineIndex!==undefined?{sdpMLineIndex:candidate.sdpMLineIndex}:{}),...(candidate.usernameFragment!==undefined?{usernameFragment:candidate.usernameFragment}:{})}};
}
