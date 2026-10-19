import {afterEach,describe,expect,it} from 'vitest';
import {mkdtempSync,readFileSync,rmSync,statSync,readdirSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {AuditJournal,runAuditedTool} from './audit.js';
import {GoClient} from './go-client.js';
import {AccessError} from './auth.js';
const dirs:string[]=[];
function path(){const dir=mkdtempSync(join(tmpdir(),'mcp-audit-'));dirs.push(dir);return join(dir,'events.jsonl')}
afterEach(()=>{for(const dir of dirs.splice(0))rmSync(dir,{recursive:true,force:true})});
function events(p:string){return readFileSync(p,'utf8').trim().split('\n').filter(Boolean).map(v=>JSON.parse(v))}
const client=()=>new GoClient('http://127.0.0.1:9','must-never-appear');
describe('durable adapter audit',()=>{
 it('reopens success and denied calls with safe metadata only',async()=>{const p=path();const journal=new AuditJournal(p);await runAuditedTool(client(),journal,'create_scene',{title:'private-title',workspaceId:'11111111-1111-4111-8111-111111111111'},async()=>({ok:true}));await expect(runAuditedTool(client(),journal,'get_scene',{content:'private-document'},async()=>{throw new AccessError(403,'ACCESS_DENIED')})).rejects.toThrow();new AuditJournal(p);const rows=events(p);expect(rows).toHaveLength(4);expect(rows.map(r=>r.outcome)).toEqual(['started','success','started','denied']);expect(rows.every(r=>r.provenance==='adapter_observed'&&r.correlationId&&r.latencyMs>=0)).toBe(true);expect(rows[0].scope).toBe('scene:create');expect(statSync(p).mode&0o777).toBe(0o600);const raw=readFileSync(p,'utf8');for(const secret of ['must-never-appear','private-title','private-document'])expect(raw).not.toContain(secret)});
 it('records validation and unreachable upstream failures',async()=>{const p=path();const journal=new AuditJournal(p);await expect(runAuditedTool(client(),journal,'get_scene',{},async()=>{throw new AccessError(400,'INVALID_INPUT')})).rejects.toThrow();await expect(runAuditedTool(client(),journal,'list_workspaces',{},async()=>{throw new Error('private-upstream-detail')})).rejects.toThrow();expect(events(p).filter(r=>r.outcome!=='started').map(r=>r.failureClass)).toEqual(['INVALID_INPUT','UPSTREAM_UNAVAILABLE']);expect(readFileSync(p,'utf8')).not.toContain('private-upstream-detail')});
 it('fails closed before effects and marks completion failure uncertain',async()=>{let calls=0;const p=path();const journal=new AuditJournal(p);journal.record=()=>{throw new Error('disk full')};await expect(runAuditedTool(client(),journal,'create_scene',{},async()=>{calls++;return {ok:true}})).rejects.toMatchObject({code:'AUDIT_UNAVAILABLE'});expect(calls).toBe(0);let writes=0;journal.record=()=>{if(++writes===2)throw new Error('disk full')};await expect(runAuditedTool(client(),journal,'create_scene',{},async()=>{calls++;return {ok:true}})).rejects.toMatchObject({code:'AUDIT_COMPLETION_UNCERTAIN'});expect(calls).toBe(1)});
 it('bounds retained journals with restrictive permissions',async()=>{const p=path();const journal=new AuditJournal(p,{maxBytes:1024,archives:2});for(let i=0;i<20;i++)await runAuditedTool(client(),journal,'list_workspaces',{},async()=>({ok:true}));const files=readdirSync(join(p,'..'));expect(files).toHaveLength(3);for(const file of files){expect(statSync(join(p,'..',file)).size).toBeLessThanOrEqual(1024);expect(statSync(join(p,'..',file)).mode&0o777).toBe(0o600)}});
});
