import {CollabError} from './auth.js';
export class RateLimit {
 private buckets=new Map<string,{count:number;reset:number}>();
 take(key:string,limit:number,window=60000){const now=Date.now();let value=this.buckets.get(key);if(!value||value.reset<=now){value={count:0,reset:now+window};this.buckets.set(key,value);}if(++value.count>limit)throw new CollabError('RATE_LIMITED','Rate limit reached; retain pending updates',429,true);if(this.buckets.size>10000)for(const [key,value]of this.buckets)if(value.reset<=now)this.buckets.delete(key);}
}
