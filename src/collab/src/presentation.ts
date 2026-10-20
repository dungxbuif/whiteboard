import {CollabError,revision} from './auth.js';

export const PRESENTATION_WAITING_LIMIT=100;
export type PresentationParticipant={connectionId:string;principalId:string;displayName:string};
export type PresentationState={room:string;sessionId:string;hostConnectionId:string;hostPrincipalId:string;revision:number;pending:Map<string,PresentationParticipant>;admitted:Map<string,PresentationParticipant>};

const id=(value:unknown):value is string=>typeof value==='string'&&/^[0-9a-f-]{36}$/.test(value);
export class PresentationSessions {
 private sessions=new Map<string,PresentationState>();
 start(room:string,sessionId:string,hostConnectionId:string,hostPrincipalId:string){
  if(!id(sessionId)||!hostConnectionId||!hostPrincipalId)throw new CollabError('PRESENTATION_INVALID','Invalid presentation session',400);
  const previous=this.sessions.get(room);if(previous)throw new CollabError('PRESENTATION_ACTIVE','A presentation is already active',409);
  const state={room,sessionId,hostConnectionId,hostPrincipalId,revision:1,pending:new Map(),admitted:new Map()};this.sessions.set(room,state);return this.snapshot(state);
 }
 wait(room:string,sessionId:string,participant:PresentationParticipant){
  const state=this.sessions.get(room);if(!state||state.sessionId!==sessionId)throw new CollabError('PRESENTATION_ENDED','Presentation has ended',409);
  if(participant.connectionId===state.hostConnectionId||!participant.connectionId||!participant.principalId||!participant.displayName)throw new CollabError('PRESENTATION_INVALID','Invalid waiting participant',400);
  if(!state.pending.has(participant.connectionId)&&state.pending.size>=PRESENTATION_WAITING_LIMIT)throw new CollabError('PRESENTATION_FULL','Waiting room is full',409);
  state.pending.set(participant.connectionId,{...participant});return {sessionId,status:'pending' as const,connectionId:participant.connectionId};
 }
 admit(room:string,sessionId:string,hostConnectionId:string,connectionId:string,expectedRevision:number){
  const state=this.host(room,sessionId,hostConnectionId,expectedRevision),participant=state.pending.get(connectionId);if(!participant)throw new CollabError('PRESENTATION_TARGET','Waiting participant is unavailable',404);
  state.pending.delete(connectionId);state.admitted.set(connectionId,participant);state.revision++;return {state:this.snapshot(state),participant:{...participant}};
 }
 remove(room:string,sessionId:string,hostConnectionId:string,connectionId:string,expectedRevision:number){
  const state=this.host(room,sessionId,hostConnectionId,expectedRevision);if(!state.pending.delete(connectionId)&&!state.admitted.delete(connectionId))throw new CollabError('PRESENTATION_TARGET','Participant is unavailable',404);state.revision++;return this.snapshot(state);
 }
 private host(room:string,sessionId:string,host:string,expected:number){const state=this.sessions.get(room);if(!state||state.sessionId!==sessionId)throw new CollabError('PRESENTATION_ENDED','Presentation has ended',409);if(state.hostConnectionId!==host)throw new CollabError('PRESENTATION_FORBIDDEN','Only the current host may control presentation',403);if(!Number.isInteger(expected)||expected!==state.revision)throw new CollabError('PRESENTATION_CONFLICT','Presentation revision changed',409);return state;}
 state(room:string){const state=this.sessions.get(room);return state?this.snapshot(state):undefined}
 active(room:string,sessionId:string){const state=this.sessions.get(room);return !!state&&state.sessionId===sessionId}
 disconnect(connectionId:string){for(const [room,state] of this.sessions){if(state.hostConnectionId===connectionId){this.sessions.delete(room);return {ended:this.snapshot(state),participant:false}}if(state.pending.delete(connectionId)||state.admitted.delete(connectionId)){state.revision++;return {state:this.snapshot(state),participant:true}}}return undefined}
 stop(room:string,sessionId:string,hostConnectionId:string){const state=this.host(room,sessionId,hostConnectionId,this.sessions.get(room)?.revision??0);this.sessions.delete(room);return this.snapshot(state)}
 private snapshot(state:PresentationState){return {sessionId:state.sessionId,hostConnectionId:state.hostConnectionId,hostPrincipalId:state.hostPrincipalId,revision:state.revision,pending:[...state.pending.values()].map(v=>({...v})),admitted:[...state.admitted.values()].map(v=>({...v}))};}
}

export function parsePresentationStart(value:unknown){if(!value||typeof value!=='object'||Array.isArray(value))throw new CollabError('PRESENTATION_INVALID','Invalid presentation control',400);const input=value as Record<string,unknown>;if(Object.keys(input).some(k=>!['token'].includes(k))||typeof input.token!=='string'||input.token.length>16384)throw new CollabError('PRESENTATION_INVALID','Invalid presentation control',400);return {token:input.token};}
export function parsePresentationAdmit(value:unknown){if(!value||typeof value!=='object'||Array.isArray(value))throw new CollabError('PRESENTATION_INVALID','Invalid presentation control',400);const input=value as Record<string,unknown>;if(Object.keys(input).some(k=>!['connectionId','token','expectedRevision'].includes(k))||typeof input.connectionId!=='string'||input.connectionId.length>128||typeof input.token!=='string'||input.token.length>16384||!revision(input.expectedRevision)||input.expectedRevision==='0')throw new CollabError('PRESENTATION_INVALID','Invalid presentation control',400);return {connectionId:input.connectionId,token:input.token,expectedRevision:Number(input.expectedRevision)};}
