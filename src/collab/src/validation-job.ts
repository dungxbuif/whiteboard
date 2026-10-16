import {Worker} from 'node:worker_threads';
import type {ValidatedUpdate} from './document.js';
export const VALIDATION_DEADLINE_MS=2000;
export const VALIDATION_RESOURCE_LIMITS=Object.freeze({maxOldGenerationSizeMb:256,maxYoungGenerationSizeMb:16,stackSizeMb:4});
export type ValidationInput={snapshot:Uint8Array;update:Uint8Array};
type Job={input:ValidationInput;deadline:number;timer:NodeJS.Timeout;resolve:(result:ValidatedUpdate)=>void;reject:(error:Error)=>void};
type Slot={worker:Worker;job?:Job;retiring?:boolean};

/** Fixed worker/queue bounds. Each worker processes only one fresh document at a time. */
export class ValidationPool {
 private slots=new Set<Slot>();private queue:Job[]=[];private closed=false;
 constructor(private url=new URL('../dist/validation-worker.js',import.meta.url),private capacity=2,private queueLimit=32){}
 run(input:ValidationInput):Promise<ValidatedUpdate>{
  if(this.closed)return Promise.reject(new Error('Candidate validation pool closed'));
  if(this.slots.size>=this.capacity&&[...this.slots].every(s=>s.job||s.retiring)&&this.queue.length>=this.queueLimit)return Promise.reject(new Error('Candidate validation queue full'));
  return new Promise((resolve,reject)=>{
   const job:Job={input,deadline:performance.now()+VALIDATION_DEADLINE_MS,resolve,reject,timer:setTimeout(()=>this.expire(job),VALIDATION_DEADLINE_MS)};
   this.queue.push(job);this.dispatch();
  });
 }
 private expire(job:Job){
  const index=this.queue.indexOf(job);if(index>=0)this.queue.splice(index,1);
  const slot=[...this.slots].find(s=>s.job===job);
  if(slot){slot.job=undefined;this.retire(slot);}
  clearTimeout(job.timer);job.reject(new Error('Candidate validation deadline exceeded'));this.dispatch();
 }
 private retire(slot:Slot){
  if(slot.retiring)return;slot.retiring=true;
  // Keep capacity reserved until the old isolate has actually terminated.
  void slot.worker.terminate().finally(()=>{this.slots.delete(slot);this.dispatch();});
 }
 private dispatch(){
  if(this.closed)return;
  while(this.queue.length){
   let slot=[...this.slots].find(s=>!s.job&&!s.retiring);
   if(!slot&&this.slots.size>=this.capacity)return;
   const job=this.queue.shift()!;
   if(performance.now()>=job.deadline){clearTimeout(job.timer);job.reject(new Error('Candidate validation deadline exceeded'));continue;}
   if(slot){slot.job=job;slot.worker.ref();slot.worker.postMessage(job.input);}
   else{
    const worker=new Worker(this.url,{workerData:job.input,resourceLimits:VALIDATION_RESOURCE_LIMITS,execArgv:[]});
    slot={worker,job};this.slots.add(slot);const current=slot;
    worker.on('message',(message:{ok:boolean;result?:ValidatedUpdate;error?:string})=>{
     const active=current.job;if(!active)return;
     if(performance.now()>=active.deadline){this.expire(active);return;}
     current.job=undefined;clearTimeout(active.timer);worker.unref();
     if(message.ok&&message.result)active.resolve(message.result);else active.reject(new Error(message.error??'Candidate validation failed'));
     this.dispatch();
    });
    const failed=(error:Error)=>{if(!this.slots.has(current)||current.retiring)return;const active=current.job;current.job=undefined;if(active){clearTimeout(active.timer);active.reject(error);}this.retire(current);};
    worker.on('error',error=>failed(new Error(`Candidate validation worker resource/error: ${error instanceof Error?error.message:String(error)}`)));
    worker.on('exit',code=>failed(new Error(`Candidate validation worker exited (${code})`)));
   }
  }
 }
 async close(){this.closed=true;for(const job of this.queue){clearTimeout(job.timer);job.reject(new Error('Candidate validation pool closed'));}this.queue=[];const slots=[...this.slots];this.slots.clear();for(const slot of slots)if(slot.job){clearTimeout(slot.job.timer);slot.job.reject(new Error('Candidate validation pool closed'));}await Promise.all(slots.map(s=>s.worker.terminate()));}
}
const sharedPool=new ValidationPool();
/** A custom trusted test worker gets an isolated pool, never the production pool. */
export async function runValidationJob(input:ValidationInput,workerURL?:URL):Promise<ValidatedUpdate>{
 if(!workerURL)return sharedPool.run(input);
 const pool=new ValidationPool(workerURL,1);try{return await pool.run(input);}finally{await pool.close();}
}
