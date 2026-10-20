import type {Pool,PoolClient,QueryResult} from 'pg';

export const ACTIVITY_PROJECTION_BATCH=100;
export const ACTIVITY_PROJECTION_INTERVAL_MS=1000;
const COLLAB_AUDIT_SOURCE='collab_mutation_audit';

export type ActivityProjectionCursor={occurredAt:string;key:string};
export type MutationAuditRow={workspaceId:string;sceneId:string;requestId:string;principalType:'user'|'public'|'agent';principalId:string;action:string;sequence:string;aclVersion:string;createdAt:string};

export interface ActivityProjectionStore {
 transaction<T>(work:(transaction:ActivityProjectionStore)=>Promise<T>):Promise<T>;
 readCursor():Promise<ActivityProjectionCursor>;
 readMutationAudit(cursor:ActivityProjectionCursor,limit:number):Promise<MutationAuditRow[]>;
 appendProjected(row:MutationAuditRow):Promise<void>;
 writeCursor(cursor:ActivityProjectionCursor):Promise<void>;
}

function sourceKey(row:MutationAuditRow){return `${row.sceneId}/${row.requestId}`;}

export async function projectMutationAuditBatch(store:ActivityProjectionStore,limit:number):Promise<number>{
 if(!Number.isInteger(limit)||limit<1||limit>ACTIVITY_PROJECTION_BATCH)throw new Error('activity projection limit must be 1..100');
 return store.transaction(async transaction=>{
  const cursor=await transaction.readCursor();
  const rows=await transaction.readMutationAudit(cursor,limit);
  for(const row of rows)await transaction.appendProjected(row);
  const last=rows.at(-1);if(last)await transaction.writeCursor({occurredAt:last.createdAt,key:sourceKey(last)});
  return rows.length;
 });
}

type Queryable={query<T extends Record<string,unknown>=Record<string,unknown>>(text:string,values?:unknown[]):Promise<QueryResult<T>>};

class PostgresActivityTransaction implements ActivityProjectionStore {
 constructor(private client:Queryable){}
 transaction<T>(_work:(transaction:ActivityProjectionStore)=>Promise<T>):Promise<T>{throw new Error('nested activity projection transaction');}
 async readCursor():Promise<ActivityProjectionCursor>{
  await this.client.query(`INSERT INTO workspace_event_source_cursors(source) VALUES($1) ON CONFLICT(source) DO NOTHING`,[COLLAB_AUDIT_SOURCE]);
  const result=await this.client.query<{occurredAt:string;key:string}>(`SELECT cursor_time::text AS "occurredAt",cursor_key AS key FROM workspace_event_source_cursors WHERE source=$1 FOR UPDATE`,[COLLAB_AUDIT_SOURCE]);
  if(!result.rows[0])throw new Error('activity projection cursor unavailable');return result.rows[0];
 }
 async readMutationAudit(cursor:ActivityProjectionCursor,limit:number):Promise<MutationAuditRow[]>{
  const result=await this.client.query<MutationAuditRow>(`SELECT audit.workspace_id::text AS "workspaceId",audit.scene_id::text AS "sceneId",audit.request_id AS "requestId",audit.principal_type AS "principalType",audit.principal_id AS "principalId",audit.action,audit.sequence::text,audit.acl_version::text AS "aclVersion",audit.created_at::text AS "createdAt" FROM collab_mutation_audit AS audit WHERE NOT EXISTS (SELECT 1 FROM workspace_events AS event WHERE event.source=$1 AND event.source_key=audit.scene_id::text || '/' || audit.request_id) ORDER BY audit.created_at,audit.scene_id::text || '/' || audit.request_id LIMIT $2`,[COLLAB_AUDIT_SOURCE,limit]);
  return result.rows;
 }
 async appendProjected(row:MutationAuditRow):Promise<void>{
  const kind=`collab.${row.action.replace(':','.')}`;
  const metadata=JSON.stringify({action:row.action,sequence:row.sequence,aclVersion:row.aclVersion});
  await this.client.query(`WITH inserted AS (INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,occurred_at,safe_metadata,source,source_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10) ON CONFLICT(source,source_key) DO NOTHING RETURNING id,workspace_id,scene_id) INSERT INTO workspace_event_outbox(event_id,workspace_id,scene_id) SELECT id,workspace_id,scene_id FROM inserted`,[row.workspaceId,row.sceneId,row.principalType,row.principalId,kind,row.requestId,row.createdAt,metadata,COLLAB_AUDIT_SOURCE,sourceKey(row)]);
 }
 async writeCursor(cursor:ActivityProjectionCursor):Promise<void>{
  await this.client.query(`UPDATE workspace_event_source_cursors SET cursor_time=$2::timestamptz,cursor_key=$3,updated_at=clock_timestamp() WHERE source=$1`,[COLLAB_AUDIT_SOURCE,cursor.occurredAt,cursor.key]);
 }
}

export class PostgresActivityProjectionStore implements ActivityProjectionStore {
 constructor(private pool:Pool){}
 async transaction<T>(work:(transaction:ActivityProjectionStore)=>Promise<T>):Promise<T>{
  const client:PoolClient=await this.pool.connect();
  try{await client.query('BEGIN');const result=await work(new PostgresActivityTransaction(client));await client.query('COMMIT');return result;}
  catch(error){await client.query('ROLLBACK').catch(()=>{});throw error;}finally{client.release();}
 }
 readCursor():Promise<ActivityProjectionCursor>{throw new Error('activity projection requires a transaction');}
 readMutationAudit(_cursor:ActivityProjectionCursor,_limit:number):Promise<MutationAuditRow[]>{throw new Error('activity projection requires a transaction');}
 appendProjected(_row:MutationAuditRow):Promise<void>{throw new Error('activity projection requires a transaction');}
 writeCursor(_cursor:ActivityProjectionCursor):Promise<void>{throw new Error('activity projection requires a transaction');}
}

export class ActivityProjector {
 private timer:NodeJS.Timeout;private running?:Promise<void>;private closed=false;
 constructor(private store:ActivityProjectionStore,private onFailure:(error:unknown)=>void){
  this.timer=setInterval(()=>this.run(),ACTIVITY_PROJECTION_INTERVAL_MS);this.timer.unref();this.run();
 }
 private run(){if(this.closed||this.running)return;this.running=projectMutationAuditBatch(this.store,ACTIVITY_PROJECTION_BATCH).then(()=>{}).catch(this.onFailure).finally(()=>{this.running=undefined;});}
 async close(){this.closed=true;clearInterval(this.timer);await this.running;}
}
