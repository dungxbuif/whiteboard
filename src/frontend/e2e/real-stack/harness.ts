import { spawn, execFileSync, type ChildProcess } from 'node:child_process';
import { mkdtempSync, readFileSync, readdirSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { randomBytes } from 'node:crypto';
import { createServer } from 'node:net';
import { fileURLToPath } from 'node:url';
import { Pool } from 'pg';
import { createServer as viteServer, type ViteDevServer } from 'vite';
export const root = fileURLToPath(new URL('../../../', import.meta.url));
export const repo = resolve(root, '..');
export async function until(check: () => boolean | Promise<boolean>, timeout = 20000, message = 'real-stack condition') {
  const end = Date.now() + timeout;
  while (!await check()) { if (Date.now() > end) throw new Error('Timed out: ' + message); await new Promise(r => setTimeout(r, 30)); }
}
async function freePort() { const s = createServer(); await new Promise<void>(r => s.listen(0, '127.0.0.1', r)); const p = (s.address() as {port:number}).port; await new Promise<void>(r => s.close(() => r())); return p; }
const secret = () => randomBytes(32).toString('hex');
export type Account = { token: string; id: string; email: string; workspaceId: string; password: string };
export class RealStack {
  readonly dir = mkdtempSync(join(tmpdir(), 'collab-e1-'));
  readonly schema = 'e1_' + randomBytes(8).toString('hex');
  readonly project = 'collab-' + this.schema.replace('_', '-');
  readonly baseDB = process.env.ACCESS_TEST_DATABASE_URL ?? 'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
  readonly env: NodeJS.ProcessEnv = { PATH: (process.env.PATH ?? '') + ':/opt/homebrew/bin:/usr/local/go/bin', HOME: process.env.HOME, TMPDIR: this.dir };
  pool!: Pool; admin!: Pool; vite?: ViteDevServer; children: ChildProcess[] = []; runtime?: ChildProcess;
  api = ''; baseURL = ''; ws = ''; mcp = ''; s3 = ''; databaseURL = ''; logs = ''; containerStarted = false;
  owner!: Account; editor!: Account; viewer!: Account; outsider!: Account;
  run(program: string, args: string[], cwd = root) { return execFileSync('rtk', ['proxy', program, ...args], { cwd, env: this.env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }); }
  launch(program: string, args: string[], cwd = root, env = this.env) {
    const p = spawn('rtk', ['proxy', program, ...args], { cwd, env, detached: true, stdio: ['ignore', 'pipe', 'pipe'] });
    p.stdout?.on('data', b => this.logs += b); p.stderr?.on('data', b => this.logs += b); this.children.push(p); return p;
  }
  async stop(p?: ChildProcess, signal: NodeJS.Signals = 'SIGTERM') { if (!p?.pid || p.exitCode !== null || p.signalCode !== null) return; try { process.kill(-p.pid, signal); } catch {} await until(() => p.exitCode !== null || p.signalCode !== null, 10000, 'owned process exit'); }
  compose(args: string[]) { return this.run('docker', ['compose', '-p', this.project, '-f', join(repo, 'docker-compose.collab-test.yml'), ...args]); }
  async start() {
    const db = new URL(this.baseDB);
    if (!['127.0.0.1', 'localhost', '[::1]'].includes(db.hostname) || !db.pathname.endsWith('_test') || db.searchParams.has('options')) throw new Error('E1 refuses non-loopback/non-_test database or inherited search path');
    this.admin = new Pool({ connectionString: this.baseDB });
    await this.admin.query('CREATE SCHEMA ' + this.schema);
    db.searchParams.set('options', '-c search_path=' + this.schema + ',public'); this.databaseURL = db.toString();
    this.pool = new Pool({ connectionString: this.databaseURL, max: 4 });
    for (const file of readdirSync(join(root, 'backend/migrations')).filter(f => f.endsWith('.up.sql')).sort()) await this.pool.query(readFileSync(join(root, 'backend/migrations', file), 'utf8'));
    const apiPort = await freePort(), authPort = await freePort(), collabPort = await freePort(), internalPort = await freePort(), mcpPort = await freePort();
    const requested = process.env.BASE_URL ? new URL(process.env.BASE_URL) : undefined;
    if (requested && (!['127.0.0.1','localhost'].includes(requested.hostname) || requested.protocol !== 'http:' || requested.pathname !== '/')) throw new Error('BASE_URL must be a loopback HTTP origin');
    const frontPort = requested ? Number(requested.port) : await freePort();
    this.baseURL = requested?.origin ?? 'http://127.0.0.1:' + frontPort;
    this.api = 'http://127.0.0.1:' + apiPort; this.ws = 'ws://127.0.0.1:' + collabPort; this.mcp = 'http://127.0.0.1:' + mcpPort;
    Object.assign(this.env, { E1_S3_ACCESS: secret().slice(0,20), E1_S3_SECRET: secret(), E1_DB_PASSWORD: secret() });
    this.containerStarted = true; this.compose(['up', '-d', 'objectstore']);
    this.s3 = 'http://' + this.compose(['port', 'objectstore', '9000']).trim();
    await until(async () => { try { return (await fetch(this.s3 + '/')).status === 403; } catch { return false; } }, 30000, 'S3 readiness');
    this.run('curl', ['--silent','--show-error','--fail','--aws-sigv4','aws:amz:us-east-1:s3','--user',this.env.E1_S3_ACCESS + ':' + this.env.E1_S3_SECRET,'-X','PUT',this.s3 + '/e1-assets']);
    this.run('curl', ['--silent','--show-error','--fail','--aws-sigv4','aws:amz:us-east-1:s3','--user',this.env.E1_S3_ACCESS + ':' + this.env.E1_S3_SECRET,'-X','PUT','-H','Content-Type: application/xml','--data-binary',`<CORSConfiguration><CORSRule><AllowedOrigin>${this.baseURL}</AllowedOrigin><AllowedMethod>GET</AllowedMethod><AllowedMethod>PUT</AllowedMethod><AllowedMethod>HEAD</AllowedMethod><AllowedHeader>*</AllowedHeader><ExposeHeader>ETag</ExposeHeader></CORSRule></CORSConfiguration>`,this.s3 + '/e1-assets?cors']);
    Object.assign(this.env, {
      DATABASE_URL: this.databaseURL, JWT_SECRET: secret(), COLLAB_JWT_SECRET: secret(), AUTHORITY_SERVICE_SECRET: secret(), MUTATION_GRANT_SECRET: secret(), COLLAB_INTERNAL_SECRET: secret(),
      HTTP_ADDR: '127.0.0.1:' + apiPort, AUTHORITY_LISTEN_ADDR: '127.0.0.1:' + authPort, AUTHORITY_URL: 'http://127.0.0.1:' + authPort,
      COLLAB_INTERNAL_URL: 'http://127.0.0.1:' + internalPort, COLLAB_PORT: String(collabPort), COLLAB_HOST:'127.0.0.1', COLLAB_INTERNAL_HOST:'127.0.0.1', COLLAB_INTERNAL_PORT:String(internalPort),
      COLLAB_ALLOWED_ORIGINS: this.baseURL, COLLAB_V2_ENABLED:'true', CASBIN_MODEL_PATH:join(root,'backend/configs/casbin_model.conf'),
      S3_ENDPOINT:this.s3,S3_ACCESS_KEY:this.env.E1_S3_ACCESS,S3_SECRET_KEY:this.env.E1_S3_SECRET,S3_BUCKET:'e1-assets',S3_REGION:'us-east-1',FRONTEND_URL:this.baseURL,
      WHITEBOARD_API_URL:this.api,WHITEBOARD_PUBLIC_URL:this.baseURL,MCP_AUDIT_PATH:join(this.dir,'audit.jsonl'),MCP_ALLOWED_ORIGINS:this.baseURL,PORT:String(mcpPort),MCP_BIND_HOST:'127.0.0.1',
    });
    this.run('go',['build','-o',join(this.dir,'server'),'./cmd/server'],join(root,'backend'));
    this.run('go',['build','-o',join(this.dir,'userctl'),'./cmd/userctl'],join(root,'backend'));
    const go = this.launch(join(this.dir,'server'), [], join(root,'backend'));
    await until(async () => { if (go.exitCode !== null) throw new Error(this.safeLogs()); try { return (await fetch(this.api+'/healthz')).ok; } catch { return false; } }, 20000, 'Go server');
    await this.startRuntime();
    this.launch(process.execPath,['mcp/dist/http.js'],root,Object.fromEntries(['PATH','HOME','TMPDIR','WHITEBOARD_API_URL','WHITEBOARD_PUBLIC_URL','MCP_AUDIT_PATH','MCP_ALLOWED_ORIGINS','PORT','MCP_BIND_HOST'].map(k=>[k,this.env[k]])));
    await until(async () => { try { return (await fetch(this.mcp+'/mcp')).status === 401; } catch { return false; } }, 20000, 'MCP HTTP');
    this.vite = await viteServer({ root:join(root,'frontend'), configFile:join(root,'frontend/vite.config.ts'), envFile:false, server:{host:requested?.hostname ?? '127.0.0.1', port:frontPort, strictPort:true, proxy:{'/api':this.api,'/collab':{target:this.ws,ws:true},'/mcp':this.mcp}} });
    await this.vite.listen();
    this.owner = await this.account('owner'); this.outsider = await this.account('outsider');
    this.editor = await this.invite('editor'); this.viewer = await this.invite('viewer');
  }
  async startRuntime() {
    this.env.E1_FAULT_DIR = this.dir;
    this.runtime = this.launch(process.execPath,['--import',join(repo,'tests/collab/fault-runtime.mjs'),'collab/dist/server.js']);
    await until(async () => { if (this.runtime?.exitCode !== null) throw new Error(this.safeLogs()); try { return (await fetch(this.ws.replace('ws:','http:')+'/readyz')).ok; } catch { return false; } },20000,'collab readiness');
  }
  async request(path: string, token = '', workspaceId = '', method = 'GET', data?: unknown) {
    return fetch(this.api + '/api/v1' + path, { method, headers:{ ...(token?{authorization:'Bearer '+token}:{}),...(workspaceId?{'X-Workspace-ID':workspaceId}:{}),'content-type':'application/json'},body:data===undefined?undefined:JSON.stringify(data) });
  }
  async json(path: string, token = '', workspaceId = '', method = 'GET', data?: unknown): Promise<any> {
    const r=await this.request(path,token,workspaceId,method,data); const text=await r.text(); if(!r.ok)throw new Error(method+' '+path+' '+r.status+' '+text); return text?JSON.parse(text):undefined;
  }
  async account(name: string): Promise<Account> {
    const email=name+'@'+this.schema.replace('_','-')+'.example',password=secret(); this.run(join(this.dir,'userctl'),['--email',email,'--password',password,'--name',name]);
    const {token}=await this.json('/auth/login','','','POST',{email,password});const user=await this.json('/users/me',token); const workspaces=await this.json('/workspaces',token);
    return {token,id:user.id,email,password,workspaceId:workspaces[0].id};
  }
  async invite(role: 'editor'|'viewer'): Promise<Account> {
    const email=role+'@'+this.schema.replace('_','-')+'.example',password=secret();
    const invitation=await this.json('/workspaces/'+this.owner.workspaceId+'/invitations',this.owner.token,this.owner.workspaceId,'POST',{email,role});
    const inviteToken=new URL(invitation.inviteUrl).pathname.split('/').pop();
    await this.json('/invitations/register','','','POST',{token:inviteToken,email,password,name:role});
    const {token}=await this.json('/auth/login','','','POST',{email,password});const user=await this.json('/users/me',token);
    return {token,id:user.id,email,password,workspaceId:this.owner.workspaceId};
  }
  scene(title='E1 '+secret().slice(0,8)) {return this.json('/scenes',this.owner.token,this.owner.workspaceId,'POST',{title,content:{elements:[],appState:{},files:{}}});}
  async room(sceneId:string,account=this.owner) { return this.json('/scenes/'+sceneId+'/room-token',account.token,this.owner.workspaceId,'POST',{}); }
  safeLogs() { let s=this.logs; for(const [k,v] of Object.entries(this.env))if(v&&/SECRET|PASSWORD|KEY|DATABASE_URL/.test(k))s=s.replaceAll(v,'[redacted]');return s.slice(-10000); }
  async close() {
    await this.vite?.close();
    for(const p of this.children.slice().reverse())await this.stop(p).catch(()=>this.stop(p,'SIGKILL'));
    await this.pool?.end();
    if(this.admin){if(!/^e1_[a-f0-9]{16}$/.test(this.schema))throw new Error('Invalid cleanup schema');await this.admin.query('DROP SCHEMA '+this.schema+' CASCADE');await this.admin.end();}
    if(this.containerStarted)this.compose(['down','--volumes','--remove-orphans']);
    rmSync(this.dir,{recursive:true,force:true});
  }
}
