import {afterEach,expect,it,vi} from 'vitest';
import {Presence} from './presence';
import {Voice} from './voice';
import {PresenceTransportFixture,participant} from './presence-fixture';
const tick=()=>new Promise(resolve=>setTimeout(resolve,0));
const cleanups:(()=>void)[]=[];afterEach(()=>{for(const f of cleanups.splice(0))f();});
function setup(id='a'){
 const t=new PresenceTransportFixture();t.state.connectionId=id;const presence=new Presence(t,()=> 'Alice');presence.start();
 const track={enabled:true,stop:vi.fn(),addEventListener:vi.fn(),removeEventListener:vi.fn()},stream={getTracks:()=>[track],getAudioTracks:()=>[track]} as unknown as MediaStream;
 const peers:any[]=[],audios:any[]=[];
 const media=vi.fn(async()=>stream),config=vi.fn(async():Promise<RTCConfiguration&{expiresAt:string}>=>({iceServers:[],expiresAt:new Date(Date.now()+600000).toISOString()}));
 const peer=vi.fn(()=>{const p:any={signalingState:'stable',connectionState:'new',remoteDescription:null,localDescription:null,onicecandidate:null,ontrack:null,onconnectionstatechange:null,onnegotiationneeded:null,addTrack:vi.fn(),close:vi.fn(),setConfiguration:vi.fn(),restartIce:vi.fn(),createOffer:vi.fn(async()=>({type:'offer',sdp:'offer'})),createAnswer:vi.fn(async()=>({type:'answer',sdp:'answer'})),setLocalDescription:vi.fn(async(d:any)=>{p.localDescription=d;p.signalingState=d.type==='offer'?'have-local-offer':'stable';}),setRemoteDescription:vi.fn(async(d:any)=>{p.remoteDescription=d;p.signalingState=d.type==='offer'?'have-remote-offer':'stable';}),addIceCandidate:vi.fn(async()=>{if(!p.remoteDescription)throw new Error('ICE before remote description');})};peers.push(p);return p;});
 const audio=()=>{const a={autoplay:false,srcObject:null,play:vi.fn(async()=>{}),pause:vi.fn(),remove:vi.fn()};audios.push(a);return a as unknown as HTMLAudioElement;};
 const voice=new Voice(t,presence,{config,media,peer,audio});voice.start();cleanups.push(()=>{voice.stop();presence.stop();});
 const roster=(ids=['a','b'])=>t.emit('roster',{participants:ids.map(p=>participant(p,null,true))});
 const admit=()=>{const request=t.sent.filter(c=>c.type==='voice_join').at(-1)!;t.emit('voice_admission',{admitted:true,connectionId:id,peers:[]},request.requestId);};
 return {t,presence,voice,track,stream,peers,audios,media,config,peer,roster,admit};
}
it('requests media only after admission; voice quota and permission denial leave data usable',async()=>{
 const f=setup();f.voice.join();expect(f.media).not.toHaveBeenCalled();expect(f.config).not.toHaveBeenCalled();
 const id=f.t.sent.at(-1)!.requestId;f.t.emit('voice_admission',{admitted:false,reason:'VOICE_FULL',connectionId:'a',peers:[]},id);await tick();expect(f.media).not.toHaveBeenCalled();expect(f.voice.getSnapshot().error).toMatch(/10/);
 f.media.mockRejectedValueOnce(new DOMException('Denied','NotAllowedError'));f.voice.join();f.admit();await vi.waitFor(()=>expect(f.voice.getSnapshot().status).toBe('error'));
 expect(f.t.sent.some(c=>c.type==='voice_leave')).toBe(true);expect(f.t.state.canEdit).toBe(true);
});
it('cancelling a pending permission prompt stops late tracks and never creates peers',async()=>{
 const f=setup();let release!:(value:MediaStream)=>void;f.media.mockImplementationOnce(()=>new Promise(resolve=>release=resolve));f.voice.join();f.admit();await vi.waitFor(()=>expect(f.media).toHaveBeenCalled());f.voice.leave();release(f.stream);await tick();
 expect(f.track.stop).toHaveBeenCalled();expect(f.peers).toHaveLength(0);expect(f.voice.getSnapshot().status).toBe('idle');
});
it('buffers early ICE until the offer and uses one deterministic offerer',async()=>{
 const f=setup('b');f.roster();f.voice.join();f.admit();await vi.waitFor(()=>expect(f.voice.getSnapshot().status).toBe('connected'));
 f.t.emit('voice_signal',{from:'a',to:'b',candidate:{candidate:'early'}});await tick();expect(f.peers[0].addIceCandidate).not.toHaveBeenCalled();
 f.t.emit('voice_signal',{from:'a',to:'b',description:{type:'offer',sdp:'remote'}});await vi.waitFor(()=>expect(f.peers[0].addIceCandidate).toHaveBeenCalled());
 expect(f.peers[0].createOffer).not.toHaveBeenCalled();expect(f.peers[0].createAnswer).toHaveBeenCalledTimes(1);
 const a=setup('a');a.roster();a.voice.join();a.admit();await vi.waitFor(()=>expect(a.peers[0]?.createOffer).toHaveBeenCalledTimes(1));a.roster();await tick();expect(a.peers[0].createOffer).toHaveBeenCalledTimes(1);
});
it('mute updates tracks; disconnect closes peers, audio and tracks and clears the session',async()=>{
 const f=setup();f.roster();f.voice.join();f.admit();await vi.waitFor(()=>expect(f.peers).toHaveLength(1));
 f.peers[0].ontrack({streams:[f.stream]});await tick();f.voice.toggleMute();expect(f.track.enabled).toBe(false);expect(f.t.sent.at(-1)?.type).toBe('voice_mute');
 f.t.status('offline');expect(f.track.stop).toHaveBeenCalled();expect(f.peers[0].close).toHaveBeenCalled();expect(f.audios[0].srcObject).toBeNull();expect(f.audios[0].remove).toHaveBeenCalled();expect(f.voice.getSnapshot().status).toBe('idle');
});

it('admission remains valid while the server roster is still showing pre-join state',async()=>{
 const f=setup();f.t.emit('roster',{participants:[participant('a'),participant('b')]});f.voice.join();f.admit();
 await vi.waitFor(()=>expect(f.voice.getSnapshot().status).toBe('connected'));expect(f.track.stop).not.toHaveBeenCalled();
 f.roster();await vi.waitFor(()=>expect(f.peers).toHaveLength(1));f.t.emit('roster',{participants:[participant('a',null,true)]});expect(f.peers[0].close).toHaveBeenCalled();
});
it('cancelling before TURN configuration completes never requests a microphone',async()=>{
 const f=setup();let release!:(value:RTCConfiguration&{expiresAt:string})=>void;f.config.mockImplementationOnce(()=>new Promise(resolve=>release=resolve));f.voice.join();f.admit();await tick();f.voice.leave();release({iceServers:[],expiresAt:new Date(Date.now()+600000).toISOString()});await tick();expect(f.media).not.toHaveBeenCalled();expect(f.voice.getSnapshot().status).toBe('idle');
});
