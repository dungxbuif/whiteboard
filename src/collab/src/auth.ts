import {jwtVerify,type JWTPayload} from 'jose';
import {createHash} from 'node:crypto';
import type {SceneRef} from '@collab/contracts';
export class CollabError extends Error {
 constructor(public code:string,message=code,public status=403,public retryable=false){super(message);this.name='CollabError';}
}
export type Principal={roomToken:string;principalId:string;principalType:string};
export type Decision=JWTPayload & SceneRef & {purpose:string;principalId:string;principalType:string;grantId:string;grantVersion?:string;access:'view'|'edit';aclVersion:string;action?:string;requestId?:string;payloadHash?:string;expectedSequence?:string;exp:number;iat:number};
export type PresentationDecision=JWTPayload & SceneRef & {purpose:'presentation-host'|'presentation-pending'|'presentation-audience';sessionId:string;principalId:string;principalType:'user'|'public';grantId:string;grantVersion?:string;displayName:string;access:'edit'|'pending'|'view';aclVersion:string;exp:number;iat:number};
export const hash=(bytes:Uint8Array)=>createHash('sha256').update(bytes).digest('hex');
export const revision=(value:unknown):value is string=>typeof value==='string'&&/^(0|[1-9][0-9]*)$/.test(value)&&BigInt(value)<=9223372036854775807n;
const uuid=(value:unknown):value is string=>typeof value==='string'&&/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value)&&value!=='00000000-0000-0000-0000-000000000000';
export function requestID(value:unknown):asserts value is string {if(typeof value!=='string'||!/^([\x21-\x7e]| ){1,128}$/.test(value))throw new CollabError('INVALID_UPDATE','Invalid request ID',400);}
export class Authority {
 constructor(public config:{roomSecret:string;grantSecret:string;serviceSecret:string;url:string}){}
 private async verify(token:string,decision:boolean):Promise<Decision>{
  try {
   if(typeof token!=='string'||token.length>16384)throw new Error();
   const {payload:p}=await jwtVerify(token,new TextEncoder().encode(decision?this.config.grantSecret:this.config.roomSecret),{algorithms:['HS256'],issuer:'whiteboard-api',audience:decision?'whiteboard-mutation':'whiteboard-collab',requiredClaims:['exp','iat','jti','sub'],clockTolerance:0});
   const audience=decision?'whiteboard-mutation':'whiteboard-collab',now=Date.now()/1000;
   if(!(p.aud===audience||(Array.isArray(p.aud)&&p.aud.length===1&&p.aud[0]===audience))||p.purpose!==(decision?'mutation-decision':'room')||!Number.isInteger(p.iat)||!Number.isInteger(p.exp)||p.iat!>now||p.exp!-p.iat!>(decision?5:600)||p.exp!<=p.iat!||typeof p.jti!=='string'||!p.jti||p.sub!==p.principalId||typeof p.principalId!=='string'||!p.principalId||!uuid(p.workspaceId)||!uuid(p.sceneId)||!['view','edit'].includes(String(p.access))||!revision(p.aclVersion)||p.aclVersion==='0')throw new Error();
   if(p.principalType==='user'){
    if(!uuid(p.principalId)||p.grantId!=='membership:'+p.workspaceId+':'+p.principalId)throw new Error();
   }else if(p.principalType==='public'){
    if(!p.principalId.startsWith('guest:')||p.principalId.length<=6||!uuid(p.grantId)||!revision(p.grantVersion)||p.grantVersion==='0')throw new Error();
   }else if(p.principalType==='agent'){
    if(!decision||!uuid(p.grantId)||p.principalId!=='agent:'+p.grantId||p.grantVersion!==undefined||!['scene:command','scene:read','scene:export'].includes(String(p.action)))throw new Error();
   }else throw new Error();
   if(decision){if(!['scene:update','scene:command','scene:read','scene:export','version:create','version:restore','library:insert'].includes(String(p.action))||typeof p.payloadHash!=='string'||!/^[a-f0-9]{64}$/.test(p.payloadHash))throw new Error();requestID(p.requestId);if(p.expectedSequence!==undefined&&!revision(p.expectedSequence))throw new Error();}
   return p as Decision;
  }catch {throw new CollabError('UNAUTHENTICATED','Invalid or expired credential',401);}
 }
 verifyRoom(token:string):Promise<Decision>{return this.verify(token,false);}
 async verifyPresentation(token:string,purpose?:PresentationDecision['purpose']):Promise<PresentationDecision>{
  try{
   if(typeof token!=='string'||token.length>16384)throw new Error();
   const {payload:p}=await jwtVerify(token,new TextEncoder().encode(this.config.roomSecret),{algorithms:['HS256'],issuer:'whiteboard-api',audience:'whiteboard-presentation',requiredClaims:['exp','iat','jti','sub'],clockTolerance:0});
   const now=Date.now()/1000;if(!['presentation-host','presentation-pending','presentation-audience'].includes(String(p.purpose))||purpose&&p.purpose!==purpose||!Number.isInteger(p.iat)||!Number.isInteger(p.exp)||p.iat!>now||p.exp!<=p.iat!||p.exp!-p.iat!>(p.purpose==='presentation-pending'?300:600)||typeof p.jti!=='string'||!p.jti||p.sub!==p.principalId||!uuid(p.workspaceId)||!uuid(p.sceneId)||!uuid(p.sessionId)||typeof p.principalId!=='string'||!p.principalId||typeof p.displayName!=='string'||p.displayName.length>100)throw new Error();
   if(p.purpose==='presentation-host'){
    if(p.principalType!=='user'||!uuid(p.principalId)||p.grantId!=='membership:'+p.workspaceId+':'+p.principalId||p.grantVersion!==''||p.access!=='edit'||!revision(p.aclVersion)||p.aclVersion==='0')throw new Error();
   }else if(p.purpose==='presentation-pending'){
    if(p.principalType!=='public'||!p.principalId.startsWith('presentation:')||!uuid(p.grantId)||!revision(p.grantVersion)||p.grantVersion==='0'||p.access!=='pending'||p.aclVersion!=='0')throw new Error();
   }else if(p.principalType!=='public'||!p.principalId.startsWith('presentation:')||p.grantId!=='session:'+p.sessionId||!revision(p.grantVersion)||p.grantVersion==='0'||p.access!=='view'||!revision(p.aclVersion)||p.aclVersion==='0')throw new Error();
   return p as PresentationDecision;
  }catch{throw new CollabError('UNAUTHENTICATED','Invalid or expired presentation credential',401);}
 }
 async verifyDecision(token:string,binding:Record<string,unknown>):Promise<Decision>{
  const decision=await this.verify(token,true);
  for(const [key,value] of Object.entries(binding))if(decision[key]!==value)throw new CollabError('FORBIDDEN','Decision binding mismatch');
  if(!['scene:read','scene:export'].includes(String(decision.action))&&decision.access!=='edit')throw new CollabError('FORBIDDEN');
  return decision;
 }
 async decide(token:string,ref:SceneRef,action:string,requestId:string,bytes:Uint8Array,expectedSequence?:string):Promise<Decision>{
  const binding={...ref,action,requestId,payloadHash:hash(bytes),...(expectedSequence!==undefined?{expectedSequence}:{})};
  let response:Response;
  try{response=await fetch(new URL('/internal/authority/decisions',this.config.url),{method:'POST',headers:{authorization:'Bearer '+this.config.serviceSecret,'content-type':'application/json'},body:JSON.stringify({roomToken:token,...binding}),signal:AbortSignal.timeout(2500)});}catch{throw new CollabError('UNAVAILABLE','Authority unavailable',503,true);}
  if(!response.ok){if(response.status===401)throw new CollabError('UNAUTHENTICATED','Invalid or expired credential',401);if([403,404].includes(response.status))throw new CollabError('FORBIDDEN');throw new CollabError('UNAVAILABLE','Authority unavailable',503,true);}
  const body=await response.json() as {token?:string};
  return this.verifyDecision(body.token??'',binding);
 }
}
export function authorizeSync(access:string,subtype:number):void {
 if(![0,1,2].includes(subtype))throw new CollabError('INVALID_UPDATE','Unknown sync subtype',400);
 if(subtype!==0&&access!=='edit')throw new CollabError('FORBIDDEN');
}
