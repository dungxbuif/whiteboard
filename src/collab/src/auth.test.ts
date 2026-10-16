import {describe,it,expect} from 'vitest';
import {Authority,authorizeSync} from './auth.js';
import {signed,secret,grantSecret} from '../tests/fixture.js';
const authority=new Authority({roomSecret:secret,grantSecret,serviceSecret:'s'.repeat(32),url:'http://127.0.0.1:1'});
const now=Math.floor(Date.now()/1000);
const claims={iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'id',sub:'11111111-1111-4111-8111-111111111111',principalType:'user',principalId:'11111111-1111-4111-8111-111111111111',workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333',grantId:'membership:22222222-2222-4222-8222-222222222222:11111111-1111-4111-8111-111111111111',access:'view',aclVersion:'9007199254740993'};
describe('authority',()=>{
 it('accepts exact room claims with lossless ACL version',async()=>expect((await authority.verifyRoom(await signed(claims))).aclVersion).toBe('9007199254740993'));
 it('viewer may read state vector but every sync write subtype is forbidden',()=>{expect(()=>authorizeSync('view',0)).not.toThrow();for(const type of [1,2])expect(()=>authorizeSync('view',type)).toThrowError(/FORBIDDEN/);});
 it.each([{exp:now-1},{purpose:'scene-session'},{aud:['whiteboard-collab','other']},{principalType:'agent'},{sub:'other'},{iat:now+1},{aclVersion:'1e3'}])('rejects invalid or expired room %j',async patch=>expect(authority.verifyRoom(await signed({...claims,...patch}))).rejects.toMatchObject({code:'UNAUTHENTICATED'}));
 it('revoked authority and authority outage deny a decision',async()=>expect(authority.decide('invalid',{workspaceId:claims.workspaceId,sceneId:claims.sceneId},'scene:update','request',new Uint8Array())).rejects.toMatchObject({code:'UNAVAILABLE'}));
});
