import { parsePresentationAdmitted, parsePresentationState, parsePresentationWaiting, type NativeElement, type PresentationParticipant, type PresentationReaction } from '@collab/contracts';
import type { Control, TransportState } from './transport';

export type LivePresentationStatus='idle'|'starting'|'waiting'|'observer'|'host'|'audience'|'ended'|'error';
export type LivePresentationState={
 status:LivePresentationStatus;sessionId?:string;revision:number;hostConnectionId?:string;frameId:string|null;spotlight:string|null;locked:boolean;
 reactions:PresentationReaction[];pending:PresentationParticipant[];admitted:PresentationParticipant[];error?:string;reason?:string;
};
export type LivePresentationTransport={
 subscribeControl(listener:(control:Control)=>void):()=>void;subscribe(listener:()=>void):()=>void;getSnapshot():TransportState;
 sendControl(type:string,payload:Record<string,unknown>):string|undefined;
};
export type PresentationGrant={sessionId:string;token:string;expiresAt:string};
export type PresentationInvite={id:string;token:string;expiresAt:string};
export type LivePresentationAPI={
 create(requestId:string):Promise<PresentationGrant>;
 createLink(sessionId:string,requestId:string):Promise<PresentationInvite>;
 admit(sessionId:string,connectionId:string,expectedRevision:string):Promise<PresentationGrant>;
};
const initial=():LivePresentationState=>({status:'idle',revision:0,frameId:null,spotlight:null,locked:false,reactions:[],pending:[],admitted:[]});
const allowedReactions=new Set(['👏','❤️','👍','🎉','😂']);

export function presentationFrame(elements:NativeElement[],frameId:string|null):{kind:'none'}|{kind:'missing';frameId:string}|{kind:'frame';frameId:string;element:NativeElement}{
 if(!frameId)return {kind:'none'};
 const element=elements.find(item=>item.id===frameId&&item.type==='frame'&&!item.isDeleted);
 return element?{kind:'frame',frameId,element}:{kind:'missing',frameId};
}

