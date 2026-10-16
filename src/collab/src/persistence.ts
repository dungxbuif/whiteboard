import {randomUUID} from 'node:crypto';
import {EventEmitter} from 'node:events';
import type {Pool,PoolClient} from 'pg';
import * as Y from 'yjs';
import {MAX_FRAME_BYTES,type SceneRef,type MutationReceipt,type SceneContent} from '@collab/contracts';
import {Authority,CollabError,hash,requestID,type Principal,type Decision} from './auth.js';
import {createSceneDoc,materializeScene,validateCandidate,MAX_SNAPSHOT_BYTES,SnapshotHistoryLimitError} from './document.js';
import {parseMutation,commandUpdate} from './commands.js';
import {promoteLegacy} from './promotion.js';
import {RateLimit} from './rate-limit.js';
export type SceneState={doc:Y.Doc;content:SceneContent;documentId:string;sequence:string;stateVector:string;access:'view'|'edit'};
type Row={document_id:string;sequence:string;snapshot_sequence:string;snapshot:Buffer;snapshot_at:Date;legacy_content:unknown;legacy_content_version:string};
export class Persistence extends EventEmitter {
 private rates=new RateLimit();
 constructor(public pool:Pool,public authority:Authority,private options:{promoteV2:boolean;updatesPerMinute?:number}={promoteV2:false}){super();}
 private async transaction<T>(ref:SceneRef,decision:Decision|undefined,fn:(client:PoolClient)=>Promise<T>):Promise<T>{
  const client=await this.pool.connect();let failure=false;
  const connectionError=()=>{failure=true;};client.once('error',connectionError);
  try{
   await client.query('BEGIN');await client.query("SET LOCAL lock_timeout='3s'");await client.query("SET LOCAL statement_timeout='5s'");
   const workspace=await client.query('SELECT acl_version::text FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId]);
   if(!workspace.rowCount)throw new CollabError('FORBIDDEN');
   if(decision){this.fresh(decision);if(decision.workspaceId!==ref.workspaceId||decision.sceneId!==ref.sceneId||decision.aclVersion!==workspace.rows[0].acl_version)throw new CollabError('STALE_AUTHORITY','Authority changed; obtain a fresh decision',409,true);}
   const scene=await client.query('SELECT id FROM scenes WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId,ref.sceneId]);
   if(!scene.rowCount)throw new CollabError('FORBIDDEN');
   const result=await fn(client);if(decision)this.fresh(decision);
   await client.query('COMMIT');return result;
  }catch(error){failure=true;try{await client.query('ROLLBACK');}catch{}throw error;}finally{if(!failure)client.off('error',connectionError);client.release(failure);}
 }
 private fresh(decision:Decision):void {if(Date.now()>=decision.exp*1000)throw new CollabError('STALE_AUTHORITY','Decision expired; obtain a fresh decision',409,true);}
 private async load(client:PoolClient,ref:SceneRef,allowOverLimit=false):Promise<{row:Row;doc:Y.Doc}>{
  let row=(await client.query<Row>('SELECT * FROM collab_documents WHERE scene_id=$1 FOR UPDATE',[ref.sceneId])).rows[0];
  if(!row){
   if(!this.options.promoteV2)throw new CollabError('COLLAB_DISABLED','New collaboration promotions are disabled',409);
   const scene=(await client.query('SELECT content,content_version::text,writer_mode FROM scenes WHERE id=$1',[ref.sceneId])).rows[0];
   if(scene.writer_mode!=='v1')throw new CollabError('UNAVAILABLE','Missing durable document; recovery required',503);
   const raw=scene.content;const content=await promoteLegacy(client,ref,raw);
   const seeded=createSceneDoc(content),snapshot=Y.encodeStateAsUpdate(seeded);seeded.destroy();
   row=(await client.query<Row>('INSERT INTO collab_documents(scene_id,workspace_id,snapshot,legacy_content,legacy_content_version) VALUES($1,$2,$3,$4,$5) RETURNING *',[ref.sceneId,ref.workspaceId,Buffer.from(snapshot),raw,scene.content_version])).rows[0];
   await client.query("UPDATE scenes SET writer_mode='v2',content=$2 WHERE id=$1",[ref.sceneId,JSON.stringify(content)]);
  }
  if(!allowOverLimit&&row.snapshot.length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  const doc=new Y.Doc({gc:false});
  try{
   Y.applyUpdate(doc,row.snapshot);
   const updates=await client.query('SELECT update_bytes FROM collab_updates WHERE scene_id=$1 AND sequence>$2 ORDER BY sequence',[ref.sceneId,row.snapshot_sequence]);
   for(const update of updates.rows)Y.applyUpdate(doc,update.update_bytes);
   if(!allowOverLimit&&Y.encodeStateAsUpdate(doc).length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
   return {row,doc};
  }catch(error){doc.destroy();throw error;}
 }
 async read(ref:SceneRef,principal:Principal):Promise<SceneState>{
  const decision=await this.authority.decide(principal.roomToken,ref,'scene:read',randomUUID(),new Uint8Array());
  this.bindPrincipal(decision,principal);
  return this.transaction(ref,decision,async client=>{const {row,doc}=await this.load(client,ref);try{return {doc,content:materializeScene(doc),documentId:row.document_id,sequence:row.sequence,stateVector:Buffer.from(Y.encodeStateVector(doc)).toString('base64'),access:decision.access};}catch(error){doc.destroy();throw error;}});
 }
 private bindPrincipal(decision:Decision,principal:Principal):void {if(decision.principalId!==principal.principalId||decision.principalType!==principal.principalType)throw new CollabError('FORBIDDEN');}
 async appendUpdate(ref:SceneRef,principal:Principal,requestId:string,bytes:Uint8Array):Promise<MutationReceipt>{
  requestID(requestId);if(bytes.byteLength>MAX_FRAME_BYTES)throw new CollabError('INVALID_UPDATE','Update too large',400);
  bytes=bytes.slice();const decision=await this.authority.decide(principal.roomToken,ref,'scene:update',requestId,bytes);this.bindPrincipal(decision,principal);
  return this.append(ref,decision,bytes,()=>bytes);
 }
 async applyCommands(grant:string,bytes:Uint8Array):Promise<MutationReceipt>{
  if(bytes.byteLength>MAX_FRAME_BYTES)throw new CollabError('INVALID_UPDATE','Command too large',400);
  bytes=bytes.slice();const mutation=parseMutation(bytes),ref={workspaceId:mutation.workspaceId,sceneId:mutation.sceneId};
  const decision=await this.authority.verifyDecision(grant,{...ref,action:'scene:command',requestId:mutation.requestId,payloadHash:hash(bytes),expectedSequence:mutation.expectedSequence});
  return this.append(ref,decision,bytes,doc=>commandUpdate(doc,mutation));
 }
 private async append(ref:SceneRef,decision:Decision,original:Uint8Array,makeUpdate:(doc:Y.Doc)=>Uint8Array):Promise<MutationReceipt>{
  if(decision.access!=='edit')throw new CollabError('FORBIDDEN');
  const requestId=decision.requestId!,payloadHash=hash(original);if(payloadHash!==decision.payloadHash)throw new CollabError('FORBIDDEN');
  let accepted:Uint8Array|undefined;
  const receipt=await this.transaction(ref,decision,async client=>{
   const {row,doc}=await this.load(client,ref);
   try{
    const previous=(await client.query('SELECT * FROM collab_receipts WHERE scene_id=$1 AND request_id=$2',[ref.sceneId,requestId])).rows[0];
    if(previous){if(previous.principal_type!==decision.principalType||previous.principal_id!==decision.principalId||previous.payload_hash!==payloadHash||previous.action!==decision.action)throw new CollabError('REQUEST_ID_REUSED','Request ID already binds another mutation',409);return {requestId,sequence:previous.sequence,stateVector:previous.state_vector};}
    this.rates.take(decision.principalType+':'+decision.principalId,this.options.updatesPerMinute??120);
    if(decision.expectedSequence!==undefined&&decision.expectedSequence!==row.sequence)throw new CollabError('CONFLICT','Expected sequence differs from current sequence',409);
    let validated;
    try{validated=await validateCandidate(doc,makeUpdate(doc));}catch(error){if(error instanceof SnapshotHistoryLimitError)throw error;throw new CollabError('INVALID_UPDATE',error instanceof Error?error.message:'Invalid update',400);}
    // Check the resulting retained history too; never acknowledge a state the service cannot reload.
    Y.applyUpdate(doc,validated.update);if(Y.encodeStateAsUpdate(doc).length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
    this.fresh(decision);const sequence=(BigInt(row.sequence)+1n).toString(),stateVector=Buffer.from(validated.stateVector).toString('base64');
    await client.query('INSERT INTO collab_updates(scene_id,sequence,update_bytes) VALUES($1,$2,$3)',[ref.sceneId,sequence,Buffer.from(validated.update)]);
    await client.query('INSERT INTO collab_receipts(scene_id,request_id,principal_type,principal_id,payload_hash,action,sequence,state_vector) VALUES($1,$2,$3,$4,$5,$6,$7,$8)',[ref.sceneId,requestId,decision.principalType,decision.principalId,payloadHash,decision.action,sequence,stateVector]);
    await client.query('INSERT INTO collab_mutation_audit(scene_id,request_id,workspace_id,principal_type,principal_id,sequence,payload_hash,action,acl_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)',[ref.sceneId,requestId,ref.workspaceId,decision.principalType,decision.principalId,sequence,payloadHash,decision.action,decision.aclVersion]);
    await client.query('UPDATE collab_documents SET sequence=$2 WHERE scene_id=$1',[ref.sceneId,sequence]);
    accepted=validated.update;return {requestId,sequence,stateVector};
   }finally{doc.destroy();}
  });
  if(accepted)this.emit('committed',{ref,receipt,update:accepted});
  return receipt;
 }
 async compactScene(ref:SceneRef):Promise<void>{
  await this.transaction(ref,undefined,async client=>{
   const {row,doc}=await this.load(client,ref);
   try{
    const snapshot=Y.encodeStateAsUpdate(doc),content=materializeScene(doc);
    await client.query('UPDATE collab_documents SET snapshot=$2,snapshot_sequence=sequence,snapshot_at=clock_timestamp() WHERE scene_id=$1',[ref.sceneId,Buffer.from(snapshot)]);
    await client.query('UPDATE scenes SET content=$2,content_version=GREATEST(content_version,$3),updated_at=clock_timestamp() WHERE id=$1',[ref.sceneId,JSON.stringify(content),(BigInt(row.sequence)+BigInt(row.legacy_content_version)).toString()]);
    await client.query('DELETE FROM collab_updates WHERE scene_id=$1 AND sequence<=$2',[ref.sceneId,row.sequence]);
   }finally{doc.destroy();}
  });
 }
 async compactDue():Promise<void>{
  const due=await this.pool.query("SELECT workspace_id,scene_id FROM collab_documents WHERE sequence>snapshot_sequence AND (sequence-snapshot_sequence>=100 OR snapshot_at<=clock_timestamp()-interval '5 seconds')");
  for(const row of due.rows)await this.compactScene({workspaceId:row.workspace_id,sceneId:row.scene_id});
 }
 async exportRecovery(ref:SceneRef):Promise<Record<string,unknown>>{
  return this.transaction(ref,undefined,async client=>{
   const {row,doc}=await this.load(client,ref,true);
   try{
    const updates=(await client.query('SELECT sequence::text,update_bytes FROM collab_updates WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows.map(r=>({sequence:r.sequence,bytes:r.update_bytes.toString('base64'),hash:hash(r.update_bytes)}));
    return {format:1,...ref,documentId:row.document_id,sequence:row.sequence,promotionCheckpoint:{content:row.legacy_content,contentVersion:row.legacy_content_version},snapshotSequence:row.snapshot_sequence,snapshot:row.snapshot.toString('base64'),snapshotHash:hash(row.snapshot),updates,receipts:(await client.query('SELECT * FROM collab_receipts WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,audit:(await client.query('SELECT * FROM collab_mutation_audit WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,content:materializeScene(doc)};
   }finally{doc.destroy();}
  });
 }
}
