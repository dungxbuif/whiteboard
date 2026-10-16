import {CollabError} from './auth.js';
const DEFAULT_WINDOW_MS=60_000,SWEEP_INTERVAL_MS=1000,MAX_BUCKETS=10_000;
export class RateLimit {
 private buckets=new Map<string,{count:number;reset:number}>();private sweepAt=0;
 constructor(private now:()=>number=Date.now){}
 get size(){return this.buckets.size;}
 take(key:string,limit:number,window=DEFAULT_WINDOW_MS){
  if(!Number.isSafeInteger(limit)||limit<1||!Number.isSafeInteger(window)||window<1)throw new Error('Invalid rate limit');
  const now=this.now();
  if(now>=this.sweepAt){for(const [id,bucket]of this.buckets)if(bucket.reset<=now)this.buckets.delete(id);this.sweepAt=now+SWEEP_INTERVAL_MS;}
  let value=this.buckets.get(key);
  if(!value||value.reset<=now){if(!value&&this.buckets.size>=MAX_BUCKETS)throw new CollabError('RATE_LIMITED','Rate limiter capacity reached',429,true);value={count:0,reset:now+window};this.buckets.set(key,value);}
  value.count=Math.min(limit+1,value.count+1);
  if(value.count>limit)throw new CollabError('RATE_LIMITED','Rate limit reached; retain pending updates',429,true);
 }
}
