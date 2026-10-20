import {randomUUID} from 'node:crypto';
import {EventEmitter} from 'node:events';
import type {Pool,PoolClient} from 'pg';
import * as Y from 'yjs';
import {MAX_FRAME_BYTES,validateSceneContent,type SceneRef,type MutationReceipt,type SceneContent} from '@collab/contracts';
import {Authority,CollabError,hash,requestID,revision,type Principal,type Decision,type PresentationDecision} from './auth.js';
import {createSceneDoc,materializeScene,validateCandidate,validateRestoreCandidate,MAX_SNAPSHOT_BYTES,SnapshotHistoryLimitError} from './document.js';
import {parseMutation,commandUpdate} from './commands.js';
import {promoteLegacy} from './promotion.js';
import {RateLimit} from './rate-limit.js';
import {RuntimeOwnership,type OwnerLease} from './ownership.js';
import {MAX_VERSION_BYTES,canonicalSceneBytes,canonicalVersionPayload,referencedAssetIDs,type CaptureVersionInput,type VersionSummary} from './versions.js';
import {canonicalRestorePayload,restoreUpdate,type RestoreVersionInput,type RestoreVersionResult} from './version-restore.js';
import {libraryInsertionUpdate,materializeLibraryInsertion,parseLibraryInsertion,type LibraryInsertionResult} from './library.js';
export type SceneState={doc:Y.Doc;content:SceneContent;documentId:string;sequence:string;stateVector:string;access:'view'|'edit'};
type Row={document_id:string;sequence:string;snapshot_sequence:string;snapshot:Buffer;snapshot_at:Date;legacy_content:unknown;legacy_content_version:string};
function versionSummary(row:Record<string,unknown>):VersionSummary{
 return {id:String(row.id),documentId:String(row.document_id),sequence:String(row.source_sequence),hash:String(row.content_hash),...(row.label===null||row.label===undefined?{}:{label:String(row.label)}),pinned:Boolean(row.pinned),kind:row.kind as 'manual'|'automatic',metadataRevision:String(row.metadata_revision),createdAt:(row.created_at as Date).toISOString(),updatedAt:(row.updated_at as Date).toISOString(),creator:{kind:row.creator_kind as 'user'|'system',id:String(row.creator_id)}};
}
export class Persistence extends EventEmitter {
 private rates=new RateLimit();
 private ownership:RuntimeOwnership|undefined;
 bindRuntimeOwnership(ownership:RuntimeOwnership){this.ownership=ownership;}
 constructor(public pool:Pool,public authority:Authority,private options:{promoteV2:boolean;updatesPerMinute?:number;now?:()=>number}={promoteV2:false}){super();}
 private async transaction<T>(ref:SceneRef,decision:Decision|undefined,fn:(client:PoolClient)=>Promise<T>,skipInactive=false):Promise<T>{
  const ownership=this.ownership;ownership?.assert();
  const client=await this.pool.connect();let failure=false;let lease:OwnerLease|undefined;
  const connectionError=()=>{failure=true;};client.once('error',connectionError);
  try{
   await client.query('BEGIN');await client.query("SET LOCAL lock_timeout='3s'");await client.query("SET LOCAL statement_timeout='5s'");
   if(ownership)lease=await ownership.enter(client);
   const workspace=await client.query('SELECT acl_version::text FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId]);
   if(!workspace.rowCount){if(skipInactive){await client.query('ROLLBACK');return undefined as T;}throw new CollabError('FORBIDDEN');}
   if(decision){this.fresh(decision);if(decision.workspaceId!==ref.workspaceId||decision.sceneId!==ref.sceneId||decision.aclVersion!==workspace.rows[0].acl_version)throw new CollabError('STALE_AUTHORITY','Authority changed; obtain a fresh decision',409,true);}
   const scene=await client.query('SELECT id FROM scenes WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE',[ref.workspaceId,ref.sceneId]);
   if(!scene.rowCount){if(skipInactive){await client.query('ROLLBACK');return undefined as T;}throw new CollabError('FORBIDDEN');}
   const result=await fn(client);if(decision)this.fresh(decision);
   if(ownership)await ownership.check(client,lease!);
   await client.query('COMMIT');
   // A loss in the final-check/COMMIT gap leaves an ambiguous durable receipt, never an ack/publication.
   if(ownership)await ownership.check(client,lease!);return result;
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
 async readPresentation(ref:SceneRef,principal:Principal,credential:PresentationDecision):Promise<SceneState>{
  if(credential.purpose!=='presentation-audience'||credential.access!=='view'||credential.workspaceId!==ref.workspaceId||credential.sceneId!==ref.sceneId||credential.principalId!==principal.principalId||principal.principalType!=='public')throw new CollabError('FORBIDDEN');
  const decision={...credential,purpose:'room',access:'view'} as Decision;
  return this.transaction(ref,decision,async client=>{const {row,doc}=await this.load(client,ref);try{return {doc,content:materializeScene(doc),documentId:row.document_id,sequence:row.sequence,stateVector:Buffer.from(Y.encodeStateVector(doc)).toString('base64'),access:'view'};}catch(error){doc.destroy();throw error;}});
 }
 async readProjection(grant:string,ref:SceneRef,exporting=false):Promise<Record<string,unknown>>{
  const decision=await this.authority.verifyDecision(grant,{...ref,action:exporting?'scene:export':'scene:read',payloadHash:hash(new Uint8Array())});
  return this.transaction(ref,decision,async client=>{
   const {row,doc}=await this.load(client,ref);
   try{
    const content=materializeScene(doc);
    const result={...ref,documentId:row.document_id,sequence:row.sequence,stateVector:Buffer.from(Y.encodeStateVector(doc)).toString('base64'),content};
    if(!exporting&&(content.elements.length>500||Buffer.byteLength(JSON.stringify(result))>256*1024-2048))throw new CollabError('PROJECTION_TOO_LARGE','Scene exceeds projection limit',413);
    return result;
   }finally{doc.destroy();}
  });
 }
 async readActivity(grant:string,ref:SceneRef,cursor='0',limit=50):Promise<Record<string,unknown>>{
  if(!revision(cursor)||!Number.isInteger(limit)||limit<1||limit>50)throw new CollabError('INVALID_UPDATE','Invalid activity page',400);
  const decision=await this.authority.verifyDecision(grant,{...ref,action:'scene:read',payloadHash:hash(new Uint8Array())});
  return this.transaction(ref,decision,async client=>{
   const rows=(await client.query('SELECT sequence::text,request_id AS "requestId",principal_type AS "principalType",action,created_at AS "createdAt" FROM collab_mutation_audit AS activity WHERE workspace_id=$1 AND scene_id=$2 AND sequence>$3 ORDER BY activity.sequence LIMIT $4',[ref.workspaceId,ref.sceneId,cursor,limit+1])).rows;
   return {items:rows.slice(0,limit),...(rows.length>limit?{nextCursor:rows[limit-1].sequence}:{})};
  });
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
 async applyLibraryInsertion(grant:string,bytes:Uint8Array):Promise<LibraryInsertionResult>{
  if(bytes.byteLength>MAX_VERSION_BYTES)throw new CollabError('INVALID_UPDATE','Library insertion too large',400);
  bytes=bytes.slice();const input=parseLibraryInsertion(bytes),ref={workspaceId:input.workspaceId,sceneId:input.sceneId},content=materializeLibraryInsertion(input);
  const decision=await this.authority.verifyDecision(grant,{...ref,action:'library:insert',requestId:input.requestId,payloadHash:hash(bytes),expectedSequence:input.expectedSequence});
  if(decision.principalType!=='user')throw new CollabError('FORBIDDEN');
  const receipt=await this.append(ref,decision,bytes,doc=>libraryInsertionUpdate(doc,content));return {...receipt,elementIds:content.elements.map(element=>element.id)};
 }
 async captureVersion(ref:SceneRef,grant:string,input:CaptureVersionInput):Promise<VersionSummary>{
  requestID(input.requestId);
  if(!revision(input.expectedSequence)||typeof input.pinned!=='boolean'||input.label!==undefined&&(typeof input.label!=='string'||input.label.trim()!==input.label||input.label.length<1||input.label.length>255))throw new CollabError('INVALID_UPDATE','Invalid version input',400);
  const payload=canonicalVersionPayload(input),payloadHash=hash(payload);
  const decision=await this.authority.verifyDecision(grant,{...ref,action:'version:create',requestId:input.requestId,payloadHash,expectedSequence:input.expectedSequence});
  if(decision.access!=='edit'||decision.principalType!=='user')throw new CollabError('FORBIDDEN');
  return this.transaction(ref,decision,async client=>{
   if(!await this.versionsEnabled(client,ref))throw new CollabError('FEATURE_DISABLED','Version history is disabled',403);
   const previous=(await client.query('SELECT id,document_id,source_sequence::text,content_hash,label,pinned,kind,creator_kind,creator_id,metadata_revision::text,created_at,updated_at,payload_hash FROM scene_versions WHERE scene_id=$1 AND request_id=$2',[ref.sceneId,input.requestId])).rows[0];
   if(previous){
    if(previous.payload_hash!==payloadHash||previous.creator_kind!==decision.principalType||previous.creator_id!==decision.principalId)throw new CollabError('REQUEST_ID_REUSED','Request ID already binds another version',409);
    return versionSummary(previous);
   }
   const {row,doc}=await this.load(client,ref);
   try{
    if(row.sequence!==input.expectedSequence)throw new CollabError('CONFLICT','Expected sequence differs from current sequence',409);
    const content=materializeScene(doc),contentBytes=canonicalSceneBytes(content);
    if(contentBytes.byteLength>MAX_VERSION_BYTES)throw new CollabError('VERSION_TOO_LARGE','Version exceeds 25 MiB',413);
    const assetIDs=referencedAssetIDs(content);
    if(assetIDs.length){
     const assets=await client.query('SELECT id FROM scene_assets WHERE workspace_id=$1 AND scene_id=$2 AND id=ANY($3::uuid[]) AND deleted_at IS NULL FOR SHARE',[ref.workspaceId,ref.sceneId,assetIDs]);
     if(assets.rowCount!==assetIDs.length)throw new CollabError('VERSION_ASSET_UNAVAILABLE','A referenced image is unavailable',409);
    }
    const id=randomUUID(),created=(await client.query(`INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,label,pinned,kind,creator_kind,creator_id,request_id,payload_hash)
     VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'manual',$12,$13,$14,$15)
     RETURNING id,document_id,source_sequence::text,content_hash,label,pinned,kind,creator_kind,creator_id,metadata_revision::text,created_at,updated_at`,[id,ref.workspaceId,ref.sceneId,row.document_id,row.sequence,JSON.stringify(content),Buffer.from(contentBytes),hash(contentBytes),contentBytes.byteLength,input.label??null,input.pinned,decision.principalType,decision.principalId,input.requestId,payloadHash])).rows[0];
    for(const assetID of assetIDs)await client.query('INSERT INTO version_assets(version_id,workspace_id,scene_id,asset_id) VALUES($1,$2,$3,$4)',[id,ref.workspaceId,ref.sceneId,assetID]);
    const event=(await client.query("INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key) VALUES($1,$2,$3,$4,'version.created',$5,$6,'scene_version',$5) RETURNING id",[ref.workspaceId,ref.sceneId,decision.principalType,decision.principalId,id,JSON.stringify({versionId:id,sequence:row.sequence,kind:'manual',pinned:input.pinned})])).rows[0];
    await client.query('INSERT INTO workspace_event_outbox(event_id,workspace_id,scene_id) VALUES($1,$2,$3)',[event.id,ref.workspaceId,ref.sceneId]);
    return versionSummary(created);
   }finally{doc.destroy();}
  });
 }
 async captureAutomaticVersion(ref:SceneRef):Promise<VersionSummary|undefined>{
  return this.transaction(ref,undefined,async client=>{
   if(!await this.versionsEnabled(client,ref))return undefined;
   const {row,doc}=await this.load(client,ref);
   try{
    const previous=(await client.query("SELECT source_sequence::text,created_at FROM scene_versions WHERE scene_id=$1 AND kind='automatic' ORDER BY created_at DESC,id DESC LIMIT 1",[ref.sceneId])).rows[0];
    if(previous&&(previous.source_sequence===row.sequence||new Date(previous.created_at).getTime()>(this.options.now?.()??Date.now())-60_000))return undefined;
    const content=materializeScene(doc),contentBytes=canonicalSceneBytes(content);
    if(contentBytes.byteLength>MAX_VERSION_BYTES)throw new CollabError('VERSION_TOO_LARGE','Version exceeds 25 MiB',413);
    const assetIDs=referencedAssetIDs(content);
    if(assetIDs.length){
     const assets=await client.query('SELECT id FROM scene_assets WHERE workspace_id=$1 AND scene_id=$2 AND id=ANY($3::uuid[]) AND deleted_at IS NULL FOR SHARE',[ref.workspaceId,ref.sceneId,assetIDs]);
     if(assets.rowCount!==assetIDs.length)throw new CollabError('VERSION_ASSET_UNAVAILABLE','A referenced image is unavailable',409);
    }
    const id=randomUUID(),requestId='automatic:'+randomUUID(),payloadHash=hash(new TextEncoder().encode(JSON.stringify({requestId,sequence:row.sequence})));
    const created=(await client.query(`INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,pinned,kind,creator_kind,creator_id,request_id,payload_hash)
     VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,false,'automatic','system','collab-checkpointer',$10,$11)
     RETURNING id,document_id,source_sequence::text,content_hash,label,pinned,kind,creator_kind,creator_id,metadata_revision::text,created_at,updated_at`,[id,ref.workspaceId,ref.sceneId,row.document_id,row.sequence,JSON.stringify(content),Buffer.from(contentBytes),hash(contentBytes),contentBytes.byteLength,requestId,payloadHash])).rows[0];
    for(const assetID of assetIDs)await client.query('INSERT INTO version_assets(version_id,workspace_id,scene_id,asset_id) VALUES($1,$2,$3,$4)',[id,ref.workspaceId,ref.sceneId,assetID]);
    const event=(await client.query("INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key) VALUES($1,$2,'system','collab-checkpointer','version.created',$3,$4,'scene_version',$3) RETURNING id",[ref.workspaceId,ref.sceneId,id,JSON.stringify({versionId:id,sequence:row.sequence,kind:'automatic',pinned:false})])).rows[0];
    await client.query('INSERT INTO workspace_event_outbox(event_id,workspace_id,scene_id) VALUES($1,$2,$3)',[event.id,ref.workspaceId,ref.sceneId]);
    return versionSummary(created);
   }finally{doc.destroy();}
  },true);
 }
 async restoreVersion(ref:SceneRef,grant:string,input:RestoreVersionInput):Promise<RestoreVersionResult>{
  requestID(input.requestId);
  if(!revision(input.expectedSequence)||!/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(input.versionId))throw new CollabError('INVALID_UPDATE','Invalid restore input',400);
  const payload=canonicalRestorePayload(ref,input),payloadHash=hash(payload);
  const decision=await this.authority.verifyDecision(grant,{...ref,action:'version:restore',requestId:input.requestId,payloadHash,expectedSequence:input.expectedSequence});
  if(decision.access!=='edit'||decision.principalType!=='user')throw new CollabError('FORBIDDEN');
  let accepted:Uint8Array|undefined;
  const result=await this.transaction(ref,decision,async client=>{
   if(!await this.versionsEnabled(client,ref))throw new CollabError('FEATURE_DISABLED','Version history is disabled',403);
   const previous=(await client.query('SELECT * FROM collab_receipts WHERE scene_id=$1 AND request_id=$2',[ref.sceneId,input.requestId])).rows[0];
   if(previous){
    if(previous.principal_type!==decision.principalType||previous.principal_id!==decision.principalId||previous.payload_hash!==payloadHash||previous.action!=='version:restore')throw new CollabError('REQUEST_ID_REUSED','Request ID already binds another mutation',409);
    const recovery=(await client.query('SELECT id FROM scene_versions WHERE scene_id=$1 AND request_id=$2',[ref.sceneId,input.requestId])).rows[0];
    if(!recovery)throw new CollabError('UNAVAILABLE','Restore recovery checkpoint is missing',503);
    return {requestId:input.requestId,sequence:String(previous.sequence),stateVector:String(previous.state_vector),preRestoreVersionId:String(recovery.id)};
   }
   const checkpointCollision=(await client.query('SELECT id FROM scene_versions WHERE scene_id=$1 AND request_id=$2',[ref.sceneId,input.requestId])).rows[0];
   if(checkpointCollision)throw new CollabError('REQUEST_ID_REUSED','Request ID already belongs to a version checkpoint',409);
   const {row,doc}=await this.load(client,ref);
   try{
    if(row.sequence!==input.expectedSequence)throw new CollabError('CONFLICT','Expected sequence differs from current sequence',409);
    const targetRow=(await client.query('SELECT document_id,content FROM scene_versions WHERE workspace_id=$1 AND scene_id=$2 AND id=$3',[ref.workspaceId,ref.sceneId,input.versionId])).rows[0];
    if(!targetRow||targetRow.document_id!==row.document_id)throw new CollabError('VERSION_INCOMPATIBLE','Version does not belong to the current document',409);
    let target:SceneContent;try{target=validateSceneContent(targetRow.content);}catch{throw new CollabError('VERSION_INCOMPATIBLE','Version content is incompatible',409);}
    const targetAssets=referencedAssetIDs(target);
    if(targetAssets.length){
     const pins=await client.query(`SELECT va.asset_id FROM version_assets va JOIN scene_assets a ON a.id=va.asset_id AND a.workspace_id=va.workspace_id AND a.scene_id=va.scene_id AND a.deleted_at IS NULL
      WHERE va.version_id=$1 AND va.workspace_id=$2 AND va.scene_id=$3 AND va.asset_id=ANY($4::uuid[]) FOR SHARE OF a`,[input.versionId,ref.workspaceId,ref.sceneId,targetAssets]);
     if(pins.rowCount!==targetAssets.length)throw new CollabError('VERSION_ASSET_UNAVAILABLE','A version image is unavailable',409);
    }
    const current=materializeScene(doc),currentBytes=canonicalSceneBytes(current),currentAssets=referencedAssetIDs(current);
    if(currentBytes.byteLength>MAX_VERSION_BYTES)throw new CollabError('VERSION_TOO_LARGE','Recovery version exceeds 25 MiB',413);
    if(currentAssets.length){const assets=await client.query('SELECT id FROM scene_assets WHERE workspace_id=$1 AND scene_id=$2 AND id=ANY($3::uuid[]) AND deleted_at IS NULL FOR SHARE',[ref.workspaceId,ref.sceneId,currentAssets]);if(assets.rowCount!==currentAssets.length)throw new CollabError('VERSION_ASSET_UNAVAILABLE','A recovery image is unavailable',409);}
    let validated;try{validated=await validateRestoreCandidate(doc,restoreUpdate(doc,target));}catch(error){if(error instanceof SnapshotHistoryLimitError)throw error;throw new CollabError('INVALID_UPDATE',error instanceof Error?error.message:'Invalid restore',400);}
    Y.applyUpdate(doc,validated.update);if(Y.encodeStateAsUpdate(doc).length>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
    const preRestoreVersionId=randomUUID();
    await client.query(`INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,label,pinned,kind,creator_kind,creator_id,request_id,payload_hash)
     VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'Before restore',true,'manual',$10,$11,$12,$13)`,[preRestoreVersionId,ref.workspaceId,ref.sceneId,row.document_id,row.sequence,JSON.stringify(current),Buffer.from(currentBytes),hash(currentBytes),currentBytes.byteLength,decision.principalType,decision.principalId,input.requestId,payloadHash]);
    for(const assetID of currentAssets)await client.query('INSERT INTO version_assets(version_id,workspace_id,scene_id,asset_id) VALUES($1,$2,$3,$4)',[preRestoreVersionId,ref.workspaceId,ref.sceneId,assetID]);
    this.fresh(decision);const sequence=(BigInt(row.sequence)+1n).toString(),stateVector=Buffer.from(validated.stateVector).toString('base64');
    await client.query('INSERT INTO collab_updates(scene_id,sequence,update_bytes) VALUES($1,$2,$3)',[ref.sceneId,sequence,Buffer.from(validated.update)]);
    await client.query('INSERT INTO collab_receipts(scene_id,request_id,principal_type,principal_id,payload_hash,action,sequence,state_vector) VALUES($1,$2,$3,$4,$5,$6,$7,$8)',[ref.sceneId,input.requestId,decision.principalType,decision.principalId,payloadHash,'version:restore',sequence,stateVector]);
    await client.query('INSERT INTO collab_mutation_audit(scene_id,request_id,workspace_id,principal_type,principal_id,sequence,payload_hash,action,acl_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)',[ref.sceneId,input.requestId,ref.workspaceId,decision.principalType,decision.principalId,sequence,payloadHash,'version:restore',decision.aclVersion]);
    await client.query('UPDATE collab_documents SET sequence=$2 WHERE scene_id=$1',[ref.sceneId,sequence]);
    const recoveryEvent=(await client.query("INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key) VALUES($1,$2,$3,$4,'version.created',$5,$6,'scene_version',$5) RETURNING id",[ref.workspaceId,ref.sceneId,decision.principalType,decision.principalId,preRestoreVersionId,JSON.stringify({versionId:preRestoreVersionId,sequence:row.sequence,kind:'manual',pinned:true,recovery:true})])).rows[0];
    const restoreEvent=(await client.query("INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key) VALUES($1,$2,$3,$4,'version.restored',$5,$6,'scene_version_restore',$7) RETURNING id",[ref.workspaceId,ref.sceneId,decision.principalType,decision.principalId,input.versionId,JSON.stringify({versionId:input.versionId,preRestoreVersionId,sequence,requestId:input.requestId}),input.requestId])).rows[0];
    for(const event of [recoveryEvent,restoreEvent])await client.query('INSERT INTO workspace_event_outbox(event_id,workspace_id,scene_id) VALUES($1,$2,$3)',[event.id,ref.workspaceId,ref.sceneId]);
    accepted=validated.update;return {requestId:input.requestId,sequence,stateVector,preRestoreVersionId};
   }finally{doc.destroy();}
  });
  if(accepted)this.emit('committed',{ref,receipt:result,update:accepted});
  return result;
 }
 async listDueVersionScenes(limit=64,after?:SceneRef):Promise<Array<{ref:SceneRef;sequence:string}>>{
  this.ownership?.assert();
  const rows=await this.pool.query(`SELECT d.workspace_id,d.scene_id,d.sequence::text FROM collab_documents d
   JOIN workspaces w ON w.id=d.workspace_id AND w.deleted_at IS NULL
   JOIN scenes s ON s.id=d.scene_id AND s.workspace_id=d.workspace_id AND s.deleted_at IS NULL
   JOIN workspace_feature_policies policy ON policy.workspace_id=d.workspace_id AND policy.versions_enabled
   LEFT JOIN LATERAL (SELECT max(source_sequence) AS sequence FROM scene_versions WHERE scene_id=d.scene_id) v ON true
   JOIN LATERAL (SELECT min(created_at) AS dirty_at FROM collab_mutation_audit WHERE scene_id=d.scene_id AND sequence>COALESCE(v.sequence,-1)) dirty ON dirty.dirty_at IS NOT NULL
   WHERE d.sequence>COALESCE(v.sequence,-1) AND dirty.dirty_at<=clock_timestamp()-interval '5 minutes'
    AND ($2::uuid IS NULL OR (d.workspace_id,d.scene_id)>($2::uuid,$3::uuid))
   ORDER BY d.workspace_id,d.scene_id LIMIT $1`,[Math.max(1,Math.min(64,limit)),after?.workspaceId??null,after?.sceneId??null]);
  return rows.rows.map(row=>({ref:{workspaceId:row.workspace_id,sceneId:row.scene_id},sequence:row.sequence}));
 }
 private async versionsEnabled(client:PoolClient,ref:SceneRef):Promise<boolean>{
  const policy=await client.query('SELECT versions_enabled FROM workspace_feature_policies WHERE workspace_id=$1',[ref.workspaceId]);
  return policy.rowCount===1&&policy.rows[0].versions_enabled===true;
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
   const pending=await client.query('SELECT sequence,snapshot_sequence FROM collab_documents WHERE scene_id=$1 FOR UPDATE',[ref.sceneId]);
   if(!pending.rowCount||pending.rows[0].sequence===pending.rows[0].snapshot_sequence)return;
   const {row,doc}=await this.load(client,ref);
   try{
    const snapshot=Y.encodeStateAsUpdate(doc),content=materializeScene(doc);
    await client.query('UPDATE collab_documents SET snapshot=$2,snapshot_sequence=sequence,snapshot_at=clock_timestamp() WHERE scene_id=$1',[ref.sceneId,Buffer.from(snapshot)]);
    await client.query('UPDATE scenes SET content=$2,content_version=GREATEST(content_version,$3),updated_at=clock_timestamp() WHERE id=$1',[ref.sceneId,JSON.stringify(content),(BigInt(row.sequence)+BigInt(row.legacy_content_version)).toString()]);
    await client.query('DELETE FROM collab_updates WHERE scene_id=$1 AND sequence<=$2',[ref.sceneId,row.sequence]);
   }finally{doc.destroy();}
  },true);
 }
 async listDueScenes(limit=64,after?:SceneRef):Promise<SceneRef[]>{
  this.ownership?.assert();
  const client=await this.pool.connect();let failure=false;
  const connectionError=()=>{failure=true;};client.once('error',connectionError);
  try{
   await client.query('BEGIN');await client.query("SET LOCAL statement_timeout='5s'");
   const due=await client.query(`SELECT d.workspace_id,d.scene_id FROM collab_documents d
    JOIN workspaces w ON w.id=d.workspace_id AND w.deleted_at IS NULL
    JOIN scenes s ON s.id=d.scene_id AND s.workspace_id=d.workspace_id AND s.deleted_at IS NULL
    WHERE d.sequence>d.snapshot_sequence AND (d.sequence-d.snapshot_sequence>=100 OR d.snapshot_at<=clock_timestamp()-interval '5 seconds')
     AND ($2::uuid IS NULL OR (d.workspace_id,d.scene_id)>($2::uuid,$3::uuid))
    ORDER BY d.workspace_id,d.scene_id LIMIT $1`,[Math.max(1,Math.min(64,limit)),after?.workspaceId??null,after?.sceneId??null]);
   await client.query('COMMIT');return due.rows.map(row=>({workspaceId:row.workspace_id,sceneId:row.scene_id}));
  }catch(error){failure=true;try{await client.query('ROLLBACK');}catch{}throw error;}
  finally{if(!failure)client.off('error',connectionError);client.release(failure);}
 }
 async compactDue():Promise<void>{
  const errors:unknown[]=[];
  for(const ref of await this.listDueScenes())try{await this.compactScene(ref);}catch(error){errors.push(error);}
  if(errors.length)throw new AggregateError(errors,'Scene compaction failures');
 }
 async exportRecovery(ref:SceneRef):Promise<Record<string,unknown>>{
  return this.transaction(ref,undefined,async client=>{
   const {row,doc}=await this.load(client,ref,true);
   try{
    const updates=(await client.query('SELECT sequence::text,update_bytes FROM collab_updates AS updates WHERE scene_id=$1 ORDER BY updates.sequence',[ref.sceneId])).rows.map(r=>({sequence:r.sequence,bytes:r.update_bytes.toString('base64'),hash:hash(r.update_bytes)}));
    return {format:1,...ref,documentId:row.document_id,sequence:row.sequence,promotionCheckpoint:{content:row.legacy_content,contentVersion:row.legacy_content_version},snapshotSequence:row.snapshot_sequence,snapshot:row.snapshot.toString('base64'),snapshotHash:hash(row.snapshot),updates,receipts:(await client.query('SELECT * FROM collab_receipts WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,audit:(await client.query('SELECT * FROM collab_mutation_audit WHERE scene_id=$1 ORDER BY sequence',[ref.sceneId])).rows,content:materializeScene(doc)};
   }finally{doc.destroy();}
  });
 }
}
