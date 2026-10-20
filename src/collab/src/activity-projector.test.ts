import {describe,expect,it} from 'vitest';
import {projectMutationAuditBatch,type ActivityProjectionStore,type MutationAuditRow} from './activity-projector.js';

class MemoryProjectionStore implements ActivityProjectionStore {
 cursor={occurredAt:'1970-01-01T00:00:00.000Z',key:''};
 rows:MutationAuditRow[]=[];
 events=new Map<string,MutationAuditRow>();
 failBeforeCommit=false;
 async transaction<T>(work:(transaction:ActivityProjectionStore)=>Promise<T>):Promise<T>{
  const staged=new MemoryProjectionStore();staged.cursor={...this.cursor};staged.rows=this.rows;staged.events=new Map(this.events);staged.failBeforeCommit=this.failBeforeCommit;
  const result=await work(staged);if(this.failBeforeCommit){this.failBeforeCommit=false;throw new Error('commit lost');}
  this.cursor=staged.cursor;this.events=staged.events;return result;
 }
 async readCursor(){return this.cursor;}
 async readMutationAudit(_cursor:{occurredAt:string;key:string},limit:number){return this.rows.filter(row=>!this.events.has(`${row.sceneId}/${row.requestId}`)).sort((a,b)=>a.createdAt.localeCompare(b.createdAt)||`${a.sceneId}/${a.requestId}`.localeCompare(`${b.sceneId}/${b.requestId}`)).slice(0,limit);}
 async appendProjected(row:MutationAuditRow){this.events.set(`${row.sceneId}/${row.requestId}`,row);}
 async writeCursor(cursor:{occurredAt:string;key:string}){this.cursor=cursor;}
}

describe('projectMutationAuditBatch',()=>{
 it('projectionCrashRetryDeduplicates',async()=>{
  const store=new MemoryProjectionStore();store.rows=[{workspaceId:crypto.randomUUID(),sceneId:crypto.randomUUID(),requestId:crypto.randomUUID(),principalType:'user',principalId:crypto.randomUUID(),action:'scene:update',sequence:'1',aclVersion:'2',createdAt:'2026-10-20T00:00:00.000Z'}];
  store.failBeforeCommit=true;await expect(projectMutationAuditBatch(store,100)).rejects.toThrow('commit lost');
  expect(store.events.size).toBe(0);expect(store.cursor.key).toBe('');
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(1);
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(0);
  expect(store.events.size).toBe(1);
 });
 it('rejects unbounded batches',async()=>{await expect(projectMutationAuditBatch(new MemoryProjectionStore(),101)).rejects.toThrow('limit');});
 it('projects a late row even when the durable cursor is stale',async()=>{
  const store=new MemoryProjectionStore();store.cursor={occurredAt:'2099-01-01T00:00:00.000Z',key:'zzzz'};
  const row={workspaceId:crypto.randomUUID(),sceneId:crypto.randomUUID(),requestId:crypto.randomUUID(),principalType:'user' as const,principalId:crypto.randomUUID(),action:'scene:update',sequence:'1',aclVersion:'2',createdAt:'2026-10-20T00:00:00.000Z'};
  store.rows=[row];
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(1);
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(0);
  expect(store.events.get(`${row.sceneId}/${row.requestId}`)).toEqual(row);
 });
});
