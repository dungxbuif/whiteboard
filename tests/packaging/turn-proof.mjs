import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {chromium} from '../../src/node_modules/playwright/index.mjs';
// Only short-lived REST credentials cross SSH; the shared secret never leaves VM100.
const mint=`import json,subprocess,time,hmac,hashlib,base64
c=json.loads(subprocess.check_output(['docker','inspect','whiteboard-backend']))[0]
e=dict(v.split('=',1) for v in c['Config']['Env'] if '=' in v)
u=str(int(time.time())+600)+':whiteboard-e2'
p=base64.b64encode(hmac.new(e['TURN_AUTH_SECRET'].encode(),u.encode(),hashlib.sha1).digest()).decode()
print(json.dumps({'urls':e['TURN_URL'],'username':u,'credential':p}))`;
const ice=JSON.parse(execFileSync('ssh',['-o','BatchMode=yes','vm100','python3 -'],{input:mint,encoding:'utf8',stdio:['pipe','pipe','pipe']}));
const configuredURL=ice.urls;if(process.env.TURN_PROOF_URL)ice.urls=process.env.TURN_PROOF_URL;
let local,remote;
try{
 console.error('TURN: launch local browser');
 local=await chromium.launch({headless:true,args:['--autoplay-policy=no-user-gesture-required']});
 console.error('TURN: connect gateway browser');
 remote=await chromium.connectOverCDP('http://127.0.0.1:19223');
 console.error('TURN: create pages');
 const pages=[await local.newPage(),await remote.newPage()];
 for(const page of pages){await page.goto('data:text/html,<title>Isolated TURN QA</title>');}
 for(const page of pages)await page.evaluate(async ice=>{
  const ctx=new AudioContext();void ctx.resume();const osc=ctx.createOscillator(),dest=ctx.createMediaStreamDestination();osc.frequency.value=440;osc.connect(dest);osc.start();
  const pc=new RTCPeerConnection({iceServers:[ice],iceTransportPolicy:'relay'});dest.stream.getTracks().forEach(t=>pc.addTrack(t,dest.stream));
  window.iceErrors=[];window.candidates=[];pc.onicecandidateerror=e=>window.iceErrors.push({code:e.errorCode,text:e.errorText});pc.onicecandidate=e=>{if(e.candidate)window.candidates.push(e.candidate.type)};
  pc.ontrack=e=>{const audio=new Audio();audio.srcObject=e.streams[0];audio.autoplay=true;window.audio=audio;};window.pc=pc;window.audioContext=ctx;
 },ice);
 const gather=async(page,offer)=>page.evaluate(async offer=>{
  const pc=window.pc;if(offer)await pc.setRemoteDescription(offer);
  await pc.setLocalDescription(offer?await pc.createAnswer():await pc.createOffer());
  await new Promise((resolve,reject)=>{if(pc.iceGatheringState==='complete')return resolve();const t=setTimeout(()=>reject(new Error('ICE gathering timed out '+JSON.stringify({candidates:window.candidates,errors:window.iceErrors}))),45000);pc.addEventListener('icegatheringstatechange',()=>{if(pc.iceGatheringState==='complete'){clearTimeout(t);resolve();}});});
  return pc.localDescription.toJSON();
 },offer);
 console.error('TURN: gather relay candidates');
 const offer=await gather(pages[0]);const answer=await gather(pages[1],offer);await pages[0].evaluate(a=>window.pc.setRemoteDescription(a),answer);
 console.error('TURN: wait connected');
 await Promise.all(pages.map(page=>page.waitForFunction(()=>window.pc.connectionState==='connected',null,{timeout:20000})));
 await new Promise(r=>setTimeout(r,5000));
 const evidence=[];
 for(const [i,page] of pages.entries())evidence.push(await page.evaluate(async i=>{
  const stats=await window.pc.getStats(),rows=[...stats.values()],transport=rows.find(r=>r.type==='transport'&&r.selectedCandidatePairId),pair=stats.get(transport?.selectedCandidatePairId),local=stats.get(pair?.localCandidateId),remote=stats.get(pair?.remoteCandidateId),audio=rows.find(r=>r.type==='inbound-rtp'&&r.kind==='audio');
  return {location:i===0?'Mac LAN10.10.0.10 WAN117.1.89.250':'Gateway WAN103.82.21.202',connection:window.pc.connectionState,pair:{state:pair?.state,localType:local?.candidateType,remoteType:remote?.candidateType,localAddress:local?.address,remoteAddress:remote?.address,localPort:local?.port,remotePort:remote?.port,protocol:local?.protocol,relayProtocol:local?.relayProtocol},audio:{bytesReceived:audio?.bytesReceived,packetsReceived:audio?.packetsReceived,totalAudioEnergy:audio?.totalAudioEnergy,totalSamplesReceived:audio?.totalSamplesReceived}};
 },i));
 for(const e of evidence){assert.equal(e.pair.localType,'relay');assert.equal(e.pair.remoteType,'relay');assert(e.audio.bytesReceived>0);assert(e.audio.packetsReceived>0);assert(e.audio.totalAudioEnergy>0);}
 console.log(JSON.stringify({pass:true,configuredURL,testedURL:ice.urls,generatedAudio:'440Hz WebAudio oscillator; no microphone permission',evidence},null,2));
 for(const page of pages){await page.evaluate(()=>{window.pc.close();return window.audioContext.close();});await page.close();}
}finally{await local?.close();await remote?.close();}
