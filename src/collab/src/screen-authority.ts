import type {SceneRef} from '@collab/contracts';
import {CollabError} from './auth.js';

type FeatureAction='screen:publish'|'presentation:host';
type FeatureAuthority={config:{url:string;serviceSecret:string};requireFeature?:(token:string,ref:SceneRef,action:FeatureAction)=>Promise<void>};
export async function requireFeature(authority:FeatureAuthority,roomToken:string,ref:SceneRef,action:FeatureAction):Promise<void>{
 if(authority.requireFeature)return authority.requireFeature(roomToken,ref,action);
 const config=authority.config;
 let response:Response;
 try{response=await fetch(new URL('/internal/authority/features',config.url),{method:'POST',headers:{authorization:'Bearer '+config.serviceSecret,'content-type':'application/json'},body:JSON.stringify({roomToken,...ref,action}),signal:AbortSignal.timeout(2500)});}catch{throw new CollabError('UNAVAILABLE','Feature authority unavailable',503,true);}
 if(response.ok)return;
 if(response.status===401)throw new CollabError('UNAUTHENTICATED','Invalid or expired credential',401);
 if([403,404].includes(response.status))throw new CollabError('FORBIDDEN','Feature is not available',403);
 throw new CollabError('UNAVAILABLE','Feature authority unavailable',503,true);
}
export function requireScreenFeature(authority:FeatureAuthority,roomToken:string,ref:SceneRef){return requireFeature(authority,roomToken,ref,'screen:publish');}
