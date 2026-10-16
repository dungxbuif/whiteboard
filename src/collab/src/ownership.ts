import type {PoolClient} from 'pg';
import {CollabError} from './auth.js';
export const OWNER_KEY="hashtext(current_database()),hashtext(current_schema() || ':collab-writer')";
export const HANDOFF_KEY="hashtext(current_database()),hashtext(current_schema() || ':collab-handoff')";
export type OwnerLease=Readonly<{pid:number;backendStart:string}>;
/** The shared transaction barrier and exclusive takeover barrier close the precommit-check race. */
export class RuntimeOwnership {
 private lease:OwnerLease|undefined;
 activate(lease:OwnerLease){this.lease=Object.freeze({...lease});}
 invalidate(){this.lease=undefined;}
 private unavailable(){return new CollabError('UNAVAILABLE','Runtime ownership unavailable; retain pending mutation',503,true);}
 assert(lease?:OwnerLease):OwnerLease {if(!this.lease||(lease!==undefined&&this.lease!==lease))throw this.unavailable();return this.lease;}
 async enter(client:PoolClient):Promise<OwnerLease>{
  const lease=this.assert();
  await client.query(`SELECT pg_advisory_xact_lock_shared(${HANDOFF_KEY})`);
  await this.check(client,lease);return lease;
 }
 async check(client:PoolClient,lease:OwnerLease):Promise<void>{
  this.assert(lease);
  const result=await client.query(`SELECT EXISTS (
   SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid
   WHERE l.locktype='advisory' AND l.granted AND l.mode='ExclusiveLock'
    AND l.classid=hashtext(current_database())::oid
    AND l.objid=hashtext(current_schema() || ':collab-writer')::oid AND l.objsubid=2
    AND l.pid=$1 AND a.backend_start=$2::timestamptz
  ) AS owned`,[lease.pid,lease.backendStart]);
  if(!result.rows[0].owned)throw this.unavailable();this.assert(lease);
 }
}
