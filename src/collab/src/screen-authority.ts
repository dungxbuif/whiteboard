import type {SceneRef} from '@collab/contracts';
import {CollabError} from './auth.js';

type ScreenAuthority={config:{url:string;serviceSecret:string};requireFeature?:(token:string,ref:SceneRef,action:'screen:publish')=>Promise<void>};
export async function requireScreenFeature(authority:ScreenAuthority,roomToken:string,ref:SceneRef):Promise<void>{
 if(authority.requireFeature)return authority.requireFeature(roomToken,ref,'screen:publish');
 const config=authority.config;
 let response:Response;
 try{response=await fetch(new URL('/internal/authority/features',config.url),{method:'POST',headers:{authorization:'Bearer '+config.serviceSecret,'content-type':'application/json'},body:JSON.stringify({roomToken,...ref,action:'screen:publish'}),signal:AbortSignal.timeout(2500)});}catch{throw new CollabError('UNAVAILABLE','Feature authority unavailable',503,true);}
 if(response.ok)return;
 if(response.status===401)throw new CollabError('UNAUTHENTICATED','Invalid or expired credential',401);
 if([403,404].includes(response.status))throw new CollabError('FORBIDDEN','Feature is not available',403);
 throw new CollabError('UNAVAILABLE','Feature authority unavailable',503,true);
}
