import {Worker} from 'node:worker_threads';
import type {ValidatedUpdate} from './document.js';
export const VALIDATION_DEADLINE_MS=2000;
export const VALIDATION_RESOURCE_LIMITS=Object.freeze({maxOldGenerationSizeMb:256,maxYoungGenerationSizeMb:16,stackSizeMb:4});
export type ValidationInput={snapshot:Uint8Array;update:Uint8Array};

/** Internal worker runner; the URL is supplied only by trusted server/test code. */
export function runValidationJob(input:ValidationInput,workerURL=new URL('../dist/validation-worker.js',import.meta.url)):Promise<ValidatedUpdate> {
  return new Promise((resolve,reject)=>{
    const started=performance.now();
    const worker=new Worker(workerURL,{workerData:input,resourceLimits:VALIDATION_RESOURCE_LIMITS,execArgv:[]});
    let finished=false;
    const finish=(error:Error|null,result?:ValidatedUpdate)=>{
      if(finished)return;finished=true;clearTimeout(timer);void worker.terminate();
      if(error)reject(error);else resolve(result!);
    };
    const timer=setTimeout(()=>finish(new Error('Candidate validation deadline exceeded')),Math.max(0,VALIDATION_DEADLINE_MS-(performance.now()-started)));
    worker.once('message',(message:{ok:boolean;result?:ValidatedUpdate;error?:string})=>{
      if(performance.now()-started>=VALIDATION_DEADLINE_MS){finish(new Error('Candidate validation deadline exceeded'));return;}
      if(message.ok&&message.result)finish(null,message.result);else finish(new Error(message.error??'Candidate validation failed'));
    });
    worker.once('error',error=>finish(new Error(`Candidate validation worker resource/error: ${error instanceof Error?error.message:String(error)}`)));
    worker.once('exit',code=>{if(!finished)finish(new Error(`Candidate validation worker exited (${code})`));});
  });
}
