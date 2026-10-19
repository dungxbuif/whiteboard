import type {Control} from './transport';
import type {Presence,PresenceTransport} from './presence';
export type VoiceState={status:'idle'|'admitting'|'permission'|'connected'|'error';muted:boolean;error?:string;audioBlocked:boolean};
export type VoiceDependencies={config:()=>Promise<RTCConfiguration&{expiresAt:string}>;media:()=>Promise<MediaStream>;peer:(config:RTCConfiguration)=>RTCPeerConnection;audio:()=>HTMLAudioElement};
type Peer={pc:RTCPeerConnection;ice:(RTCIceCandidateInit|null)[];chain:Promise<void>;audio?:HTMLAudioElement;offered:boolean};
const MAX_PEERS=9,MAX_PENDING_SIGNALS=128,MAX_EARLY_ICE=32,JOIN_TIMEOUT_MS=15000,MEDIA_TIMEOUT_MS=60000;
const initial=():VoiceState=>({status:'idle',muted:false,audioBlocked:false,error:undefined});
export class Voice {
 private seenJoined=false;private state=initial();private listeners=new Set<()=>void>();private cleanups:(()=>void)[]=[];private generation=0;private connectionId?:string;private requestId?:string;private stream?:MediaStream;private configuration:RTCConfiguration={};private peers=new Map<string,Peer>();private pending:Control[]=[];private timer?:ReturnType<typeof setTimeout>;private renewal?:ReturnType<typeof setTimeout>;
 private deps:VoiceDependencies;
 constructor(private transport:PresenceTransport,private presence:Presence,dependencies:Partial<VoiceDependencies>&Pick<VoiceDependencies,'config'>){this.deps={media:()=>navigator.mediaDevices.getUserMedia({audio:true}),peer:config=>new RTCPeerConnection(config),audio:()=>new Audio(),...dependencies};}
 getSnapshot=()=>this.state;
 subscribe=(f:()=>void)=>{this.listeners.add(f);return()=>{this.listeners.delete(f);};};
 private set(update:Partial<VoiceState>){this.state={...this.state,...update};for(const f of this.listeners)f();}
 start(){if(this.cleanups.length)return;this.cleanups=[this.transport.subscribe(this.changed),this.transport.subscribeControl(this.control),this.presence.subscribe(this.roster)];}
 stop(){this.leave();for(const f of this.cleanups)f();this.cleanups=[];}
 join(){
  if(!['idle','error'].includes(this.state.status))return;
  const id=this.transport.getSnapshot().connectionId;if(!id){this.set({status:'error',error:'Connect to the scene before joining voice.'});return;}
  this.connectionId=id;this.generation++;this.requestId=this.transport.sendControl('voice_join',{});
  if(!this.requestId){this.set({status:'error',error:'Voice is unavailable while the scene reconnects.'});return;}
  this.set({status:'admitting',error:undefined,muted:false});this.timer=setTimeout(()=>this.fail('Voice admission timed out. Try again.'),JOIN_TIMEOUT_MS);
 }
 leave(){if(this.connectionId===this.transport.getSnapshot().connectionId&&['admitting','permission','connected'].includes(this.state.status))this.transport.sendControl('voice_leave',{});this.clear();this.set(initial());}
 private clear(){this.seenJoined=false;this.generation++;clearTimeout(this.timer);clearTimeout(this.renewal);this.requestId=undefined;this.connectionId=undefined;this.pending=[];for(const id of [...this.peers.keys()])this.removePeer(id);for(const track of this.stream?.getTracks()??[]){track.removeEventListener('ended',this.ended);track.stop();}this.stream=undefined;}
 private fail(message:string){this.leave();this.set({status:'error',error:message});}
 private ended=()=>this.fail('Microphone stopped. Join voice to reconnect.');
 private changed=()=>{if(this.connectionId&&(this.transport.getSnapshot().connectionId!==this.connectionId||['offline','revoked','degraded','legacy'].includes(this.transport.getSnapshot().status))){this.clear();this.set(initial());}};
 private current(generation:number){return generation===this.generation&&!!this.connectionId&&this.transport.getSnapshot().connectionId===this.connectionId;}
 private control=(control:Control)=>{
  const p=control.payload;
  if(control.type==='voice_admission'&&control.requestId===this.requestId&&this.state.status==='admitting'){
   clearTimeout(this.timer);this.requestId=undefined;
   if(p.admitted!==true||p.connectionId!==this.connectionId){this.clear();this.set({status:'error',error:p.reason==='VOICE_FULL'?'Voice is full (10 people). You can keep editing.':'Voice admission failed. You can keep editing.'});return;}
   this.set({status:'permission'});const generation=this.generation;this.timer=setTimeout(()=>this.fail('Microphone permission timed out. You can keep editing.'),MEDIA_TIMEOUT_MS);void this.activate(generation);return;
  }
  if(control.type==='voice_signal'&&p.to===this.connectionId&&typeof p.from==='string'&&['permission','connected'].includes(this.state.status)){
   if(!this.stream||!this.presence.getSnapshot().participants.some(v=>v.connectionId===p.from&&v.voiceJoined)){if(this.pending.length<MAX_PENDING_SIGNALS)this.pending.push(control);else this.fail('Too many pending voice signals. Join again.');return;}
   this.signal(control);return;
  }
  if(control.type==='error'&&String(p.code).startsWith('VOICE_')){
   if(control.requestId===this.requestId){this.fail('Voice could not start. Try again.');return;}
   if(this.state.status==='connected')this.set({error:'A voice connection could not complete. Leave and join again to retry.'});
  }
 };
 private async activate(generation:number){
  try{
   const config=await this.deps.config();if(!this.current(generation))return;this.configure(config,generation);
   const stream=await this.deps.media();if(!this.current(generation)){stream.getTracks().forEach(track=>track.stop());return;}
   this.stream=stream;for(const track of stream.getTracks())track.addEventListener('ended',this.ended);
   clearTimeout(this.timer);this.set({status:'connected',muted:false,error:undefined});this.roster();
  }catch(error){if(this.current(generation))this.fail(error instanceof DOMException&&error.name==='NotAllowedError'?'Microphone permission was denied. You can keep editing.':'Voice could not start. Check microphone permissions and try again.');}
 }
 private configure(config:RTCConfiguration&{expiresAt:string},generation:number){
  const remaining=Date.parse(config.expiresAt)-Date.now();if(!Number.isFinite(remaining)||remaining<=0)throw new Error('Expired ICE configuration');
  this.configuration={iceServers:config.iceServers??[]};for(const peer of this.peers.values())peer.pc.setConfiguration(this.configuration);
  clearTimeout(this.renewal);this.renewal=setTimeout(()=>{void this.deps.config().then(next=>{if(this.current(generation))this.configure(next,generation);}).catch(()=>{if(this.current(generation))this.fail('Voice credentials could not refresh. Join again when connected.');});},Math.max(1000,remaining>120000?remaining-60000:remaining*0.8));
 }
 private roster=()=>{
  const participants=this.presence.getSnapshot().participants;const self=participants.find(v=>v.connectionId===this.connectionId);
  if(self?.voiceJoined)this.seenJoined=true;
  if(!this.stream||this.state.status!=='connected')return;
  // Admission can precede the next roster; the previous roster is not a revocation.
  if(this.seenJoined&&self&&!self.voiceJoined){this.clear();this.set(initial());return;}
  const active=new Set(participants.filter(v=>v.voiceJoined&&v.connectionId!==this.connectionId).map(v=>v.connectionId));
  for(const id of this.peers.keys())if(!active.has(id))this.removePeer(id);
  for(const id of active)if(this.peers.size<MAX_PEERS||this.peers.has(id))this.ensurePeer(id);
  const ready=this.pending.filter(c=>active.has(String(c.payload.from)));this.pending=this.pending.filter(c=>!active.has(String(c.payload.from)));for(const c of ready)this.signal(c);
 };
 private ensurePeer(id:string):Peer|undefined{
  const existing=this.peers.get(id);if(existing)return existing;if(!this.stream||this.peers.size>=MAX_PEERS)return;
  const pc=this.deps.peer(this.configuration),peer:Peer={pc,ice:[],chain:Promise.resolve(),offered:false};this.peers.set(id,peer);const generation=this.generation;
  const alive=()=>this.current(generation)&&this.peers.get(id)===peer;
  pc.onicecandidate=event=>{if(alive())this.transport.sendControl('voice_signal',{to:id,candidate:event.candidate?.toJSON()??null});};
  pc.ontrack=event=>{if(!alive())return;const audio=peer.audio??this.deps.audio();peer.audio=audio;audio.autoplay=true;audio.srcObject=event.streams[0]??new MediaStream([event.track]);void audio.play().catch(()=>{if(alive())this.set({audioBlocked:true});});};
  pc.onconnectionstatechange=()=>{if(alive()&&pc.connectionState==='failed'){this.removePeer(id);this.set({error:'A voice connection failed. Leave and join again to retry.'});}};
  for(const track of this.stream.getTracks())pc.addTrack(track,this.stream);
  // Only one side creates the initial offer; repeated rosters never renegotiate.
  if(this.connectionId! < id){peer.offered=true;this.queue(id,peer,async()=>{if(pc.signalingState!=='stable')return;const offer=await pc.createOffer();if(!alive())return;await pc.setLocalDescription(offer);if(alive())this.transport.sendControl('voice_signal',{to:id,description:pc.localDescription?.toJSON?.()??offer});});}
  return peer;
 }
 private queue(id:string,peer:Peer,operation:()=>Promise<void>){const generation=this.generation;peer.chain=peer.chain.then(async()=>{if(this.current(generation)&&this.peers.get(id)===peer)await operation();}).catch(()=>{if(this.current(generation)&&this.peers.get(id)===peer){this.removePeer(id);this.set({error:'A voice connection could not complete. Leave and join again to retry.'});}});}
 private signal(control:Control){
  const p=control.payload,id=String(p.from);if(id===this.connectionId)return;const peer=this.ensurePeer(id);if(!peer)return;const generation=this.generation;
  this.queue(id,peer,async()=>{
   const alive=()=>this.current(generation)&&this.peers.get(id)===peer;
   if(p.description){const description=p.description as RTCSessionDescriptionInit;
    if(description.type==='offer'&&peer.pc.signalingState!=='stable'){
     if(this.connectionId! < id)return; // impolite deterministic offerer ignores glare
     await peer.pc.setLocalDescription({type:'rollback'});if(!alive())return;
    }
    await peer.pc.setRemoteDescription(description);if(!alive())return;
    for(const candidate of peer.ice.splice(0)){await peer.pc.addIceCandidate(candidate??undefined);if(!alive())return;}
    if(description.type==='offer'){const answer=await peer.pc.createAnswer();if(!alive())return;await peer.pc.setLocalDescription(answer);if(alive())this.transport.sendControl('voice_signal',{to:id,description:peer.pc.localDescription?.toJSON?.()??answer});}
   }else if('candidate'in p){const candidate=p.candidate as RTCIceCandidateInit|null;if(!peer.pc.remoteDescription){if(peer.ice.length>=MAX_EARLY_ICE)throw new Error('ICE buffer full');peer.ice.push(candidate);}else await peer.pc.addIceCandidate(candidate??undefined);}
  });
 }
 private removePeer(id:string){const peer=this.peers.get(id);if(!peer)return;this.peers.delete(id);peer.pc.onicecandidate=null;peer.pc.ontrack=null;peer.pc.onconnectionstatechange=null;peer.pc.onnegotiationneeded=null;peer.pc.close();if(peer.audio){peer.audio.pause();peer.audio.srcObject=null;peer.audio.remove();}peer.ice=[];}
 toggleMute(){if(this.state.status!=='connected'||!this.stream)return;const muted=!this.state.muted;for(const track of this.stream.getAudioTracks())track.enabled=!muted;this.transport.sendControl('voice_mute',{muted});this.set({muted});}
 async resumeAudio(){let blocked=false;await Promise.all([...this.peers.values()].map(async p=>{try{await p.audio?.play();}catch{blocked=true;}}));this.set({audioBlocked:blocked});}
}
