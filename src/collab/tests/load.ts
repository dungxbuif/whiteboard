import WebSocket from 'ws';
import * as Y from 'yjs';
import { decodeFrame, encodeFrame } from '@collab/contracts';
import { randomUUID } from 'node:crypto';
import { cpus, totalmem, platform, arch } from 'node:os';
import { execFileSync } from 'node:child_process';
const wait=(ms:number)=>new Promise(r=>setTimeout(r,ms));
async function until(check:()=>boolean,ms=30000){const deadline=Date.now()+ms;while(!check()){if(Date.now()>deadline)throw new Error('Load timed out');await wait(10);}}
export class DataClient {
  socket:WebSocket; doc=new Y.Doc({gc:false}); controls:any[]=[]; receipts=new Map<string,any>(); started=new Map<string,number>(); latencies:number[]=[];samples:{requestId:string;latencyMs:number}[]=[]; failures:string[]=[];closes:{code:number;reason:string}[]=[];ready:any; synced=false;
  constructor(url:string,origin:string,token:string){
    this.socket=new WebSocket(url+'/collab',{origin});
    this.socket.on('open',()=>this.control('auth',{token}));
    this.socket.on('error',e=>this.failures.push(e.message));this.socket.on('close',(code,reason)=>this.closes.push({code,reason:reason.toString()}));
    let chunks:Buffer[]=[];
    this.socket.on('message',(raw,binary)=>{
      if(binary){const f=decodeFrame(new Uint8Array(raw as Buffer));if(f.channel===3)Y.applyUpdate(this.doc,f.payload,'remote');return;}
      const c=JSON.parse(raw.toString());this.controls.push(c);
      if(c.type==='ready')this.ready=c.payload;
      if(c.type==='sync_chunk'){chunks.push(Buffer.from(c.payload.data,'base64'));if(chunks.length===c.payload.count){Y.applyUpdate(this.doc,Buffer.concat(chunks));chunks=[];this.synced=true;}}
      if(c.type==='ack'){this.receipts.set(c.requestId,c.payload);const start=this.started.get(c.requestId);if(start!==undefined){const latencyMs=performance.now()-start;this.latencies.push(latencyMs);this.samples.push({requestId:c.requestId,latencyMs});}}
      if(c.type==='error')this.failures.push(c.payload.code+': '+c.payload.message);
    });
  }
  async connect(){await until(()=>this.synced||this.failures.length>0);if(this.failures.length)throw new Error(this.failures.join(','));return this;}
  control(type:string,payload:unknown){const requestId=randomUUID();this.socket.send(JSON.stringify({type,requestId,payload}));return requestId;}
  update(value:string,elementId?:string){const vector=Y.encodeStateVector(this.doc);if(elementId)this.doc.getMap<Y.Map<unknown>>('elements').get(elementId)!.set('x',Number(value));else this.doc.getMap('appState').set('viewBackgroundColor',value);const requestId=randomUUID();this.started.set(requestId,performance.now());this.socket.send(encodeFrame({channel:3,updateId:requestId,payload:Y.encodeStateAsUpdate(this.doc,vector)}));return requestId;}
  close(){this.socket.terminate();this.doc.destroy();}
}
export async function runLoad(options:{url:string;origin:string;tokens:string[];elementIds:string[];runtimeProcessGroup:number;warmupSeconds?:number;seconds?:number}){
  if(options.tokens.length!==50)throw new Error('Acceptance requires exactly 50 independent principals');
  const clients:DataClient[]=[];const processSamples:{rssBytes:number;cpuPercent:number}[]=[];
  const warmup=options.warmupSeconds??3,seconds=options.seconds??10;const runFailures:string[]=[];
  try{
    for(const token of options.tokens)clients.push(await new DataClient(options.url,options.origin,token).connect());
    const deliveryStarts=new Map<string,number>(),deliveryLatencies:number[]=[];
    clients.forEach((client,index)=>{const observed=new Map<string,number>();client.doc.on('update',(_update,origin)=>{if(origin!=='remote')return;for(const id of options.elementIds){if(id===options.elementIds[index])continue;const value=client.doc.getMap<Y.Map<unknown>>('elements').get(id)?.get('x') as number;if(observed.get(id)===value)continue;observed.set(id,value);const started=deliveryStarts.get(id+':'+value);if(started!==undefined&&value>warmup)deliveryLatencies.push(performance.now()-started);}});});
    const baselineCPU=process.cpuUsage(),start=performance.now();
    const sample=setInterval(()=>{const rows=execFileSync('rtk',['proxy','ps','-axo','pgid=,rss=,%cpu='],{encoding:'utf8',env:{...process.env,PATH:(process.env.PATH??'')+':/opt/homebrew/bin'}}).split('\n').map(r=>r.trim().split(/\s+/).map(Number)).filter(r=>r[0]===options.runtimeProcessGroup);processSamples.push({rssBytes:rows.reduce((n,r)=>n+r[1]*1024,0),cpuPercent:rows.reduce((n,r)=>n+r[2],0)});},1000);
    try{
      await Promise.all(clients.map(async(c,i)=>{await wait(i*20);for(let tick=0;tick<warmup+seconds;tick++){const target=start+i*20+tick*1000;await wait(Math.max(0,target-performance.now()));deliveryStarts.set(options.elementIds[i]+':'+(tick+1),performance.now());c.update(String(tick+1),options.elementIds[i]);}}));
      await until(()=>clients.every(c=>c.receipts.size===warmup+seconds),30000).catch(error=>runFailures.push(String(error)));
      await until(()=>clients.every(c=>options.elementIds.every(id=>c.doc.getMap<Y.Map<unknown>>('elements').get(id)?.get('x')===warmup+seconds)),5000).catch(error=>runFailures.push('Convergence: '+String(error)));
    }finally{clearInterval(sample);}
    const latency=clients.flatMap(c=>c.latencies.slice(warmup)).sort((a,b)=>a-b);
    deliveryLatencies.sort((a,b)=>a-b);
    const finalPositions=clients.map(c=>options.elementIds.map(id=>c.doc.getMap<Y.Map<unknown>>('elements').get(id)?.get('x')));
    const acknowledged=clients.flatMap(c=>[...c.receipts.entries()].map(([requestId,r])=>({requestId,sequence:r.sequence})));
    return {delivery:{samples:deliveryLatencies.length,p50Ms:deliveryLatencies[Math.floor(deliveryLatencies.length*.5)],p95Ms:deliveryLatencies[Math.floor(deliveryLatencies.length*.95)],maxMs:deliveryLatencies.at(-1),latenciesMs:deliveryLatencies},finalPositions,clients:50,updatesPerClientPerSecond:1,warmupSeconds:warmup,measurementSeconds:seconds,sent:clients.reduce((n,c)=>n+c.started.size,0),acknowledged,failures:[...runFailures,...clients.flatMap(c=>c.failures)],perClient:clients.map(c=>({sent:c.started.size,acked:c.receipts.size,readyState:c.socket.readyState,failures:c.failures,closes:c.closes})),p50Ms:latency[Math.floor(latency.length*.5)],p95Ms:latency[Math.floor(latency.length*.95)],maxMs:latency.at(-1),latenciesMs:latency,perUpdate:clients.flatMap(c=>c.samples.map((s,i)=>({...s,warmup:i<warmup,sequence:c.receipts.get(s.requestId)?.sequence}))),processSamples,peakRuntimeRSSBytes:Math.max(...processSamples.map(s=>s.rssBytes)),clientCPU:process.cpuUsage(baselineCPU),clientRSSBytes:process.memoryUsage().rss,hardware:{platform:platform(),arch:arch(),logicalCPUs:cpus().length,cpuModel:cpus()[0]?.model,hostMemoryBytes:totalmem()}};
  }finally{clients.forEach(c=>c.close());}
}
