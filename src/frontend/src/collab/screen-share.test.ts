import {afterEach,expect,it,vi} from 'vitest';
import {Presence} from './presence';
import {PresenceTransportFixture,participant} from './presence-fixture';
import {ScreenShare} from './screen-share';

const cleanups:(()=>void)[]=[];
const tick=()=>new Promise(resolve=>setTimeout(resolve,0));
const lastScreenRequest=(sent:PresenceTransportFixture['sent'])=>sent.slice().reverse().find(control=>control.type==='screen_request');
afterEach(()=>{for(const cleanup of cleanups.splice(0))cleanup();});

function setup(){
 const transport=new PresenceTransportFixture();transport.state.connectionId='self';
 const presence=new Presence(transport,()=> 'Publisher');presence.start();
 const videoTrack={kind:'video',stop:vi.fn(),addEventListener:vi.fn(),removeEventListener:vi.fn()};
 const audioTrack={kind:'audio',stop:vi.fn(),addEventListener:vi.fn(),removeEventListener:vi.fn()};
 const stream={getTracks:()=>[videoTrack,audioTrack],getVideoTracks:()=>[videoTrack],getAudioTracks:()=>[audioTrack]} as unknown as MediaStream;
 const peers:any[]=[];
 const peer=vi.fn(()=>{const value:any={signalingState:'stable',connectionState:'new',remoteDescription:null,localDescription:null,onicecandidate:null,ontrack:null,onconnectionstatechange:null,addTrack:vi.fn(),close:vi.fn(),createOffer:vi.fn(async()=>({type:'offer',sdp:'offer'})),createAnswer:vi.fn(async()=>({type:'answer',sdp:'answer'})),setLocalDescription:vi.fn(async(description:any)=>{value.localDescription=description;}),setRemoteDescription:vi.fn(async(description:any)=>{value.remoteDescription=description;}),addIceCandidate:vi.fn(async()=>{})};peers.push(value);return value;});
 const media=vi.fn(async()=>stream),config=vi.fn(async():Promise<RTCConfiguration&{expiresAt:string}>=>({iceServers:[],expiresAt:new Date(Date.now()+600000).toISOString()}));
 const screen=new ScreenShare(transport,presence,{media,config,peer});screen.start();cleanups.push(()=>{screen.stop();presence.stop();});
 const admitPublisher=()=>{const request=lastScreenRequest(transport.sent)!;transport.emit('screen_admission',{admitted:true,connectionId:'self',kind:'screen',publisher:'self',receiving:false,sessionId:'00000000-0000-4000-8000-000000000001',peers:[]},request.requestId);};
 return {transport,presence,screen,media,config,peer,peers,stream,videoTrack,audioTrack,admitPublisher};
}

it('waits for admission and a second explicit start before requesting video only',async()=>{
 const fixture=setup();const joined=fixture.screen.join();expect(fixture.media).not.toHaveBeenCalled();fixture.admitPublisher();await joined;
 expect(fixture.screen.getSnapshot().status).toBe('ready');expect(fixture.media).not.toHaveBeenCalled();
 const started=fixture.screen.startSharing();expect(fixture.media).toHaveBeenCalledWith({video:{width:{max:1280},height:{max:720},frameRate:{max:15}},audio:false});await started;
 expect(fixture.audioTrack.stop).toHaveBeenCalled();expect(fixture.screen.getSnapshot()).toMatchObject({status:'sharing',publisher:'self',receiving:false});
 expect(lastScreenRequest(fixture.transport.sent)?.payload).toEqual({sessionId:'00000000-0000-4000-8000-000000000001'});
});

