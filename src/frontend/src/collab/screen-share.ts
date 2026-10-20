import type {Control} from './transport';
import type {Presence,PresenceTransport} from './presence';
import {parseScreenAdmission} from '@collab/contracts';

export type ScreenShareState={status:'idle'|'admitting'|'ready'|'permission'|'sharing'|'receiving'|'error';publisher?:string;receiving:boolean;error?:string};
export type ScreenShareDependencies={config:()=>Promise<RTCConfiguration&{expiresAt:string}>;media:(constraints:DisplayMediaStreamOptions)=>Promise<MediaStream>;peer:(configuration:RTCConfiguration)=>RTCPeerConnection};
type Peer={pc:RTCPeerConnection;ice:(RTCIceCandidateInit|null)[];chain:Promise<void>;offered:boolean};
const initial=():ScreenShareState=>({status:'idle',publisher:undefined,receiving:false,error:undefined});
const constraints:DisplayMediaStreamOptions={video:{width:{max:1280},height:{max:720},frameRate:{max:15}},audio:false};
const MAX_PEERS=9,MAX_ICE=32;

export class ScreenShare {
 private state=initial();private listeners=new Set<()=>void>();private cleanups:(()=>void)[]=[];private generation=0;private connectionId?:string;private requestId?:string;private activationRequestId?:string;private sessionId?:string;private stream?:MediaStream;private remoteStream?:MediaStream;private video?:HTMLVideoElement;private peers=new Map<string,Peer>();private waiting=new Set<string>();private configuration:RTCConfiguration={};private resolveJoin?:()=>void;private rejectJoin?:(error:Error)=>void;private resolveActivation?:()=>void;
 private deps:ScreenShareDependencies;
 constructor(private transport:PresenceTransport,private presence:Presence,dependencies:Partial<ScreenShareDependencies>&Pick<ScreenShareDependencies,'config'>){this.deps={media:options=>{if(!navigator.mediaDevices?.getDisplayMedia)return Promise.reject(new Error('Screen capture is unsupported'));return navigator.mediaDevices.getDisplayMedia(options);},peer:configuration=>new RTCPeerConnection(configuration),...dependencies};}
 getSnapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>this.listeners.delete(listener);};
 private set(update:Partial<ScreenShareState>){this.state={...this.state,...update};for(const listener of this.listeners)listener();}
 start(){if(this.cleanups.length)return;this.cleanups=[this.transport.subscribe(this.changed),this.transport.subscribeControl(this.control),this.presence.subscribe(this.roster)];this.roster();}
 stop(){this.leave();for(const cleanup of this.cleanups)cleanup();this.cleanups=[];}
 join():Promise<void>{
  if(!['idle','error'].includes(this.state.status))return Promise.resolve();const connectionId=this.transport.getSnapshot().connectionId;if(!connectionId){this.set({status:'error',error:'Connect to the scene before sharing.'});return Promise.reject(new Error('Screen sharing unavailable'));}
  const requestId=this.transport.sendControl('screen_request',{});if(!requestId){this.set({status:'error',error:'Screen sharing is unavailable while reconnecting.'});return Promise.reject(new Error('Screen sharing unavailable'));}
  this.clear(false);this.connectionId=connectionId;this.requestId=requestId;this.set({status:'admitting',receiving:false,error:undefined});
  return new Promise<void>((resolve,reject)=>{this.resolveJoin=resolve;this.rejectJoin=reject;});
 }
 async startSharing():Promise<void>{
  if(this.state.status!=='ready'||!this.sessionId||!this.connectionId)return;const generation=this.generation;this.set({status:'permission',error:undefined});
  // Capture must begin synchronously in this user event before awaiting TURN.
  let captured:MediaStream|undefined;const capture=this.deps.media(constraints).then(stream=>{captured=stream;if(!this.current(generation))stream.getTracks().forEach(track=>track.stop());return stream;}),configuration=this.deps.config();
  try{
   const [stream,config]=await Promise.all([capture,configuration]);if(!this.current(generation)){stream.getTracks().forEach(track=>track.stop());return;}
   const video=stream.getVideoTracks()[0];for(const track of stream.getAudioTracks())track.stop();if(!video)throw new Error('No screen video track');
   this.configuration={iceServers:config.iceServers??[]};this.stream=stream;video.addEventListener('ended',this.ended);const requestId=this.transport.sendControl('screen_request',{sessionId:this.sessionId});if(!requestId){this.fail('Screen sharing is unavailable while reconnecting.');return;}this.activationRequestId=requestId;
   await new Promise<void>(resolve=>this.resolveActivation=resolve);
  }catch(error){captured?.getTracks().forEach(track=>track.stop());if(this.current(generation))this.fail(error instanceof DOMException&&error.name==='NotAllowedError'?'Screen selection was cancelled. You can keep drawing.':'Screen sharing could not start. You can keep drawing.');}
 }
 leave(){if(this.sessionId&&this.connectionId===this.transport.getSnapshot().connectionId)this.transport.sendControl('screen_stop',{sessionId:this.sessionId});this.clear();this.set(initial());}
 attachVideo(element:HTMLVideoElement|null){this.video=element??undefined;if(this.video){this.video.autoplay=true;this.video.playsInline=true;this.video.srcObject=this.remoteStream??null;if(this.remoteStream)void this.video.play().catch(()=>{});}}
 private current(generation:number){return generation===this.generation&&!!this.connectionId&&this.connectionId===this.transport.getSnapshot().connectionId;}
 private clear(increment=true){if(increment)this.generation++;this.rejectJoin?.(new Error('Screen admission cancelled'));this.resolveJoin=undefined;this.rejectJoin=undefined;this.requestId=undefined;this.activationRequestId=undefined;this.resolveActivation?.();this.resolveActivation=undefined;this.sessionId=undefined;this.connectionId=undefined;this.waiting.clear();for(const id of [...this.peers.keys()])this.removePeer(id);for(const track of this.stream?.getTracks()??[]){track.removeEventListener('ended',this.ended);track.stop();}for(const track of this.remoteStream?.getTracks()??[])track.stop();this.stream=undefined;this.remoteStream=undefined;if(this.video){this.video.pause();this.video.srcObject=null;}}
 private fail(message:string){const sessionId=this.sessionId,connectionId=this.connectionId;if(sessionId&&connectionId===this.transport.getSnapshot().connectionId)this.transport.sendControl('screen_stop',{sessionId});this.clear();this.set({status:'error',receiving:false,error:message,publisher:undefined});}
 private ended=()=>this.fail('Screen sharing stopped. You can keep drawing.');
 private changed=()=>{if(this.connectionId&&(this.connectionId!==this.transport.getSnapshot().connectionId||['offline','revoked','degraded','legacy'].includes(this.transport.getSnapshot().status))){this.clear();this.set(initial());}};
 private control=(control:Control)=>{
  const payload=control.payload;
  if(control.type==='screen_admission'){
   let admission;try{admission=parseScreenAdmission(payload);}catch{if(control.requestId===this.requestId){const error='Screen admission returned invalid routing data. You can keep drawing.';this.rejectJoin?.(new Error(error));this.resolveJoin=undefined;this.rejectJoin=undefined;this.clear();this.set({status:'error',receiving:false,error,publisher:undefined});}else if(control.requestId===this.activationRequestId)this.fail('Screen activation returned invalid routing data. You can keep drawing.');return;}
   if(control.requestId===this.requestId){
    this.requestId=undefined;if(!admission.admitted||admission.connectionId!==this.connectionId){const error=admission.reason==='MEDIA_FULL'?'Media is full (10 people). You can keep drawing.':'Screen admission failed. You can keep drawing.';this.rejectJoin?.(new Error(error));this.resolveJoin=undefined;this.rejectJoin=undefined;this.clear();this.set({status:'error',receiving:false,error});return;}
    this.sessionId=admission.sessionId;for(const peer of admission.peers)if(peer!==this.connectionId)this.waiting.add(peer);this.set({status:admission.receiving?'receiving':'ready',publisher:admission.publisher,receiving:admission.receiving??false,error:undefined});this.resolveJoin?.();this.resolveJoin=undefined;this.rejectJoin=undefined;return;
   }
   if(control.requestId===this.activationRequestId){
    if(!admission.admitted||admission.connectionId!==this.connectionId||admission.publisher!==this.connectionId||admission.receiving||admission.sessionId!==this.sessionId){this.fail('Screen activation failed. You can keep drawing.');return;}
    this.activationRequestId=undefined;for(const peer of admission.peers)if(peer!==this.connectionId)this.waiting.add(peer);this.set({status:'sharing',publisher:this.connectionId,receiving:false,error:undefined});for(const peer of this.waiting)this.ensurePeer(peer,true);this.waiting.clear();this.resolveActivation?.();this.resolveActivation=undefined;return;
   }
   if(admission.admitted&&admission.receiving&&admission.publisher===this.connectionId&&admission.sessionId===this.sessionId&&admission.connectionId!==this.connectionId){if(this.state.status==='sharing')this.ensurePeer(admission.connectionId,true);else if(['ready','permission'].includes(this.state.status))this.waiting.add(admission.connectionId);return;}
  }
  if(control.type==='screen_signal'&&payload.to===this.connectionId&&payload.sessionId===this.sessionId&&typeof payload.from==='string')this.signal(control);
  else if(control.type==='screen_stop'&&payload.sessionId===this.sessionId){if(typeof payload.connectionId==='string'&&this.state.status==='sharing')this.removePeer(payload.connectionId);else {const expired=payload.reason==='RESERVATION_EXPIRED';this.clear();this.set(expired?{status:'error',receiving:false,error:'Screen reservation expired. Request it again.',publisher:undefined}:initial());}}
  else if(control.type==='error'&&String(payload.code).startsWith('SCREEN_')){if(control.requestId===this.requestId)this.fail('Screen admission failed. You can keep drawing.');else if(control.requestId===this.activationRequestId)this.fail('Screen activation failed. You can keep drawing.');else this.set({error:'A screen connection could not complete. You can keep drawing.'});}
 };
 private roster=()=>{
  const self=this.transport.getSnapshot().connectionId;if(!self)return;const publisher=this.presence.getSnapshot().participants.find(participant=>participant.screenPublisher&&participant.connectionId!==self);
  if(publisher&&!this.sessionId&&!this.requestId&&['idle','error'].includes(this.state.status)){this.clear();this.connectionId=self;this.set({status:'admitting',publisher:publisher.connectionId,receiving:true,error:undefined});void this.requestReceiver(this.generation,publisher.connectionId);}
  if(this.state.receiving&&this.state.publisher&&!this.presence.getSnapshot().participants.some(participant=>participant.connectionId===this.state.publisher&&participant.screenPublisher)){this.clear();this.set(initial());}
 };
 private async requestReceiver(generation:number,publisher:string){
  try{const config=await this.deps.config();if(!this.current(generation)||!this.presence.getSnapshot().participants.some(participant=>participant.connectionId===publisher&&participant.screenPublisher))return;this.configuration={iceServers:config.iceServers??[]};this.requestId=this.transport.sendControl('screen_request',{});if(!this.requestId)this.fail('Screen sharing is unavailable while reconnecting.');}
  catch{if(this.current(generation))this.fail('Screen media configuration is unavailable. You can keep drawing.');}
 }
 private ensurePeer(id:string,offer:boolean){
  const existing=this.peers.get(id);if(existing)return existing;if(this.peers.size>=MAX_PEERS)return;if(offer&&!this.stream){this.waiting.add(id);return;}
  const pc=this.deps.peer(this.configuration),peer:Peer={pc,ice:[],chain:Promise.resolve(),offered:false};this.peers.set(id,peer);const generation=this.generation,alive=()=>this.current(generation)&&this.peers.get(id)===peer;
  pc.onicecandidate=event=>{if(alive()&&this.sessionId)this.transport.sendControl('screen_signal',{to:id,sessionId:this.sessionId,candidate:event.candidate?.toJSON()??null});};
  pc.ontrack=event=>{if(!alive())return;this.remoteStream=event.streams[0]??new MediaStream([event.track]);if(this.video){this.video.srcObject=this.remoteStream;void this.video.play().catch(()=>{});}};
  pc.onconnectionstatechange=()=>{if(alive()&&pc.connectionState==='failed'){this.removePeer(id);this.set({error:'A screen connection failed. You can keep drawing.'});}};
  if(offer&&this.stream){for(const track of this.stream.getVideoTracks())pc.addTrack(track,this.stream);peer.offered=true;this.queue(id,peer,async()=>{const description=await pc.createOffer();if(!alive())return;await pc.setLocalDescription(description);if(alive()&&this.sessionId)this.transport.sendControl('screen_signal',{to:id,sessionId:this.sessionId,description:pc.localDescription?.toJSON?.()??description});});}
  return peer;
 }
 private signal(control:Control){const payload=control.payload,id=String(payload.from);if(id===this.connectionId)return;const peer=this.ensurePeer(id,false);if(!peer)return;this.queue(id,peer,async()=>{if(payload.description){const description=payload.description as RTCSessionDescriptionInit;await peer.pc.setRemoteDescription(description);for(const candidate of peer.ice.splice(0))await peer.pc.addIceCandidate(candidate??undefined);if(description.type==='offer'){const answer=await peer.pc.createAnswer();await peer.pc.setLocalDescription(answer);if(this.sessionId)this.transport.sendControl('screen_signal',{to:id,sessionId:this.sessionId,description:peer.pc.localDescription?.toJSON?.()??answer});}}else if('candidate'in payload){const candidate=payload.candidate as RTCIceCandidateInit|null;if(!peer.pc.remoteDescription){if(peer.ice.length>=MAX_ICE)throw new Error('Screen ICE buffer full');peer.ice.push(candidate);}else await peer.pc.addIceCandidate(candidate??undefined);}});}
 private queue(id:string,peer:Peer,operation:()=>Promise<void>){const generation=this.generation;peer.chain=peer.chain.then(async()=>{if(this.current(generation)&&this.peers.get(id)===peer)await operation();}).catch(()=>{if(this.current(generation)&&this.peers.get(id)===peer){this.removePeer(id);this.set({error:'A screen connection could not complete. You can keep drawing.'});}});}
 private removePeer(id:string){const peer=this.peers.get(id);if(!peer)return;this.peers.delete(id);peer.pc.onicecandidate=null;peer.pc.ontrack=null;peer.pc.onconnectionstatechange=null;peer.pc.close();peer.ice=[];}
}
