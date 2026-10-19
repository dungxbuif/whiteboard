import { test, expect } from './fixtures';
import { runLoad, DataClient } from '../../../collab/tests/load';
import { writeFileSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { until } from './harness';
test('50 principals each send one small update/sec: durable ACKs and local p95 under 1 second',async({stack},info)=>{
  test.setTimeout(180000);const elementIds=Array.from({length:50},(_,i)=>'load-'+i);
  const shapes=elementIds.map((id,i)=>({id,type:'rectangle',x:0,y:i*30,width:20,height:20,angle:0,strokeColor:'#000000',backgroundColor:'transparent',fillStyle:'solid',strokeWidth:1,strokeStyle:'solid',roughness:0,opacity:100,groupIds:[],frameId:null,roundness:null,seed:i+1,version:1,versionNonce:i+1,index:null,isDeleted:false,boundElements:null,updated:1,link:null,locked:false}));
  const scene=await stack.json('/scenes',stack.owner.token,stack.owner.workspaceId,'POST',{title:'50-client load',content:{elements:shapes,appState:{},files:{}}});const link=await stack.json('/scenes/'+scene.id+'/share-links',stack.owner.token,stack.owner.workspaceId,'POST',{access:'edit'});const tokens:string[]=[];
  for(let i=0;i<50;i++){const session=await stack.json('/public/scene-session','','','POST',{token:link.token,displayName:'Load '+i});tokens.push((await stack.json('/scenes/'+scene.id+'/room-token',session.token,stack.owner.workspaceId,'POST',{})).token);}
  const principals=tokens.map(t=>JSON.parse(Buffer.from(t.split('.')[1],'base64url').toString()).principalId);expect(new Set(principals).size).toBe(50);
  // Fresh process isolates admission budgets from earlier acceptance tests.
  stack.env.E1_PROFILE='true';await stack.stop(stack.runtime);await stack.startRuntime();
  const result=await runLoad({url:stack.ws,origin:stack.baseURL,tokens,elementIds,runtimeProcessGroup:stack.runtime!.pid!});
  writeFileSync(info.outputPath('runtime-profile.json'),readFileSync(join(stack.dir,'runtime-profile.json')));
  writeFileSync(info.outputPath('load-results.json'),JSON.stringify(result,null,2));console.log(JSON.stringify({load:{sent:result.sent,acks:result.acknowledged.length,p50Ms:result.p50Ms,p95Ms:result.p95Ms,deliveryP95Ms:result.delivery.p95Ms,peakRuntimeRSSBytes:result.peakRuntimeRSSBytes,hardware:result.hardware,failures:[...new Set(result.failures)]}}));
  const durableRows=(await stack.pool.query('SELECT request_id,sequence::text FROM collab_receipts WHERE scene_id=$1',[scene.id])).rows;writeFileSync(info.outputPath('durable-receipts.json'),JSON.stringify(durableRows,null,2));
  expect(result.failures).toEqual([]);expect(result.finalPositions).toEqual(Array.from({length:50},()=>Array(50).fill(13)));expect(result.acknowledged).toHaveLength(result.sent);
  const stored=(await stack.pool.query('SELECT request_id,sequence::text FROM collab_receipts WHERE scene_id=$1',[scene.id])).rows;
  expect(stored).toHaveLength(result.sent);for(const ack of result.acknowledged)expect(stored.find(r=>r.request_id===ack.requestId)?.sequence).toBe(ack.sequence);
  expect(result.p95Ms).toBeLessThan(1000);expect(result.delivery.samples).toBe(50*49*10);expect(result.delivery.p95Ms).toBeLessThan(1000);
});
test('ten real voice admissions; eleventh denied while its data update commits',async({stack})=>{
  const scene=await stack.scene();await stack.stop(stack.runtime);await stack.startRuntime();const room=await stack.room(scene.id);const clients:DataClient[]=[];
  try{
    for(let i=0;i<11;i++)clients.push(await new DataClient(stack.ws,stack.baseURL,room.token).connect());
    for(let i=0;i<11;i++){const id=clients[i].control('voice_join',{});await until(()=>clients[i].controls.some(c=>c.requestId===id&&c.type==='voice_admission'));const admission=clients[i].controls.find(c=>c.requestId===id&&c.type==='voice_admission').payload;expect(admission.admitted).toBe(i<10);if(i===10)expect(admission.reason).toBe('VOICE_FULL');}
    const id=clients[10].update('#abcdef');await until(()=>clients[10].receipts.has(id));expect((await stack.pool.query('SELECT count(*) FROM collab_receipts WHERE scene_id=$1 AND request_id=$2',[scene.id,id])).rows[0].count).toBe('1');
  }finally{clients.forEach(c=>c.close());}
});

test('actual per-principal rate limit rejects excess without blocking another editor',async({stack})=>{
  const scene=await stack.scene();await stack.stop(stack.runtime);await stack.startRuntime();const a=await new DataClient(stack.ws,stack.baseURL,(await stack.room(scene.id)).token).connect();const b=await new DataClient(stack.ws,stack.baseURL,(await stack.room(scene.id,stack.editor)).token).connect();
  try{
    for(let i=0;i<120;i++){const id=a.update('#'+(i+1).toString(16).padStart(6,'0'));await until(()=>a.receipts.has(id));}
    const denied=a.update('#ffffff');await until(()=>a.controls.some(c=>c.requestId===denied&&c.type==='error'));expect(a.controls.find(c=>c.requestId===denied&&c.type==='error').payload.code).toBe('RATE_LIMITED');
    const accepted=b.update('#000000');await until(()=>b.receipts.has(accepted));expect((await stack.pool.query('SELECT count(*) FROM collab_receipts WHERE scene_id=$1',[scene.id])).rows[0].count).toBe('121');
  }finally{a.close();b.close();await stack.stop(stack.runtime);await stack.startRuntime();}
});
