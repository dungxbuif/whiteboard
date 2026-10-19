import { createServer, type Server as HTTPServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { randomUUID } from 'node:crypto';
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { CallToolRequestSchema, ListToolsRequestSchema, ListResourcesRequestSchema, ReadResourceRequestSchema, isInitializeRequest, type Tool } from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';
import { GoClient } from './go-client.js';
import { AccessError, bearer, checkOrigin } from './auth.js';
import { AuditJournal, runAuditedTool } from './audit.js';
import {sceneTools,callSceneTool} from './scene-tools.js';
import {diagramTools,callDiagramTool,diagramResources,readDiagramResource} from './diagram-tools.js';
import {ExportService} from './export-tools.js';
import {exportTools,callExportTool,exportAuthority,renderAccessError} from './export-api.js';
import { auditToolEnvelopes } from './transport-audit.js';

const page = { cursor: z.uuid().optional(), limit: z.number().int().min(1).max(50).default(50) };
const workspace = z.object({ workspaceId: z.uuid(), ...page }).strict();
const scene = z.object({ workspaceId: z.uuid(), sceneId: z.uuid() }).strict();
const schemas = {
  list_workspaces: z.object(page).strict(),
  list_scenes: workspace,
  get_scene: scene,
  create_scene: z.object({ workspaceId: z.uuid(), title: z.string().trim().min(1).max(255) }).strict(),
};
const descriptions = {
  list_workspaces: 'List workspaces allowed by this key. Requires workspace:read.',
  list_scenes: 'List scene metadata in an allowed workspace. Requires scene:read.',
  get_scene: 'Read a scene projection up to 500 elements and 256 KiB. Requires scene:read.',
  create_scene: 'Create an empty editable scene. Requires scene:create and a workspace-wide key.',
};
const tools: Tool[] = Object.entries(schemas).map(([name, schema]) => ({
  name,
  description: descriptions[name as keyof typeof descriptions],
  inputSchema: z.toJSONSchema(schema) as Tool['inputSchema'],
  annotations: {
    readOnlyHint: name !== 'create_scene', destructiveHint: false,
    idempotentHint: name !== 'create_scene', openWorldHint: false,
  },
}));

export function createMcpServer(client: GoClient, audit: AuditJournal, exports?:ExportService, publicURL?:string): Server {
  const server = new Server({ name: 'whiteboard', version: '0.1.0' }, { capabilities: { tools: {}, resources: {} } });
  server.setRequestHandler(ListToolsRequestSchema, async () => {
    await client.authenticate();
    return { tools: [...tools,...sceneTools,...diagramTools,...exportTools] };
  });
  server.setRequestHandler(ListResourcesRequestSchema, async()=>{await client.authenticate();return {resources:diagramResources()};});
  server.setRequestHandler(ReadResourceRequestSchema, async request=>{await client.authenticate();return {contents:[{uri:request.params.uri,mimeType:'application/json',text:JSON.stringify(readDiagramResource(request.params.uri))}]};});
  server.setRequestHandler(CallToolRequestSchema, async request => {
    const name = request.params.name;
    const args = request.params.arguments || {};
    try {
      return await runAuditedTool(client, audit, name, args, async api => {
        await api.authenticate();
        let result: Record<string, unknown>;
        switch (name) {
          case 'list_workspaces': {
            const a = schemas.list_workspaces.parse(args);
            const query = new URLSearchParams({ limit: String(a.limit), ...(a.cursor ? { cursor: a.cursor } : {}) });
            result = await api.request(`/api/v1/agent/workspaces?${query}`);
            break;
          }
          case 'list_scenes': {
            const a = schemas.list_scenes.parse(args);
            const query = new URLSearchParams({ limit: String(a.limit), ...(a.cursor ? { cursor: a.cursor } : {}) });
            result = await api.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes?${query}`);
            break;
          }
          case 'get_scene': {
            const a = schemas.get_scene.parse(args);
            result = await api.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes/${a.sceneId}`);
            break;
          }
          case 'create_scene': {
            const a = schemas.create_scene.parse(args);
            result = await api.request(`/api/v1/agent/workspaces/${a.workspaceId}/scenes`, 'POST', { title: a.title });
            break;
          }
          default: {
            const extra=await callSceneTool(name,args,api)??await callDiagramTool(name,args,api)??await callExportTool(name,args,api,exports,publicURL);
            if(!extra)throw new AccessError(400,'UNKNOWN_TOOL');
            result=extra;
          }
        }
        if (name === 'get_scene') {
          const projection = result.content as { elements?: unknown[] } | undefined;
          if (projection?.elements && projection.elements.length > 500) {
            throw new AccessError(413, 'PROJECTION_TOO_LARGE');
          }
        }
        // A second structured copy would double the projection response size.
        const response = { content: [{ type: 'text' as const, text: JSON.stringify(result) }] };
        if (Buffer.byteLength(JSON.stringify(response)) > 256 * 1024 - 128) {
          throw new AccessError(413, 'PROJECTION_TOO_LARGE');
        }
        return response;
      });
    } catch (error) {
      const code = error instanceof AccessError ? error.code : 'UPSTREAM_UNAVAILABLE';
      const message = code === 'AUDIT_COMPLETION_UNCERTAIN'
        ? `${code}: retain the exact mutation request ID and arguments; inspect authoritative state before retrying creation.`
        : code === 'UNSUPPORTED_NATIVE_DIAGRAM' ? 'UNSUPPORTED_NATIVE_DIAGRAM: use get_diagram_template for flowchart, sequence, erd or class and generate_diagram with its typed native data.' : code === 'CONFLICT' ? 'CONFLICT: read the current scene and deliberately submit a new requestId with its sequence; do not blindly retry stale patches.' : code;
      return {
        isError: true,
        content: [{ type: 'text', text: message }],
        ...(error instanceof AccessError && error.correlationId ? { _meta: { agentCallId: error.correlationId } } : {}),
      };
    }
  });
  return server;
}

