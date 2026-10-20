import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {chromium} from '../../src/node_modules/playwright/index.mjs';

const mint=`import json,subprocess,time,hmac,hashlib,base64
c=json.loads(subprocess.check_output(['docker','inspect','whiteboard-backend']))[0]
e=dict(v.split('=',1) for v in c['Config']['Env'] if '=' in v)
u=str(int(time.time())+600)+':whiteboard-screen-e1'
p=base64.b64encode(hmac.new(e['TURN_AUTH_SECRET'].encode(),u.encode(),hashlib.sha1).digest()).decode()
print(json.dumps({'urls':e['TURN_URL'],'username':u,'credential':p}))`;

async function prepare(page,configuration,publish){
 await page.goto('data:text/html,<title>Generated screen media proof</title><video autoplay muted playsinline></video>');
 await page.evaluate(({configuration,publish})=>{
  const pc=new RTCPeerConnection(configuration);window.pc=pc;window.candidates=[];
  pc.onicecandidate=event=>{if(event.candidate)window.candidates.push(event.candidate.type);};
  pc.ontrack=event=>{const video=document.querySelector('video');video.srcObject=event.streams[0];void video.play();};
  if(publish){const canvas=document.createElement('canvas');canvas.width=320;canvas.height=180;const context=canvas.getContext('2d');let frame=0;window.paint=setInterval(()=>{context.fillStyle=frame++%2?'#6965db':'#1b1b1f';context.fillRect(0,0,320,180);context.fillStyle='white';context.fillText(String(frame),10,20);},66);window.stream=canvas.captureStream(15);window.stream.getVideoTracks().forEach(track=>pc.addTrack(track,window.stream));}
 },{configuration,publish});
}

async function gather(page,offer){return page.evaluate(async offer=>{if(offer)await window.pc.setRemoteDescription(offer);await window.pc.setLocalDescription(offer?await window.pc.createAnswer():await window.pc.createOffer());await new Promise((resolve,reject)=>{if(window.pc.iceGatheringState==='complete')return resolve();const timer=setTimeout(()=>reject(new Error('ICE gathering timeout '+JSON.stringify(window.candidates))),45_000);window.pc.addEventListener('icegatheringstatechange',()=>{if(window.pc.iceGatheringState==='complete'){clearTimeout(timer);resolve();}});});return window.pc.localDescription.toJSON();},offer);}

async function prove(publisher,viewer,configuration,label,relayOnly){
 await prepare(publisher,configuration,true);await prepare(viewer,configuration,false);const offer=await gather(publisher);const answer=await gather(viewer,offer);await publisher.evaluate(value=>window.pc.setRemoteDescription(value),answer);
 await Promise.all([publisher,viewer].map(page=>page.waitForFunction(()=>window.pc.connectionState==='connected',null,{timeout:20_000})));await viewer.waitForFunction(()=>{const video=document.querySelector('video');return video.readyState>=2&&video.videoWidth>0;},null,{timeout:20_000});await new Promise(resolve=>setTimeout(resolve,2_000));
 const evidence=await viewer.evaluate(async()=>{const stats=[...(await window.pc.getStats()).values()],inbound=stats.find(row=>row.type==='inbound-rtp'&&row.kind==='video'),transport=stats.find(row=>row.type==='transport'&&row.selectedCandidatePairId),pair=stats.find(row=>row.id===transport?.selectedCandidatePairId),local=stats.find(row=>row.id===pair?.localCandidateId),remote=stats.find(row=>row.id===pair?.remoteCandidateId);return {connection:window.pc.connectionState,bytesReceived:inbound?.bytesReceived,framesDecoded:inbound?.framesDecoded,localType:local?.candidateType,remoteType:remote?.candidateType};});
 assert.equal(evidence.connection,'connected');assert(evidence.bytesReceived>0);assert(evidence.framesDecoded>0);if(relayOnly){assert.equal(evidence.localType,'relay');assert.equal(evidence.remoteType,'relay');}
 for(const page of [publisher,viewer])await page.evaluate(()=>{clearInterval(window.paint);window.stream?.getTracks().forEach(track=>track.stop());window.pc.close();});return {label,generatedVideo:'320x180 canvas at 15fps; no microphone or system audio',...evidence};
}

let local,remote;
try{
 local=await chromium.launch({headless:true,args:['--autoplay-policy=no-user-gesture-required']});const localContext=await local.newContext();const evidence=[await prove(await localContext.newPage(),await localContext.newPage(),{},'local generated-frame delivery',false)];
 if(process.env.MEDIA_LOCAL_ONLY==='1'){console.log(JSON.stringify({pass:true,scope:'local-only',evidence},null,2));process.exitCode=0;}else{
 remote=await chromium.connectOverCDP('http://127.0.0.1:19223');const ice=JSON.parse(execFileSync('ssh',['-o','BatchMode=yes','vm100','python3 -'],{input:mint,encoding:'utf8',stdio:['pipe','pipe','pipe']}));if(process.env.TURN_PROOF_URL)ice.urls=process.env.TURN_PROOF_URL;
 evidence.push(await prove(await localContext.newPage(),await remote.newPage(),{iceServers:[ice],iceTransportPolicy:'relay'},'two-network relay-only generated-frame delivery',true));console.log(JSON.stringify({pass:true,evidence},null,2));
 }
}finally{await local?.close();await remote?.close();}
