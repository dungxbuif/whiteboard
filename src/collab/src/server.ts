import {createServer,type Server,type IncomingMessage} from 'node:http';
import {timingSafeEqual} from 'node:crypto';
import {pathToFileURL} from 'node:url';
import {Pool,type PoolClient} from 'pg';
import {WebSocketServer} from 'ws';
import {MAX_FRAME_BYTES} from '@collab/contracts';
import {Authority,CollabError} from './auth.js';
import {Persistence} from './persistence.js';
import {Rooms} from './rooms.js';
import {Compactor} from './compaction.js';
import {RateLimit} from './rate-limit.js';
import {RuntimeOwnership,OWNER_KEY,HANDOFF_KEY} from './ownership.js';
import {ActivityProjector,PostgresActivityProjectionStore} from './activity-projector.js';
import {VersionCheckpointer} from './version-checkpointer.js';
export type ServerConfig={host:string;port:number;internalHost:string;internalPort:number;internalSecret:string;allowedOrigins:string[];updatesPerMinute:number;maxBufferedBytes:number};
const secretEquals=(a:string,b:string)=>Buffer.byteLength(a)===Buffer.byteLength(b)&&timingSafeEqual(Buffer.from(a),Buffer.from(b));
async function body(request:IncomingMessage):Promise<unknown>{const chunks:Buffer[]=[];let size=0;for await(const chunk of request){size+=chunk.length;if(size>1500000)throw new CollabError('INVALID_UPDATE','Body too large',413);chunks.push(chunk);}try{return JSON.parse(Buffer.concat(chunks).toString('utf8'));}catch{throw new CollabError('INVALID_UPDATE','Invalid JSON',400);}}
export async function startServer(store:Persistence,config:ServerConfig){
 // Bound queued acquisitions too, including a reconnect waiting while shutdown starts.
 store.pool.options.connectionTimeoutMillis=Math.min(store.pool.options.connectionTimeoutMillis||3000,3000);
 const ownership=new RuntimeOwnership();
 const rooms=new Rooms(store,config);
 let listening=false,closed=false,listener:PoolClient|undefined,reconnect:NodeJS.Timeout|undefined,connecting:Promise<void>|undefined;
 const scheduleReconnect=()=>{
  if(!closed&&!reconnect&&!connecting)reconnect=setTimeout(()=>{reconnect=undefined;void subscribe().catch(()=>{});},1000);
 };
 const lost=(source?:PoolClient)=>{
  if(source&&source!==listener)return;
  ownership.invalidate();listening=false;rooms.invalidateWorkspace(undefined,'ACL_RESYNC_REQUIRED');
  if(listener){const old=listener;listener=undefined;old.release(true);}
  scheduleReconnect();
 };
 function subscribe():Promise<void>{
  if(connecting)return connecting;
  connecting=(async()=>{
   const client=await store.pool.connect();
   if(closed){client.release(true);return;}
   listener=client;
   client.on('error',()=>lost(client));client.on('end',()=>lost(client));
   await client.query("SET statement_timeout='5s'");
   const owned=await client.query(`SELECT pg_try_advisory_lock(${OWNER_KEY}) AS owned`);
   if(!owned.rows[0].owned)throw new Error('Exactly one collab process may own a database schema');
   await client.query(`SELECT pg_advisory_lock(${HANDOFF_KEY})`);
   const lease=(await client.query('SELECT pg_backend_pid() AS pid,backend_start::text AS "backendStart" FROM pg_stat_activity WHERE pid=pg_backend_pid()')).rows[0];
   await client.query('SELECT document_id,legacy_content,legacy_content_version,snapshot_sequence FROM collab_documents LIMIT 0');
   await client.query(`SELECT pg_advisory_unlock(${HANDOFF_KEY})`);
   // Keep acl_changed as the final statement: ownership probes identify this
   // dedicated connection by its current query while both channels stay live.
   await client.query('LISTEN feature_changed');await client.query('LISTEN acl_changed');
   if(closed||listener!==client)throw new CollabError('UNAVAILABLE','Ownership acquisition interrupted',503,true);
   ownership.activate(lease);store.bindRuntimeOwnership(ownership);
   client.on('notification',message=>{try{
    const value=JSON.parse(message.payload??'');if(typeof value.workspaceId!=='string')throw new Error();
    if(message.channel==='feature_changed'){
     if(Object.keys(value).some(key=>!['workspaceId','sceneId','eventCursor'].includes(key))||typeof value.eventCursor!=='string'||!/^[1-9]\d*$/.test(value.eventCursor)||(value.sceneId!==undefined&&typeof value.sceneId!=='string'))throw new Error();
     rooms.featureChanged(value.workspaceId,value.sceneId,value.eventCursor);return;
    }
    rooms.invalidateWorkspace(value.workspaceId);
   }catch{if(message.channel==='acl_changed')rooms.invalidateWorkspace(undefined,'ACL_RESYNC_REQUIRED');}});
   listening=true;
  })().catch(error=>{lost();throw error;}).finally(()=>{connecting=undefined;if(!listening)scheduleReconnect();});
  return connecting;
 }
 const publicServer=createServer(async(request,response)=>{
  if(request.url==='/healthz'){response.writeHead(200).end('ok');return;}
  if(request.url==='/readyz'){try{if(!listening)throw new Error();await store.pool.query('SELECT 1');response.writeHead(200).end('ready');}catch{response.writeHead(503).end('not ready');}return;}
  response.writeHead(404).end();
 });
 const wss=new WebSocketServer({noServer:true,maxPayload:MAX_FRAME_BYTES,perMessageDeflate:false});
 publicServer.on('upgrade',(request,socket,head)=>{
  if(!listening||request.url!=='/collab'||!config.allowedOrigins.includes(request.headers.origin??'')){socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n');socket.destroy();return;}
  wss.handleUpgrade(request,socket,head,ws=>rooms.attach(ws,request.socket.remoteAddress??'unknown'));
 });
 const internalRates=new RateLimit();
 const internalServer=createServer(async(request,response)=>{
  let requestId='unknown';
  try{
   if(request.method!=='POST'||!/^\/internal\/scenes\/[0-9a-f-]{36}\/(commands|projection|activity|export-projection|versions)$/.test(request.url??'')){response.writeHead(404).end();return;}
   ownership.assert();
   internalRates.take('ip:'+request.socket.remoteAddress,600);
   if(!secretEquals(request.headers.authorization??'','Bearer '+config.internalSecret))throw new CollabError('UNAUTHENTICATED','Invalid service credential',401);
   const envelope=await body(request) as Record<string,unknown>;
   const operation=request.url!.split('/').pop();
   if(operation==='versions'){
    const allowed=['grant','workspaceId','sceneId','input'];
    if(!envelope||Object.keys(envelope).some(k=>!allowed.includes(k))||typeof envelope.grant!=='string'||typeof envelope.workspaceId!=='string'||typeof envelope.sceneId!=='string'||request.url!==`/internal/scenes/${envelope.sceneId}/versions`||typeof envelope.input!=='object'||envelope.input===null)throw new CollabError('INVALID_UPDATE','Invalid version envelope',400);
    const result=await store.captureVersion({workspaceId:envelope.workspaceId,sceneId:envelope.sceneId},envelope.grant,envelope.input as import('./versions.js').CaptureVersionInput);
    response.writeHead(200,{'content-type':'application/json'}).end(JSON.stringify(result));return;
   }
   if(operation!=='commands'){
    const allowed=['grant','workspaceId','sceneId',...(operation==='activity'?['cursor','limit']:[])];
    if(!envelope||Object.keys(envelope).some(k=>!allowed.includes(k))||typeof envelope.grant!=='string'||typeof envelope.workspaceId!=='string'||typeof envelope.sceneId!=='string'||request.url!==`/internal/scenes/${envelope.sceneId}/${operation}`)throw new CollabError('INVALID_UPDATE','Invalid read envelope',400);
    const ref={workspaceId:envelope.workspaceId,sceneId:envelope.sceneId};
    const result=operation==='activity'?await store.readActivity(envelope.grant,ref,envelope.cursor as string|undefined,envelope.limit as number|undefined):await store.readProjection(envelope.grant,ref,operation==='export-projection');
    response.writeHead(200,{'content-type':'application/json'}).end(JSON.stringify(result));return;
   }
   const data=envelope as {grant:string;mutationBase64:string};if(!data||Object.keys(data).some(k=>!['grant','mutationBase64'].includes(k))||typeof data.grant!=='string'||typeof data.mutationBase64!=='string'||!data.mutationBase64.length)throw new CollabError('INVALID_UPDATE','Invalid command envelope',400);
   const bytes=Buffer.from(data.mutationBase64,'base64');if(bytes.toString('base64')!==data.mutationBase64)throw new CollabError('INVALID_UPDATE','Noncanonical base64',400);
   const {parseMutation}=await import('./commands.js');const mutation=parseMutation(bytes);requestId=mutation.requestId;
   if(request.url!== '/internal/scenes/'+mutation.sceneId+'/commands')throw new CollabError('FORBIDDEN');
   const receipt=await store.applyCommands(data.grant,bytes);response.writeHead(200,{'content-type':'application/json'}).end(JSON.stringify(receipt));
  }catch(error){const fault=error instanceof CollabError?error:new CollabError('UNAVAILABLE','Operation failed; retain pending mutation',503,true);response.writeHead(fault.status,{'content-type':'application/json'}).end(JSON.stringify({code:fault.code,message:fault.message,requestId}));}
 });
 for(const server of [publicServer,internalServer]){server.requestTimeout=5000;server.headersTimeout=5000;}
 let compact:Compactor|undefined,projector:ActivityProjector|undefined,checkpointer:VersionCheckpointer|undefined;
 async function close(){
  closed=true;listening=false;ownership.invalidate();clearTimeout(reconnect);
  // Drain the checked-out connection before returning it to pg-pool: release(true)
  // first would attach its idle error handler while a termination error is still arriving.
  if(listener){const client=listener;listener=undefined;await client.end();client.release(true);}
  await Promise.allSettled([compact?.close(),checkpointer?.close(),projector?.close(),rooms.close(),connecting]);
  await Promise.all([publicServer,internalServer].map(server=>new Promise<void>(resolve=>{server.close(()=>resolve());server.closeAllConnections();})));wss.close();
 }
 try{
  await subscribe();
  compact=new Compactor(store,(_error,ref)=>{if(ref)rooms.invalidateScene(ref.sceneId,'COMPACTION_FAILED');});
  checkpointer=new VersionCheckpointer(store,(_error,ref)=>rooms.invalidateScene(ref.sceneId,'VERSION_CHECKPOINT_FAILED'));
  projector=new ActivityProjector(new PostgresActivityProjectionStore(store.pool),()=>{});
  await listen(publicServer,config.host,config.port);await listen(internalServer,config.internalHost,config.internalPort);
 }catch(error){await close();throw error;}
 return {url:'ws://'+config.host+':'+port(publicServer),httpURL:'http://'+config.host+':'+port(publicServer),internalURL:'http://'+config.internalHost+':'+port(internalServer),close};
}
function port(server:Server){return (server.address() as {port:number}).port;}
function listen(server:Server,host:string,port:number){return new Promise<void>((resolve,reject)=>{server.once('error',reject);server.listen(port,host,resolve);});}
export function configuration(env:NodeJS.ProcessEnv=process.env){
 const secrets=['COLLAB_JWT_SECRET','MUTATION_GRANT_SECRET','AUTHORITY_SERVICE_SECRET','COLLAB_INTERNAL_SECRET'].map(name=>{const value=env[name]??'';if(Buffer.byteLength(value)<32)throw new Error(name+' requires at least 32 bytes');return value;});
 if(new Set(secrets).size!==secrets.length)throw new Error('Collaboration secrets must differ');
 if(!env.DATABASE_URL)throw new Error('DATABASE_URL required');
 const allowedOrigins=(env.COLLAB_ALLOWED_ORIGINS??'').split(',').map(v=>v.trim()).filter(Boolean);if(!allowedOrigins.length)throw new Error('COLLAB_ALLOWED_ORIGINS required');for(const origin of allowedOrigins)if(new URL(origin).origin!==origin||!/^https?:/.test(origin))throw new Error('Exact HTTP(S) origins required');
 const integer=(name:string,fallback:number,min=1,max=65535)=>{const value=Number(env[name]??fallback);if(!Number.isInteger(value)||value<min||value>max)throw new Error('Invalid '+name);return value;};
 if(env.COLLAB_V2_ENABLED!==undefined&&!['true','false'].includes(env.COLLAB_V2_ENABLED))throw new Error('COLLAB_V2_ENABLED must be true or false');
 return {promoteV2:env.COLLAB_V2_ENABLED==='true',databaseURL:env.DATABASE_URL,authority:{roomSecret:secrets[0],grantSecret:secrets[1],serviceSecret:secrets[2],url:env.AUTHORITY_URL??'http://127.0.0.1:8081'},server:{host:env.COLLAB_HOST??'127.0.0.1',port:integer('COLLAB_PORT',8082),internalHost:env.COLLAB_INTERNAL_HOST??'127.0.0.1',internalPort:integer('COLLAB_INTERNAL_PORT',8083),internalSecret:secrets[3],allowedOrigins,updatesPerMinute:integer('COLLAB_UPDATES_PER_MINUTE',120),maxBufferedBytes:integer('COLLAB_MAX_BUFFERED_BYTES',4194304,1048576,67108864)}};
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
 const config=configuration();const pool=new Pool({connectionString:config.databaseURL,max:16,connectionTimeoutMillis:3000,statement_timeout:5000});pool.on('error',()=>{});
 const server=await startServer(new Persistence(pool,new Authority(config.authority),{promoteV2:config.promoteV2,updatesPerMinute:config.server.updatesPerMinute}),config.server);
 const stop=async()=>{await server.close();await pool.end();};process.once('SIGINT',()=>void stop());process.once('SIGTERM',()=>void stop());
}
