import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {readFile,readdir} from 'node:fs/promises';
import {randomUUID,createHash} from 'node:crypto';
import {once} from 'node:events';
import pg from '../../src/node_modules/pg/lib/index.js';
import WebSocket from '../../src/node_modules/ws/wrapper.mjs';
import {SignJWT} from '../../src/node_modules/jose/dist/webapi/index.js';
import {Authority} from '../../src/collab/dist/auth.js';
import {Persistence} from '../../src/collab/dist/persistence.js';
import {startServer} from '../../src/collab/dist/server.js';
import {HEARTBEAT_INTERVAL_MS} from '../../src/collab/dist/rooms.js';

const DATABASE_URL=process.env.COLLAB_TEST_DATABASE_URL??'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
const parsed=new URL(DATABASE_URL);
assert(['127.0.0.1','localhost','[::1]'].includes(parsed.hostname)&&parsed.pathname.endsWith('_test'),'Guarded loopback *_test database required');
const name='whiteboard-b4-idle-'+randomUUID().slice(0,8),schema='b4_idle_'+randomUUID().replaceAll('-','');
const containers=[],sockets=[];
const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',stdio:['ignore','pipe','pipe']}).trim();
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const admin=new pg.Pool({connectionString:DATABASE_URL});
let pool,server,created=false,network=false;
const run=(suffix,args)=>{const container=name+'-'+suffix;containers.push(container);docker('run','-d','--name',container,'--network',name,...args);return container;};
async function until(check,timeout=5000){const deadline=Date.now()+timeout;while(Date.now()<deadline){if(check())return;await pause(20);}throw new Error('Condition deadline exceeded');}
const secret='r'.repeat(32),grantSecret='g'.repeat(32);
async function sign(claims,key=secret){return new SignJWT(claims).setProtectedHeader({alg:'HS256'}).sign(new TextEncoder().encode(key));}
try {
 assert.equal(HEARTBEAT_INTERVAL_MS,25_000);
 await admin.query('CREATE SCHEMA '+schema);created=true;
 pool=new pg.Pool({connectionString:DATABASE_URL,options:'-c search_path='+schema+',public',max:8});
 const migrations=new URL('../../src/backend/migrations/',import.meta.url);
 for(const file of (await readdir(migrations)).filter(file=>file.endsWith('.up.sql')).sort())await pool.query(await readFile(new URL(file,migrations),'utf8'));
 const principalId=randomUUID(),ref={workspaceId:randomUUID(),sceneId:randomUUID()};
 await pool.query('INSERT INTO users(id,email,name) VALUES($1,$2,$3)',[principalId,principalId+'@test.invalid','Idle proof']);
 await pool.query('INSERT INTO workspaces(id,owner_id,name) VALUES($1,$2,$3)',[ref.workspaceId,principalId,'Idle proof']);
 await pool.query('INSERT INTO scenes(id,workspace_id,title,content) VALUES($1,$2,$3,$4)',[ref.sceneId,ref.workspaceId,'Idle proof',{elements:[],appState:{},files:{}}]);
 const authority=new Authority({roomSecret:secret,grantSecret,serviceSecret:'s'.repeat(32),url:'http://127.0.0.1:1'});
 authority.decide=async(_token,scope,action,requestId,bytes,expectedSequence)=>{
  const now=Math.floor(Date.now()/1000),aclVersion=(await pool.query('SELECT acl_version::text AS v FROM workspaces WHERE id=$1',[ref.workspaceId])).rows[0].v;
  const binding={...scope,action,requestId,payloadHash:createHash('sha256').update(bytes).digest('hex'),...(expectedSequence?{expectedSequence}:{})};
  return authority.verifyDecision(await sign({...binding,iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:randomUUID(),sub:principalId,principalId,principalType:'user',grantId:'membership:'+ref.workspaceId+':'+principalId,access:'edit',aclVersion},grantSecret),binding);
 };
 server=await startServer(new Persistence(pool,authority,{promoteV2:true}),{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['https://whiteboard.example'],updatesPerMinute:120,maxBufferedBytes:4194304});
 const port=new URL(server.httpURL).port;
 docker('network','create',name);network=true;
 run('forward',['--network-alias','collab','--network-alias','backend','--network-alias','mcp','--add-host','host.docker.internal:host-gateway','--init','--read-only','--cap-drop','ALL','--security-opt','no-new-privileges:true','whiteboard-e2-collab:local','node','-e',`const net=require('net');net.createServer(d=>{const u=net.connect(${port},'host.docker.internal');d.pipe(u).pipe(d);d.on('error',()=>u.destroy());u.on('error',()=>d.destroy());d.on('close',()=>u.destroy());u.on('close',()=>d.destroy());}).listen(8082,'0.0.0.0');`]);
 const frontend=run('nginx',['--read-only','--tmpfs','/tmp:size=64m','--cap-drop','ALL','--security-opt','no-new-privileges:true','-p','127.0.0.1::80','whiteboard-e2-frontend:local']);
 const image=docker('image','inspect','whiteboard-e2-frontend:local','--format','{{.Id}}');
 const base='http://127.0.0.1:'+docker('port',frontend,'80/tcp').split(':').at(-1);await pause(700);
 assert.equal((await fetch(base+'/healthz')).status,200);
 const now=Math.floor(Date.now()/1000),token=await sign({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:randomUUID(),sub:principalId,...ref,principalId,principalType:'user',grantId:'membership:'+ref.workspaceId+':'+principalId,access:'edit',aclVersion:'2'});
 async function connect(autoPong){
  const socket=new WebSocket(base.replace('http:','ws:')+'/collab',{origin:'https://whiteboard.example',autoPong});sockets.push(socket);
  const messages=[],pings=[],closes=[];socket.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});socket.on('ping',()=>pings.push(Date.now()));socket.on('close',code=>closes.push({code,at:Date.now()}));
  await once(socket,'open');socket.send(JSON.stringify({type:'auth',requestId:'admit',payload:{token}}));await until(()=>messages.some(m=>m.type==='sync_chunk'));
  return {socket,messages,pings,closes,id:messages.find(m=>m.type==='ready').payload.connectionId};
 }
 const live=await connect(true),dead=await connect(false);dead.socket.send(JSON.stringify({type:'voice_join',requestId:'voice',payload:{}}));
 await until(()=>dead.messages.some(m=>m.type==='voice_admission'&&m.payload.admitted));
 const started=Date.now();
 while(Date.now()-started<160_000){await pause(1000);assert.equal(live.socket.readyState,WebSocket.OPEN,'Idle live socket closed');if(Date.now()-started>60_000){assert.equal(dead.socket.readyState,WebSocket.CLOSED,'Unresponsive peer survived heartbeat');assert(!live.messages.filter(m=>m.type==='roster').at(-1).payload.participants.some(p=>p.connectionId===dead.id));}}
 assert(live.pings.length>=6);assert.equal(live.closes.length,0);assert.equal((await fetch(server.httpURL+'/readyz')).status,200);
 live.socket.send(JSON.stringify({type:'voice_join',requestId:'after-idle',payload:{}}));await until(()=>live.messages.some(m=>m.requestId==='after-idle'&&m.type==='voice_admission'));
 assert.deepEqual(live.messages.find(m=>m.requestId==='after-idle'&&m.type==='voice_admission').payload.peers,[]);
 console.log(JSON.stringify({pass:true,elapsedMs:Date.now()-started,heartbeatIntervalMs:HEARTBEAT_INTERVAL_MS,livePings:live.pings.length,liveCloses:live.closes.length,deadCloseMs:dead.closes[0].at-started,deadCloseCode:dead.closes[0].code,staleVoiceSlotReleased:true,ready:true,nginxImage:image,authority:'signed upstream fixture; actual current compiled runtime + isolated PostgreSQL schema + unchanged nginx image'}));
} finally {
 for(const socket of sockets)socket.terminate();
 for(const container of containers.reverse()){try{docker('rm','-f',container);}catch{}}
 if(network){try{docker('network','rm',name);}catch{}}
 await server?.close();await pool?.end();if(created)await admin.query('DROP SCHEMA '+schema+' CASCADE');await admin.end();
}
