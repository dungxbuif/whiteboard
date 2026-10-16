import {Pool} from 'pg';
import {readFile,readdir} from 'node:fs/promises';
import {SignJWT} from 'jose';
import {createHash,randomUUID} from 'node:crypto';
import {Authority} from '/app/collab/dist/auth.js';
import {Persistence} from '/app/collab/dist/persistence.js';
import {startServer} from '/app/collab/dist/server.js';
const pool=new Pool({connectionString:'postgres://fixture:fixture@db:5432/packaging_test'});
for(let i=0;;i++){try{await pool.query('SELECT 1');break;}catch(e){if(i===30)throw e;await new Promise(r=>setTimeout(r,500));}}
for(const file of (await readdir('/migrations')).filter(f=>f.endsWith('.up.sql')).sort())await pool.query(await readFile('/migrations/'+file,'utf8'));
const user='11111111-1111-4111-8111-111111111111',workspaceId='22222222-2222-4222-8222-222222222222',sceneId='33333333-3333-4333-8333-333333333333';
await pool.query("INSERT INTO users(id,email,name) VALUES($1,'packaging@test.invalid','Packaging')",[user]);
await pool.query("INSERT INTO workspaces(id,owner_id,name) VALUES($1,$2,'Packaging')",[workspaceId,user]);
await pool.query("INSERT INTO scenes(id,workspace_id,title,content) VALUES($1,$2,'Packaging',$3)",[sceneId,workspaceId,JSON.stringify({elements:[],appState:{},files:{}})]);
const authority=new Authority({roomSecret:'r'.repeat(32),grantSecret:'g'.repeat(32),serviceSecret:'s'.repeat(32),url:'http://127.0.0.1:1'});
authority.decide=async(_token,scope,action,requestId,bytes,expectedSequence)=>{
 const now=Math.floor(Date.now()/1000),aclVersion=(await pool.query('SELECT acl_version::text AS v FROM workspaces WHERE id=$1',[workspaceId])).rows[0].v;
 const binding={...scope,action,requestId,payloadHash:createHash('sha256').update(bytes).digest('hex'),...(expectedSequence?{expectedSequence}:{})};
 const claims={...binding,iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:randomUUID(),sub:user,principalId:user,principalType:'user',grantId:'membership:'+workspaceId+':'+user,access:'edit',aclVersion};
 return authority.verifyDecision(await new SignJWT(claims).setProtectedHeader({alg:'HS256'}).sign(new TextEncoder().encode('g'.repeat(32))),binding);
};
const server=await startServer(new Persistence(pool,authority,{promoteV2:true}),{host:'0.0.0.0',port:8082,internalHost:'127.0.0.1',internalPort:8083,internalSecret:'i'.repeat(32),allowedOrigins:['https://whiteboard.example'],updatesPerMinute:120,maxBufferedBytes:4194304});
console.log('Collab fixture ready');
process.on('SIGTERM',async()=>{await server.close();await pool.end();});
