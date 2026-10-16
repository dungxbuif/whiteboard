import {createServer,type Server as HTTPServer,type IncomingMessage,type ServerResponse} from 'node:http';
import {randomUUID} from 'node:crypto';
import {Server} from '@modelcontextprotocol/sdk/server/index.js';
import {StreamableHTTPServerTransport} from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import {CallToolRequestSchema,ListToolsRequestSchema,isInitializeRequest,type Tool} from '@modelcontextprotocol/sdk/types.js';
import {z} from 'zod';
import {GoClient} from './go-client.js';
import {AccessError,bearer,checkOrigin} from './auth.js';
const page={cursor:z.uuid().optional(),limit:z.number().int().min(1).max(50).default(50)};
const workspace=z.object({workspaceId:z.uuid(),...page}).strict();
const scene=z.object({workspaceId:z.uuid(),sceneId:z.uuid()}).strict();
const schemas={list_workspaces:z.object(page).strict(),list_scenes:workspace,get_scene:scene,create_scene:z.object({workspaceId:z.uuid(),title:z.string().trim().min(1).max(255)}).strict()};
const descriptions={list_workspaces:'List workspaces allowed by this key. Requires workspace:read.',list_scenes:'List scene metadata in an allowed workspace. Requires scene:read.',get_scene:'Read a scene projection up to 500 elements and 256 KiB. Requires scene:read.',create_scene:'Create an empty editable scene. Requires scene:create and a workspace-wide key.'};
const tools:Tool[]=Object.entries(schemas).map(([name,schema])=>({name,description:descriptions[name as keyof typeof descriptions],inputSchema:z.toJSONSchema(schema) as Tool['inputSchema'],annotations:{readOnlyHint:name!=='create_scene',destructiveHint:false,idempotentHint:name!=='create_scene',openWorldHint:false}}));
export function createMcpServer(client:GoClient):Server{
 const server=new Server({name:'whiteboard',version:'0.1.0'},{capabilities:{tools:{}}});
 server.setRequestHandler(ListToolsRequestSchema,async()=>{await client.authenticate();return {tools}});
 server.setRequestHandler(CallToolRequestSchema,async request=>{
  try{
   await client.authenticate();const name=request.params.name;const args=request.params.arguments||{};let result:Record<string,unknown>;
   switch(name){
    case 'list_workspaces':{const a=schemas.list_workspaces.parse(args);const query=new URLSearchParams({limit:String(a.limit),...(a.cursor?{cursor:a.cursor}:{})});result=await client.request(`/api/v1/agent/workspaces?${query}`);break}
    case 'list_scenes':{const a=schemas.list_scenes.parse(args);const query=new URLSearchParams({limit:String(a.limit),...(a.cursor?{cursor:a.cursor}:{})});result=await client.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes?${query}`);break}
    case 'get_scene':{const a=schemas.get_scene.parse(args);result=await client.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes/${a.sceneId}`);break}
    case 'create_scene':{const a=schemas.create_scene.parse(args);result=await client.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes`,'POST',{title:a.title});break}
    default:throw new AccessError(400,'UNKNOWN_TOOL');
   }
   if(name==='get_scene'){
    const projection=result.content as {elements?:unknown[]}|undefined;
    if(projection?.elements && projection.elements.length>500)throw new AccessError(413,'PROJECTION_TOO_LARGE');
   }
   // Keep one representation: duplicating text + structuredContent doubles large projections.
   const response={content:[{type:'text' as const,text:JSON.stringify(result)}]};
   if(Buffer.byteLength(JSON.stringify(response))>256*1024-128)throw new AccessError(413,'PROJECTION_TOO_LARGE');
   return response;
  }catch(e){const code=e instanceof AccessError?e.code:e instanceof z.ZodError?'INVALID_INPUT':'UPSTREAM_UNAVAILABLE';return {isError:true,content:[{type:'text',text:code}]}}
 });return server;
}
type Session={principal:string;server:Server;transport:StreamableHTTPServerTransport;expires:number};
async function body(req:IncomingMessage):Promise<unknown>{const parts:Buffer[]=[];let bytes=0;for await(const chunk of req){bytes+=chunk.length;if(bytes>1024*1024)throw new AccessError(413,'BODY_TOO_LARGE');parts.push(Buffer.from(chunk))}try{return JSON.parse(Buffer.concat(parts).toString('utf8'))}catch{throw new AccessError(400,'INVALID_JSON')}}
function fail(res:ServerResponse,status:number,code:string){if(res.headersSent){res.end();return}res.writeHead(status,{'Content-Type':'application/json','Cache-Control':'no-store'});res.end(JSON.stringify({error:code}))}
export function createMcpHTTP(options:{apiURL:string;allowedOrigins:string[]}):HTTPServer{
 const sessions=new Map<string,Session>();
 const http=createServer(async(req,res)=>{
  try{
   if(req.url!=='/mcp'){fail(res,404,'NOT_FOUND');return}
   checkOrigin(req,options.allowedOrigins);const key=bearer(req);const api=new GoClient(options.apiURL,key);const identity=await api.authenticate();
   for(const [id,s]of sessions){if(s.expires<=Date.now()){sessions.delete(id);await s.server.close()}}
   const id=req.headers['mcp-session-id'];if(Array.isArray(id))throw new AccessError(400,'INVALID_SESSION');
   if(id){const s=sessions.get(id);if(!s)throw new AccessError(404,'SESSION_NOT_FOUND');if(s.principal!==identity.principalId)throw new AccessError(403,'SESSION_PRINCIPAL_MISMATCH');await s.transport.handleRequest(req,res);return}
   if(req.method!=='POST')throw new AccessError(400,'INITIALIZE_REQUIRED');const parsed=await body(req);if(!isInitializeRequest(parsed))throw new AccessError(400,'INITIALIZE_REQUIRED');if(sessions.size>=1000)throw new AccessError(503,'SESSION_LIMIT');
   const server=createMcpServer(api);const transport=new StreamableHTTPServerTransport({sessionIdGenerator:randomUUID,enableJsonResponse:true,onsessioninitialized:sessionId=>{sessions.set(sessionId,{principal:identity.principalId,server,transport,expires:Date.now()+10*60*1000})}});
   transport.onclose=()=>{if(transport.sessionId)sessions.delete(transport.sessionId)};await server.connect(transport);await transport.handleRequest(req,res,parsed);if(!transport.sessionId)await server.close();
  }catch(e){fail(res,e instanceof AccessError?e.status:503,e instanceof AccessError?e.code:'UPSTREAM_UNAVAILABLE')}
 });
 http.requestTimeout=15000;http.headersTimeout=10000;
 http.on('close',()=>{for(const s of sessions.values())void s.server.close();sessions.clear()});return http;
}