type Session = {
  principal: string; client: GoClient; server: Server;
  transport: StreamableHTTPServerTransport; expires: number;
};
async function body(req: IncomingMessage): Promise<unknown> {
  const parts: Buffer[] = [];
  let bytes = 0;
  for await (const chunk of req) {
    bytes += chunk.length;
    if (bytes > 1024 * 1024) throw new AccessError(413, 'BODY_TOO_LARGE');
    parts.push(Buffer.from(chunk));
  }
  try { return JSON.parse(Buffer.concat(parts).toString('utf8')); }
  catch { throw new AccessError(400, 'INVALID_JSON'); }
}
function fail(res: ServerResponse, status: number, code: string, correlationId?: string): void {
  if (res.headersSent) { res.end(); return; }
  res.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' });
  res.end(JSON.stringify({ error: code, ...(correlationId ? { agentCallId: correlationId } : {}) }));
}
function toolCall(parsed: unknown): { name: string; args: Record<string, unknown> } | undefined {
  if (!parsed || typeof parsed !== 'object') return;
  const message = parsed as { method?: string; params?: { name?: unknown; arguments?: unknown } };
  if (message.method !== 'tools/call' || typeof message.params?.name !== 'string') return;
  const args = message.params.arguments;
  return { name: message.params.name, args: args && typeof args === 'object' && !Array.isArray(args) ? args as Record<string, unknown> : {} };
}

