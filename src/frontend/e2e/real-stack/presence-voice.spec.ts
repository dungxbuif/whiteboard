import {test,expect} from './fixtures';
import {openScene,draw,elements} from './browser';
import type {Page} from '@playwright/test';
async function observe(page:Page){await page.addInitScript(()=>{
 const state=(window as any).__b4={roster:[],viewports:[],awareness:[],signals:0,pcs:[],streams:[],controls:[]};
 const WS=window.WebSocket;window.WebSocket=class extends WS{constructor(...args:ConstructorParameters<typeof WebSocket>){super(...args);this.addEventListener('message',event=>{if(typeof event.data!=='string')return;let frame;try{frame=JSON.parse(event.data);}catch{return;}state.controls.push({direction:'in',frame});if(frame.type==='roster')state.roster=frame.payload.participants;if(frame.type==='viewport')state.viewports.push(frame.payload);if(frame.type==='awareness')state.awareness.push(frame.payload);if(frame.type==='voice_signal')state.signals++;});}send(data:string|ArrayBufferLike|Blob|ArrayBufferView){if(typeof data==='string'){try{state.controls.push({direction:'out',frame:JSON.parse(data)});}catch{}}super.send(data);}};
 const PC=window.RTCPeerConnection;window.RTCPeerConnection=class extends PC{constructor(config?:RTCConfiguration){super(config);state.pcs.push(this);}};
 });}
const roster=(page:Page)=>page.evaluate(()=>(window as any).__b4.roster);
const followControls=(page:Page)=>page.evaluate(()=>(window as any).__b4.controls.filter((event:any)=>event.frame.type==='follow'||event.frame.type==='error'));
test('native same-user tabs remain distinct; follow camera is ephemeral and exit clears followers',async({browser,stack},info)=>{
 const scene=await stack.scene('B4 native presence');const context=await browser.newContext();
 try{
  const a=await context.newPage(),b=await context.newPage();await observe(a);await observe(b);await openScene(a,stack,scene);await b.goto(stack.baseURL+'/w/'+stack.owner.workspaceId+'/scenes/'+scene.id);await expect(b.getByRole('status')).toHaveText('Saved');await expect(b.locator('canvas').first()).toBeVisible();
  await expect(a.getByLabel('People (2)',{exact:true})).toBeVisible();const members=await roster(a);expect(new Set(members.map((p:any)=>p.connectionId)).size).toBe(2);expect(new Set(members.map((p:any)=>p.principalId)).size).toBe(1);expect(members.every((p:any)=>p.access==='edit')).toBe(true);
  await a.getByLabel('People (2)',{exact:true}).click();await a.getByRole('button',{name:'Follow owner',exact:true}).click();await expect.poll(()=>followControls(a)).toEqual(expect.arrayContaining([expect.objectContaining({direction:'out',frame:expect.objectContaining({type:'follow'})}),expect.objectContaining({direction:'in',frame:expect.objectContaining({type:'follow'})})]));await expect(a.getByRole('button',{name:'Stop following',exact:true})).toBeVisible();
  await expect.poll(async()=>(await roster(b)).filter((p:any)=>p.following).length).toBe(1);
  const before=(await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id])).rows[0].sequence;
  const box=await b.locator('canvas').first().boundingBox();await b.bringToFront();for(let i=0;i<30;i++){await b.mouse.move(box!.x+350+i*3,box!.y+280);await b.waitForTimeout(50);}
  await expect.poll(()=>a.evaluate(()=>(window as any).__b4.awareness.filter((p:any)=>p.cursor).length)).toBeGreaterThan(10);await expect(a.getByRole('status')).toHaveText('Saved');await b.mouse.wheel(160,210);
  await expect.poll(()=>a.evaluate(()=>(window as any).__b4.viewports.length)).toBeGreaterThan(0);
  expect((await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id])).rows[0].sequence).toBe(before);
  await a.screenshot({path:info.outputPath('native-follow.png')});await b.close();await expect(a.getByLabel('People (1)',{exact:true})).toBeVisible();await expect(a.getByRole('button',{name:'Stop following',exact:true})).toHaveCount(0);
 }finally{await context.close();}
});
test('microphone denial releases admission and native editing remains usable',async({browser,stack},info)=>{
 const scene=await stack.scene('B4 microphone denial'),context=await browser.newContext();
 try{const page=await context.newPage();await observe(page);await page.addInitScript(()=>Object.defineProperty(navigator.mediaDevices,'getUserMedia',{value:async()=>{throw new DOMException('B4 denied','NotAllowedError');}}));await openScene(page,stack,scene);
  await page.getByRole('button',{name:'Join voice',exact:true}).click();await expect(page.getByRole('alert')).toContainText('Microphone permission was denied');await expect.poll(async()=>(await roster(page)).filter((p:any)=>p.voiceJoined).length).toBe(0);
  await draw(page);await expect(page.getByRole('status')).toHaveText('Saved');await expect.poll(async()=>(await elements(page)).length).toBe(1);await page.screenshot({path:info.outputPath('microphone-denial-editing.png')});
 }finally{await context.close();}
});

