// Disposable rehearsal helpers. These are intentionally not a production reset command.
import {randomUUID} from 'node:crypto';
import type {Pool,PoolClient} from 'pg';
import * as Y from 'yjs';
import type {SceneRef} from '@collab/contracts';
import {hash} from '../src/auth.js';
import {materializeScene} from '../src/document.js';
import {OWNER_KEY,HANDOFF_KEY} from '../src/ownership.js';
import {readArchive,sealCheckpoint,writeArchive,verifyArchive,type RecoveryCheckpoint} from '../src/recovery.js';

export function guardDatabase(url:string):void {
 const parsed=new URL(url);
 if(!['postgres:','postgresql:'].includes(parsed.protocol)||parsed.hostname!=='127.0.0.1'||parsed.port!=='55439'||parsed.pathname!=='/devtool_a1_test'||[...parsed.searchParams].some(([key,value])=>key!=='sslmode'||value!=='disable'))throw new Error('Disposable recovery requires exactly 127.0.0.1:55439/devtool_a1_test without connection overrides');
}
async function maintenance<T>(pool:Pool,work:(client:PoolClient)=>Promise<T>):Promise<T>{
 guardDatabase(pool.options.connectionString??'');const client=await pool.connect();
 try{
  const scope=(await client.query('SELECT current_database() AS db,current_schema() AS schema')).rows[0];
  if(scope.db!=='devtool_a1_test'||!/^b2_[0-9a-f]{32}$/.test(scope.schema))throw new Error('Disposable randomized fixture schema required');
  await client.query("SET statement_timeout='5s'");
  if(!(await client.query(`SELECT pg_try_advisory_lock(${OWNER_KEY}) AS acquired`)).rows[0].acquired)throw new Error('Recovery refused: active writer must stop and drain first');
  await client.query(`SELECT pg_advisory_lock(${HANDOFF_KEY})`);
  await client.query('BEGIN');const result=await work(client);await client.query('COMMIT');return result;
 }catch(error){try{await client.query('ROLLBACK');}catch{}throw error;}
 finally{client.release(true);}
}
async function lockScene(client:PoolClient,ref:SceneRef){
 if(!(await client.query('SELECT id FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId])).rowCount)throw new Error('Active workspace required');
 const row=(await client.query('SELECT * FROM scenes WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId,ref.sceneId])).rows[0];
 if(!row)throw new Error('Active source scene required');return row;
}
async function exported(client:PoolClient,ref:SceneRef):Promise<Record<string,unknown>>{
 const row=(await client.query('SELECT * FROM collab_documents WHERE workspace_id=$1 AND scene_id=$2 FOR UPDATE',[ref.workspaceId,ref.sceneId])).rows[0];if(!row)throw new Error('Existing durable document required');
 const updates=(await client.query('SELECT sequence::text,update_bytes FROM collab_updates WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows.map(r=>({sequence:r.sequence,bytes:r.update_bytes.toString('base64'),hash:hash(r.update_bytes)}));
 const doc=new Y.Doc({gc:false});
 try{
  Y.applyUpdate(doc,row.snapshot);for(const update of updates)Y.applyUpdate(doc,Buffer.from(update.bytes,'base64'));
  return {format:1,...ref,documentId:row.document_id,sequence:row.sequence,promotionCheckpoint:{content:row.legacy_content,contentVersion:row.legacy_content_version},snapshotSequence:row.snapshot_sequence,snapshot:row.snapshot.toString('base64'),snapshotHash:hash(row.snapshot),updates,receipts:(await client.query('SELECT * FROM collab_receipts WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,audit:(await client.query('SELECT * FROM collab_mutation_audit WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,content:materializeScene(doc)};
 }finally{doc.destroy();}
}
export async function checkpointFixture(pool:Pool,ref:SceneRef,path:string){
 return maintenance(pool,async client=>{await lockScene(client,ref);return writeArchive(path,await exported(client,ref));});
}
export async function recoverFixture(pool:Pool,path:string):Promise<SceneRef>{
 const archive=await readArchive(path),content=verifyArchive(archive),c=archive.checkpoint,ref={workspaceId:c.workspaceId,sceneId:c.sceneId};
 return maintenance(pool,async client=>{
  const scene=await lockScene(client,ref);
  const current=sealCheckpoint(await exported(client,ref));if(current.sha256!==archive.sha256)throw new Error('Source changed after checkpoint; keep maintenance and take a new archive');
  const unsupported=await client.query('SELECT EXISTS(SELECT 1 FROM scene_assets WHERE scene_id=$1) AS assets,EXISTS(SELECT 1 FROM scene_member_overrides WHERE scene_id=$1) AS overrides',[ref.sceneId]);
  if(unsupported.rows[0].assets||unsupported.rows[0].overrides||Object.keys(content.files).length||content.elements.some(element=>element.type==='image'))throw new Error('Unsupported recovery: assets or scene overrides require explicit operator mapping; source retained');
  const sceneId=randomUUID();
  await client.query('UPDATE scenes SET deleted_at=clock_timestamp() WHERE id=$1',[ref.sceneId]);
  await client.query("INSERT INTO scenes(id,workspace_id,folder_id,title,content,content_version,is_starred,created_by,updated_by,writer_mode) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'v1')",[sceneId,ref.workspaceId,scene.folder_id,scene.title,JSON.stringify(content),(BigInt(c.promotionCheckpoint.contentVersion)+BigInt(c.sequence)).toString(),scene.is_starred,scene.created_by,scene.updated_by]);
  return {workspaceId:ref.workspaceId,sceneId};
 });
}
export async function restoreFixture(pool:Pool,path:string,ownerId:string):Promise<void>{
 const archive=await readArchive(path),c:RecoveryCheckpoint=archive.checkpoint;
 await maintenance(pool,async client=>{
  // Fresh fixture IDs only. Any identity conflict aborts the complete transaction.
  await client.query('INSERT INTO workspaces(id,owner_id,name) VALUES($1,$2,$3)',[c.workspaceId,ownerId,'Isolated recovery '+c.documentId]);
  await client.query("INSERT INTO scenes(id,workspace_id,title,content,content_version,writer_mode) VALUES($1,$2,$3,$4,$5,'v2')",[c.sceneId,c.workspaceId,'Original checkpoint',JSON.stringify(c.content),(BigInt(c.promotionCheckpoint.contentVersion)+BigInt(c.sequence)).toString()]);
  await client.query('INSERT INTO collab_documents(scene_id,workspace_id,document_id,sequence,snapshot_sequence,snapshot,legacy_content,legacy_content_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8)',[c.sceneId,c.workspaceId,c.documentId,c.sequence,c.snapshotSequence,Buffer.from(c.snapshot,'base64'),JSON.stringify(c.promotionCheckpoint.content),c.promotionCheckpoint.contentVersion]);
  for(const update of c.updates)await client.query('INSERT INTO collab_updates(scene_id,sequence,update_bytes) VALUES($1,$2,$3)',[c.sceneId,update.sequence,Buffer.from(update.bytes,'base64')]);
  for(const receipt of c.receipts)await client.query('INSERT INTO collab_receipts(scene_id,request_id,principal_type,principal_id,payload_hash,action,sequence,state_vector,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)',[receipt.scene_id,receipt.request_id,receipt.principal_type,receipt.principal_id,receipt.payload_hash,receipt.action,receipt.sequence,receipt.state_vector,receipt.created_at]);
  for(const audit of c.audit)await client.query('INSERT INTO collab_mutation_audit(scene_id,request_id,workspace_id,principal_type,principal_id,sequence,payload_hash,action,acl_version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',[audit.scene_id,audit.request_id,audit.workspace_id,audit.principal_type,audit.principal_id,audit.sequence,audit.payload_hash,audit.action,audit.acl_version,audit.created_at]);
  if(sealCheckpoint(await exported(client,{workspaceId:c.workspaceId,sceneId:c.sceneId})).sha256!==archive.sha256)throw new Error('Restored history mismatch');
 });
}
