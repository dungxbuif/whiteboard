import {MAX_FRAME_BYTES} from '@collab/contracts';
import {CollabError} from './auth.js';
export const INGRESS_FRAMES_PER_SECOND=2000;
export const INGRESS_BYTES_PER_MINUTE=64*1024*1024;
export const INGRESS_BURST_BYTES=8*1024*1024;
const SECOND_MS=1000,MINUTE_MS=60_000,MAX_IP_BUCKETS=10_000;
type Bucket={frames:number;reset:number;tokens:number;updated:number};
export class IngressLimit {
 private buckets=new Map<string,Bucket>();private sweepAt=0;
 constructor(private now:()=>number=Date.now){}
 get size(){return this.buckets.size;}
 take(ip:string,bytes:number){
  const now=this.now();
  if(now>=this.sweepAt){for(const [key,bucket]of this.buckets)if(now-bucket.updated>=MINUTE_MS)this.buckets.delete(key);this.sweepAt=now+SECOND_MS;}
  let bucket=this.buckets.get(ip);
  if(!bucket){if(this.buckets.size>=MAX_IP_BUCKETS)throw new CollabError('INGRESS_RATE_LIMITED','Ingress limiter capacity reached',429);bucket={frames:0,reset:now+SECOND_MS,tokens:INGRESS_BURST_BYTES,updated:now};this.buckets.set(ip,bucket);}
  if(bucket.reset<=now){bucket.frames=0;bucket.reset=now+SECOND_MS;}
  const available=Math.min(INGRESS_BURST_BYTES,bucket.tokens+Math.max(0,now-bucket.updated)*INGRESS_BYTES_PER_MINUTE/MINUTE_MS);
  bucket.updated=now;bucket.frames=Math.min(INGRESS_FRAMES_PER_SECOND+1,bucket.frames+1);
  if(!Number.isSafeInteger(bytes)||bytes<0)throw new CollabError('INVALID_UPDATE','Invalid frame length',400);
  // Debit before parsing or size rejection, including malformed envelopes.
  bucket.tokens=Math.max(0,available-bytes);
  if(bytes>MAX_FRAME_BYTES)throw new CollabError('INVALID_UPDATE','Frame too large',413);
  if(bucket.frames>INGRESS_FRAMES_PER_SECOND||bytes>available)throw new CollabError('INGRESS_RATE_LIMITED','Ingress frame or byte budget exceeded',429);
 }
}
