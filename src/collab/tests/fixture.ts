import {Pool} from 'pg';
import {randomUUID} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {SignJWT} from 'jose';
import {createHash} from 'node:crypto';
import {Authority} from '../src/auth.js';
export const secret='r'.repeat(32), grantSecret='g'.repeat(32);
export async function signed(claims:Record<string,unknown>,key=secret){return new SignJWT(claims).setProtectedHeader({alg:'HS256'}).sign(new TextEncoder().encode(key));}
export async function fixture(){
 const url=postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
 const parsed=new URL(url);if(!['127.0.0.1','localhost','[::1]'].includes(parsed.hostname)||!parsed.pathname.endsWith('_test'))throw new Error('Disposable loopback *_test database required');
 const admin=new Pool({connectionString:url}),schema='b2_'+randomUUID().replaceAll('-','');
 await admin.query('CREATE SCHEMA '+schema);
 const pool=new Pool({connectionString:url,options:'-c search_path='+schema+',public',application_name:schema,max:8});
 const {readdir}=await import('node:fs/promises');const files=(await readdir(new URL('../../backend/migrations/',import.meta.url))).filter(f=>/^00000[1-8]_.*up.sql$/.test(f)).sort();
 for(const file of files)await pool.query(await readFile(new URL('../../backend/migrations/'+file,import.meta.url),'utf8'));
 // Before migration 9 exists, a faithful test-only schema lets API behavior fail rather than fixture setup.
 const migration=new URL('../../backend/migrations/000009_collab_documents.up.sql',import.meta.url);
 try{await pool.query(await readFile(migration,'utf8'));}catch(error){if((error as NodeJS.ErrnoException).code!=='ENOENT')throw error;}
 const user=randomUUID(),workspaceId=randomUUID(),sceneId=randomUUID();
 await pool.query('INSERT INTO users(id,email,name) VALUES($1,$2,$3)',[user,user+'@test.invalid','Tester']);
 await pool.query('INSERT INTO workspaces(id,owner_id,name) VALUES($1,$2,$3)',[workspaceId,user,'Test']);
 await pool.query('INSERT INTO scenes(id,workspace_id,title,content) VALUES($1,$2,$3,$4)',[sceneId,workspaceId,'Scene',JSON.stringify({elements:[],appState:{},files:{}})]);
 const ref={workspaceId,sceneId},principal={roomToken:'fixture',principalId:user,principalType:'user'};
 const authority=new Authority({roomSecret:secret,grantSecret,serviceSecret:'s'.repeat(32),url:'http://127.0.0.1:1'});
 // Replace only the upstream HTTP boundary; signatures, database version locks and candidate workers remain real.
 authority.decide=async(_token,scope,action,requestId,bytes,expectedSequence)=>{
 const version=(await pool.query('SELECT acl_version::text AS v FROM workspaces WHERE id=$1',[workspaceId])).rows[0].v;
 const now=Math.floor(Date.now()/1000);
 const claims={iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:randomUUID(),sub:principal.principalId,...scope,principalId:principal.principalId,principalType:principal.principalType,grantId:'membership:'+workspaceId+':'+principal.principalId,access:'edit',aclVersion:version,action,requestId,payloadHash:createHash('sha256').update(bytes).digest('hex'),...(expectedSequence?{expectedSequence}:{})};
 return authority.verifyDecision(await signed(claims,grantSecret),{...scope,action,requestId,payloadHash:claims.payloadHash,...(expectedSequence?{expectedSequence}:{})});
 };
 return {pool,admin,schema,ref,principal,authority,async close(){await pool.end();await admin.query('DROP SCHEMA '+schema+' CASCADE');await admin.end();}};
}

export const rectangle=(id='shape')=>({id,type:'rectangle' as const,x:0,y:0,width:100,height:100,angle:0,strokeColor:'#111111',backgroundColor:'transparent',fillStyle:'solid' as const,strokeWidth:1,strokeStyle:'solid' as const,roughness:1,opacity:100,groupIds:[] as string[],frameId:null,roundness:null,seed:1,version:1,versionNonce:1,index:null,isDeleted:false,boundElements:null,updated:1,link:null,locked:false});