async function generatedMicrophone(page:Page){await page.addInitScript(()=>{
 Object.defineProperty(navigator.mediaDevices,'getUserMedia',{value:async()=>{const audio=new AudioContext();await audio.resume();const source=audio.createOscillator(),gain=audio.createGain(),destination=audio.createMediaStreamDestination();source.frequency.value=440;gain.gain.value=0.03;source.connect(gain);gain.connect(destination);source.start();(window as any).__b4.streams.push(destination.stream);const track=destination.stream.getAudioTracks()[0],stop=track.stop.bind(track);track.stop=()=>{stop();source.stop();void audio.close();};return destination.stream;}});
 });}
async function audioBytes(page:Page){return page.evaluate(async()=>{let bytes=0;for(const pc of (window as any).__b4.pcs)for(const report of (await pc.getStats()).values())if(report.type==='inbound-rtp'&&report.kind==='audio')bytes+=report.bytesReceived??0;return bytes;});}
test('two native clients exchange generated audio, mute and leave without affecting data',async({browser,stack},info)=>{
 const scene=await stack.scene('B4 native voice'),context=await browser.newContext();
 try{const a=await context.newPage(),b=await context.newPage();for(const page of [a,b]){await observe(page);await generatedMicrophone(page);}
  await openScene(a,stack,scene);await b.goto(stack.baseURL+'/w/'+stack.owner.workspaceId+'/scenes/'+scene.id);await expect(b.getByRole('status')).toHaveText('Saved');
  await a.getByRole('button',{name:'Join voice',exact:true}).click();await expect(a.getByRole('button',{name:'Mute microphone',exact:true})).toBeVisible();
  await b.getByRole('button',{name:'Join voice',exact:true}).click();await expect(b.getByRole('button',{name:'Mute microphone',exact:true})).toBeVisible();
  await expect.poll(()=>audioBytes(a),{timeout:20000}).toBeGreaterThan(100);await expect.poll(()=>audioBytes(b),{timeout:20000}).toBeGreaterThan(100);
  await a.getByRole('button',{name:'Mute microphone',exact:true}).click();expect(await a.evaluate(()=>(window as any).__b4.streams[0].getAudioTracks()[0].enabled)).toBe(false);await expect.poll(async()=>(await roster(b)).filter((p:any)=>p.voiceMuted).length).toBe(1);
  await a.screenshot({path:info.outputPath('native-voice-muted.png')});await a.getByRole('button',{name:'Leave voice',exact:true}).click();
  await expect.poll(()=>a.evaluate(()=>(window as any).__b4.streams.every((s:MediaStream)=>s.getTracks().every(t=>t.readyState==='ended')))).toBe(true);await expect.poll(()=>b.evaluate(()=>(window as any).__b4.pcs.every((p:RTCPeerConnection)=>p.connectionState==='closed'))).toBe(true);
  await draw(a);await expect(a.getByRole('status')).toHaveText('Saved');await expect.poll(async()=>(await elements(b)).length).toBe(1);
 }finally{await context.close();}
});
