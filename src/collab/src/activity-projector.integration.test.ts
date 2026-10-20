import {afterEach,expect,it} from 'vitest';
import {readFile} from 'node:fs/promises';
import type {PoolClient} from 'pg';
import {fixture} from '../tests/fixture.js';
import {PostgresActivityProjectionStore,projectMutationAuditBatch} from './activity-projector.js';

const cleanups:Array<()=>Promise<void>>=[];
afterEach(async()=>{for(const cleanup of cleanups.splice(0).reverse())await cleanup();});

async function audit(client:Pick<PoolClient,'query'>,ref:{workspaceId:string;sceneId:string},principalId:string,requestId:string,sequence:number,createdAt:string){
 await client.query(`INSERT INTO collab_receipts(scene_id,request_id,principal_type,principal_id,payload_hash,action,sequence,state_vector,created_at) VALUES($1,$2,'user',$3,$4,'scene:update',$5,'',$6)`,[ref.sceneId,requestId,principalId,'a'.repeat(64),sequence,createdAt]);
 await client.query(`INSERT INTO collab_mutation_audit(scene_id,request_id,workspace_id,principal_type,principal_id,sequence,payload_hash,action,acl_version,created_at) VALUES($1,$2,$3,'user',$4,$5,$6,'scene:update',1,$7)`,[ref.sceneId,requestId,ref.workspaceId,principalId,sequence,'a'.repeat(64),createdAt]);
}

it('projects an audit row that commits behind the durable cursor',async()=>{
 const f=await fixture();cleanups.push(()=>f.close());
 await f.pool.query(await readFile(new URL('../../backend/migrations/000015_workspace_events.up.sql',import.meta.url),'utf8'));
 await f.pool.query(`INSERT INTO collab_documents(scene_id,workspace_id,snapshot,legacy_content,legacy_content_version) VALUES($1,$2,$3,'{}',1)`,[f.ref.sceneId,f.ref.workspaceId,Buffer.alloc(0)]);
 const late=await f.pool.connect();
 try{
  await late.query('BEGIN');
  await audit(late,f.ref,f.principal.principalId,'late',1,'2026-10-20T00:00:00.000Z');
  await audit(f.pool as unknown as PoolClient,f.ref,f.principal.principalId,'visible',2,'2026-10-20T00:00:01.000Z');
  const store=new PostgresActivityProjectionStore(f.pool);
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(1);
  await late.query('COMMIT');
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(1);
  await expect(projectMutationAuditBatch(store,100)).resolves.toBe(0);
  expect((await f.pool.query(`SELECT count(*)::int AS count FROM workspace_events WHERE source='collab_mutation_audit'`)).rows[0].count).toBe(2);
 }finally{await late.query('ROLLBACK').catch(()=>{});late.release();}
});