export class LivePresentation {
 private state=initial();private listeners=new Set<()=>void>();private cleanups:(()=>void)[]=[];private reactionTimers=new Map<string,ReturnType<typeof setTimeout>>();
 private createRequestId?:string;private linkRequestId?:string;private startTimer?:ReturnType<typeof setTimeout>;
 constructor(private transport:LivePresentationTransport,private api?:LivePresentationAPI){}
 getSnapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>this.listeners.delete(listener)};
 start(){if(this.cleanups.length)return;this.cleanups=[this.transport.subscribeControl(this.control),this.transport.subscribe(this.transportChanged)];this.transportChanged();}
 stop(){for(const cleanup of this.cleanups.splice(0))cleanup();for(const timer of this.reactionTimers.values())clearTimeout(timer);this.reactionTimers.clear();this.clearStartTimer();this.state=initial();this.emit();}

 async beginHost(){
  if(!this.api)throw new Error('Presentation API is unavailable');
  this.set({...initial(),status:'starting'});this.createRequestId??=crypto.randomUUID();
  try{const grant=await this.api.create(this.createRequestId);if(!this.transport.sendControl('presentation_start',{token:grant.token}))throw new Error('Presentation socket is reconnecting');this.set({...this.state,sessionId:grant.sessionId});this.startTimer=setTimeout(()=>{this.startTimer=undefined;if(this.state.status==='starting')this.set({...this.state,status:'error',reason:'START_TIMEOUT',error:'The presentation did not start. Try again.'});},10_000);this.createRequestId=undefined;return grant;}
  catch(error){this.clearStartTimer();this.set({...this.state,status:'error',error:error instanceof Error?error.message:'Could not start presentation'});throw error;}
 }
 async createInvite(){
  if(!this.api||!this.state.sessionId||this.state.status!=='host')throw new Error('Start the live presentation first');
  this.linkRequestId??=crypto.randomUUID();try{const invite=await this.api.createLink(this.state.sessionId,this.linkRequestId);this.linkRequestId=undefined;return invite;}catch(error){this.set({...this.state,error:error instanceof Error?error.message:'Could not create invite'});throw error;}
 }
 async admit(connectionId:string){
  if(!this.api||!this.host())throw new Error('Only the current host may admit participants');
  const grant=await this.api.admit(this.state.sessionId!,connectionId,String(this.state.revision));
  if(!this.transport.sendControl('presentation_admit',{connectionId,token:grant.token,expectedRevision:String(this.state.revision)}))throw new Error('Presentation socket is reconnecting');
 }
 setFrame(frameId:string|null){return this.revisioned('presentation_set_frame',{frameId});}
 spotlight(connectionId:string|null){return this.revisioned('presentation_spotlight',{connectionId});}
 setLocked(locked:boolean){return this.revisioned('presentation_lock',{locked});}
 remove(connectionId:string){return this.revisioned('presentation_remove',{connectionId});}
 transfer(connectionId:string){return this.revisioned('presentation_transfer',{connectionId});}
 react(emoji:string){if(!['host','audience'].includes(this.state.status)||!this.state.sessionId||!allowedReactions.has(emoji))return false;return !!this.transport.sendControl('presentation_react',{sessionId:this.state.sessionId,emoji});}
 leave(){if(!['host','audience','waiting'].includes(this.state.status)||!this.state.sessionId)return false;const sent=this.transport.sendControl('presentation_leave',{sessionId:this.state.sessionId});if(sent)this.set({...this.state,status:'ended',reason:'LEFT'});return !!sent;}
 private host(){return this.state.status==='host'&&this.state.hostConnectionId===this.transport.getSnapshot().connectionId&&!!this.state.sessionId;}
 private revisioned(type:string,payload:Record<string,unknown>){if(!this.host())return false;return !!this.transport.sendControl(type,{sessionId:this.state.sessionId,...payload,expectedRevision:String(this.state.revision)});}
 private transportChanged=()=>{const current=this.transport.getSnapshot();if(current.status==='offline'&&this.state.status==='starting'){this.clearStartTimer();this.set({...this.state,status:'error',reason:'START_OFFLINE',locked:false,error:current.error??'Connection ended before the presentation started. Try again.'});}else if(current.status==='offline'&&this.state.status==='host')this.set({...this.state,status:'ended',reason:'HOST_OFFLINE',locked:false,error:current.error});else if(current.status==='offline'&&this.state.status==='audience')this.set({...this.state,status:'ended',reason:'AUDIENCE_OFFLINE',locked:false,error:'Connection ended. Rejoin through the waiting room.'});else if(current.status==='offline'&&this.state.status==='observer')this.set({...this.state,status:'ended',reason:'OBSERVER_OFFLINE',locked:false,error:current.error});else if(['revoked','degraded'].includes(current.status)&&['host','audience','waiting','starting','observer'].includes(this.state.status)){this.clearStartTimer();this.set({...this.state,status:'ended',reason:current.status.toUpperCase(),error:current.error});}};
 private control=(control:Control)=>{
  try{
   if(control.type==='presentation_state'){
    this.clearStartTimer();
    const value=parsePresentationState(control.payload),connectionId=this.transport.getSnapshot().connectionId,status=value.hostConnectionId===connectionId?'host':value.admitted.some(person=>person.connectionId===connectionId)?'audience':'observer';
    this.set({status,sessionId:value.sessionId,revision:value.revision,hostConnectionId:value.hostConnectionId,frameId:value.frameId,spotlight:value.spotlight,locked:value.locked,reactions:value.reactions.filter(item=>item.expiresAt>Date.now()),pending:value.pending,admitted:value.admitted,error:undefined});
    for(const reaction of value.reactions)this.expire(reaction);return;
   }
   if(control.type==='presentation_waiting'){const value=parsePresentationWaiting(control.payload);this.set({...initial(),status:'waiting',sessionId:value.sessionId});return;}
   if(control.type==='presentation_admitted'){const value=parsePresentationAdmitted(control.payload);this.set({...this.state,status:'audience',sessionId:value.sessionId,error:undefined});return;}
   if(control.type==='presentation_reaction'){
    const value=control.payload;if(typeof value.sessionId!=='string'||value.sessionId!==this.state.sessionId||typeof value.id!=='string'||typeof value.connectionId!=='string'||typeof value.emoji!=='string'||!allowedReactions.has(value.emoji)||!Number.isSafeInteger(value.expiresAt)||Number(value.expiresAt)<=Date.now())return;
    const reaction={id:value.id,connectionId:value.connectionId,emoji:value.emoji,expiresAt:Number(value.expiresAt)} as PresentationReaction;
    this.set({...this.state,reactions:[...this.state.reactions.filter(item=>item.id!==reaction.id&&item.expiresAt>Date.now()),reaction]});this.expire(reaction);return;
   }
   if(control.type==='presentation_stop'&&control.payload.sessionId===this.state.sessionId){this.set({...this.state,status:'ended',reason:String(control.payload.reason??'ENDED'),locked:false});return;}
   if(control.type==='error'&&String(control.payload.code).startsWith('PRESENTATION_')){if(this.state.status==='starting')this.clearStartTimer();this.set({...this.state,status:this.state.status==='starting'?'error':this.state.status,error:String(control.payload.message??control.payload.code)});}
  }catch{this.set({...this.state,status:'error',error:'Invalid presentation state received'});}
 };
 private expire(reaction:PresentationReaction){clearTimeout(this.reactionTimers.get(reaction.id));const timer=setTimeout(()=>{this.reactionTimers.delete(reaction.id);this.set({...this.state,reactions:this.state.reactions.filter(item=>item.id!==reaction.id)});},Math.max(0,reaction.expiresAt-Date.now()));this.reactionTimers.set(reaction.id,timer);}
 private clearStartTimer(){if(this.startTimer)clearTimeout(this.startTimer);this.startTimer=undefined;}
 private set(state:LivePresentationState){this.state=state;this.emit();}
 private emit(){for(const listener of this.listeners)listener();}
}
