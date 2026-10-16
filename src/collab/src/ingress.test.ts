import {expect,it} from 'vitest';
import {IngressLimit} from './ingress.js';
import {RateLimit} from './rate-limit.js';
it('allows fifty clients at 20Hz cursors plus 1Hz small updates without consuming operation budgets',()=>{
 let now=0;const ingress=new IngressLimit(()=>now),operations=new RateLimit(()=>now);
 for(let second=0;second<60;second++){
  for(let tick=0;tick<20;tick++){
   now=second*1000+tick*50;
   for(let client=0;client<50;client++){
    ingress.take('shared-nat',128);operations.take('awareness:'+client,30,1000);
    if(tick===0){ingress.take('shared-nat',1024);operations.take('write:'+client,120);}
   }
  }
 }
 expect(()=>operations.take('auth:shared-nat',60)).not.toThrow();
 expect(()=>operations.take('write:0',120)).not.toThrow();
});
it('bounds frames before parsing, bounds raw bytes with refill, and expires IP entries',()=>{
 let now=0;const ingress=new IngressLimit(()=>now);
 for(let i=0;i<2000;i++)ingress.take('flood',1);
 expect(()=>ingress.take('flood',1)).toThrow(/ingress/i);
 for(let i=0;i<8;i++)ingress.take('bytes',1024*1024);
 expect(()=>ingress.take('bytes',1)).toThrow(/ingress/i);
 now=1000;expect(()=>ingress.take('bytes',1024*1024)).not.toThrow();
 expect(()=>ingress.take('oversized',1024*1024+1)).toThrow(/frame/i);
 now=61_000;expect(()=>ingress.take('flood',1)).not.toThrow();expect(ingress.size).toBe(1);
});
it('operation windows expire and reject malformed limits',()=>{
 let now=0;const limits=new RateLimit(()=>now);limits.take('old',1,1000);expect(()=>limits.take('old',1,1000)).toThrow();
 now=2000;limits.take('new',1,1000);expect(limits.size).toBe(1);
 expect(()=>limits.take('invalid',0)).toThrow(/limit/i);expect(()=>limits.take('invalid',1,0)).toThrow(/limit/i);
});