export function createMcpHTTP(options: { apiURL: string; allowedOrigins: string[]; audit: AuditJournal; publicURL?:string }): HTTPServer {
  const sessions = new Map<string, Session>();
  const exports = new ExportService();
  const publicURL=()=>options.publicURL??`http://127.0.0.1:${(http.address() as {port:number}).port}`;
  const http = createServer(async (req, res) => {
    let parsed: unknown;
    let api: GoClient | undefined;
    let routed = false;
    try {
      const artifact=/^\/mcp\/artifacts\/([0-9a-f-]{36})$/.exec(req.url??'');
      if (req.url !== '/mcp' && req.url!=='/mcp/exports' && !artifact) { fail(res, 404, 'NOT_FOUND'); return; }
      checkOrigin(req, options.allowedOrigins);
      const key = bearer(req);
      if(req.url==='/mcp/exports'||artifact){
        api=new GoClient(options.apiURL,key,undefined,publicURL());await api.authenticate();
        if(req.url==='/mcp/exports'&&req.method==='POST'){
          const input=await body(req) as Record<string,unknown>;
          const result=await runAuditedTool(api,options.audit,'export_scene',input,async client=> (await callExportTool('export_scene',input,client,exports,publicURL()))!);
          res.writeHead(200,{'Content-Type':'application/json','Cache-Control':'no-store'}).end(JSON.stringify(result));return;
        }
        if(artifact&&req.method==='GET'){
          const result=await runAuditedTool(api,options.audit,'export_scene',{},async client=>{
            try{return await exports.download(exportAuthority(client),artifact[1]);}catch(e){renderAccessError(e);}
          });
          res.writeHead(200,{'Content-Type':result.mimeType,'Content-Length':result.bytes.length,'Content-Disposition':`attachment; filename="scene.${result.mimeType==='image/png'?'png':result.mimeType==='image/svg+xml'?'svg':'excalidraw'}"`,'Cache-Control':'no-store','X-Content-Type-Options':'nosniff','Content-Security-Policy':"default-src 'none'; sandbox"}).end(result.bytes);return;
        }
        fail(res,405,'METHOD_NOT_ALLOWED');return;
      }
      if (req.method === 'POST') parsed = await body(req);
      const id = req.headers['mcp-session-id'];
      if (Array.isArray(id)) throw new AccessError(400, 'INVALID_SESSION');
      const existing = id ? sessions.get(id) : undefined;
      // Reuse prior verified identity only when this request proves possession
      // of the same credential. A session ID alone cannot attribute a denial.
      api = existing?.client.matchesCredential(key) ? existing.client : new GoClient(options.apiURL, key,undefined,publicURL());
      const identity = await api.authenticate();
      for (const [sessionId, session] of sessions) {
        if (session.expires <= Date.now()) {
          sessions.delete(sessionId);
          await session.server.close();
        }
      }
      if (id) {
        const session = sessions.get(id);
        if (!session) throw new AccessError(404, 'SESSION_NOT_FOUND');
        if (session.principal !== identity.principalId) throw new AccessError(403, 'SESSION_PRINCIPAL_MISMATCH');
        routed = true;
        await session.transport.handleRequest(req, res, parsed);
        return;
      }
      if (req.method !== 'POST' || !isInitializeRequest(parsed)) throw new AccessError(400, 'INITIALIZE_REQUIRED');
      if (sessions.size >= 1000) throw new AccessError(503, 'SESSION_LIMIT');
      const client = api;
      const server = createMcpServer(client, options.audit,exports,publicURL());
      const transport = new StreamableHTTPServerTransport({
        sessionIdGenerator: randomUUID,
        enableJsonResponse: true,
        onsessioninitialized: sessionId => {
          sessions.set(sessionId, { principal: identity.principalId, client, server, transport, expires: Date.now() + 10 * 60 * 1000 });
        },
      });
      transport.onclose = () => { if (transport.sessionId) sessions.delete(transport.sessionId); };
      await server.connect(auditToolEnvelopes(transport, client, options.audit));
      await transport.handleRequest(req, res, parsed);
      if (!transport.sessionId) await server.close();
    } catch (error) {
      let failure = error;
      const call = !routed && toolCall(parsed);
      if (call && api) {
        try {
          await runAuditedTool(api, options.audit, call.name, call.args, async () => { throw error; });
        } catch (auditedFailure) { failure = auditedFailure; }
      }
      fail(res, failure instanceof AccessError ? failure.status : 503,
        failure instanceof AccessError ? failure.code : 'UPSTREAM_UNAVAILABLE',
        failure instanceof AccessError ? failure.correlationId : undefined);
    }
  });
  http.requestTimeout = 15000;
  http.headersTimeout = 10000;
  http.on('close', () => {
    for (const session of sessions.values()) void session.server.close();
    sessions.clear();
  });
  return http;
}