it('lateCaptureStopsAllTracks',async()=>{
 const fixture=setup();let release!:(stream:MediaStream)=>void;fixture.media.mockImplementationOnce(()=>new Promise(resolve=>release=resolve));
 const joined=fixture.screen.join();fixture.admitPublisher();await joined;const starting=fixture.screen.startSharing();fixture.screen.leave();release(fixture.stream);await starting;await tick();
 expect(fixture.videoTrack.stop).toHaveBeenCalled();expect(fixture.audioTrack.stop).toHaveBeenCalled();expect(fixture.peers).toHaveLength(0);expect(fixture.screen.getSnapshot().status).toBe('idle');
});

it('stops a late capture when relay configuration fails first',async()=>{
 const fixture=setup();let release!:(stream:MediaStream)=>void;fixture.media.mockImplementationOnce(()=>new Promise(resolve=>release=resolve));fixture.config.mockRejectedValueOnce(new Error('relay unavailable'));
 const joined=fixture.screen.join();fixture.admitPublisher();await joined;const starting=fixture.screen.startSharing();await tick();release(fixture.stream);await starting;await tick();
 expect(fixture.videoTrack.stop).toHaveBeenCalled();expect(fixture.audioTrack.stop).toHaveBeenCalled();expect(fixture.screen.getSnapshot().status).toBe('error');
});

it('receives a remote screen on an independent peer without joining voice',async()=>{
 const fixture=setup();const video={autoplay:false,playsInline:false,srcObject:null,play:vi.fn(async()=>{}),pause:vi.fn()} as unknown as HTMLVideoElement;fixture.screen.attachVideo(video);
 fixture.transport.emit('roster',{participants:[participant('self'),{...participant('publisher'),screenJoined:true,screenPublisher:true,screenSessionId:'00000000-0000-4000-8000-000000000001'}]});
 await vi.waitFor(()=>expect(fixture.transport.sent.some(control=>control.type==='screen_request')).toBe(true));const request=lastScreenRequest(fixture.transport.sent)!;
 fixture.transport.emit('screen_admission',{admitted:true,connectionId:'self',kind:'screen',publisher:'publisher',receiving:true,sessionId:'00000000-0000-4000-8000-000000000001',peers:['publisher']},request.requestId);
 fixture.transport.emit('screen_signal',{from:'publisher',to:'self',sessionId:'00000000-0000-4000-8000-000000000001',description:{type:'offer',sdp:'remote'}});await vi.waitFor(()=>expect(fixture.peers).toHaveLength(1));
 fixture.peers[0].ontrack({streams:[fixture.stream],track:fixture.videoTrack});await tick();expect(video.srcObject).toBe(fixture.stream);expect(video.play).toHaveBeenCalled();expect(fixture.media).not.toHaveBeenCalled();
 fixture.screen.leave();expect(fixture.videoTrack.stop).toHaveBeenCalled();expect(video.srcObject).toBeNull();
});

it('loads relay configuration before requesting receiver admission',async()=>{
 const fixture=setup();let release!:(config:RTCConfiguration&{expiresAt:string})=>void;fixture.config.mockImplementationOnce(()=>new Promise(resolve=>release=resolve));
 fixture.transport.emit('roster',{participants:[participant('self'),{...participant('publisher'),screenJoined:true,screenPublisher:true,screenSessionId:'00000000-0000-4000-8000-000000000001'}]});
 expect(fixture.transport.sent.some(control=>control.type==='screen_request')).toBe(false);
 release({iceServers:[{urls:'turn:relay.example'}],expiresAt:new Date(Date.now()+600000).toISOString()});
 await vi.waitFor(()=>expect(fixture.transport.sent.some(control=>control.type==='screen_request')).toBe(true));
});

it('fails closed on malformed admission metadata',()=>{
 const fixture=setup();void fixture.screen.join().catch(()=>{});const request=lastScreenRequest(fixture.transport.sent)!;
 fixture.transport.emit('screen_admission',{admitted:true,connectionId:'self',publisher:'self',receiving:false,sessionId:'not-a-session',peers:[]},request.requestId);
 expect(fixture.screen.getSnapshot()).toMatchObject({status:'error',receiving:false});
});
