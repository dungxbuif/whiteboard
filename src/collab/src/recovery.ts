import {open,readFile,stat} from 'node:fs/promises';
import {dirname} from 'node:path';
import * as Y from 'yjs';
import type {SceneContent} from '@collab/contracts';
import {hash} from './auth.js';
import {materializeScene} from './document.js';

const ARCHIVE_FORMAT='collab-recovery-v1';
const MAX_ARCHIVE_BYTES=256*1024*1024;
const UUID=/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const DIGEST=/^[0-9a-f]{64}$/;
type HistoryRow=Record<string,unknown>;
export type RecoveryCheckpoint={
 format:1;workspaceId:string;sceneId:string;documentId:string;sequence:string;snapshotSequence:string;
 snapshot:string;snapshotHash:string;updates:{sequence:string;bytes:string;hash:string}[];
 receipts:HistoryRow[];audit:HistoryRow[];promotionCheckpoint:{content:unknown;contentVersion:string};content:SceneContent;
};
export type RecoveryArchive={format:typeof ARCHIVE_FORMAT;sha256:string;checkpoint:RecoveryCheckpoint};
function requireCondition(value:unknown,message:string):asserts value {if(!value)throw new Error('Invalid recovery checkpoint: '+message);}
function revision(value:unknown):bigint {requireCondition(typeof value==='string'&&/^(0|[1-9][0-9]*)$/.test(value),'revision');return BigInt(value);}
function canonical(value:unknown):string {
 if(value===null||typeof value!=='object')return JSON.stringify(value);
 if(Array.isArray(value))return '['+value.map(canonical).join(',')+']';
 return '{'+Object.entries(value).sort(([a],[b])=>a<b?-1:a>b?1:0).map(([k,v])=>JSON.stringify(k)+':'+canonical(v)).join(',')+'}';
}
function digest(value:unknown){return hash(Buffer.from(canonical(value)));}
function decode(value:unknown,expected:unknown):Buffer {
 requireCondition(typeof value==='string'&&typeof expected==='string'&&DIGEST.test(expected),'binary hash fields');
 const bytes=Buffer.from(value,'base64');requireCondition(bytes.toString('base64')===value&&hash(bytes)===expected,'binary hash mismatch');return bytes;
}
function validate(checkpoint:RecoveryCheckpoint):SceneContent {
 const c=checkpoint;requireCondition(c&&c.format===1,'format');
 requireCondition([c.workspaceId,c.sceneId,c.documentId].every(v=>typeof v==='string'&&UUID.test(v)),'identity');
 const last=revision(c.sequence),snapshotSequence=revision(c.snapshotSequence);requireCondition(snapshotSequence<=last,'snapshot ahead of durable sequence');
 requireCondition(c.promotionCheckpoint&&Object.hasOwn(c.promotionCheckpoint,'content'),'original promotion JSON');revision(c.promotionCheckpoint.contentVersion);
 requireCondition(Array.isArray(c.updates)&&Array.isArray(c.receipts)&&Array.isArray(c.audit),'history arrays');
 requireCondition(BigInt(c.receipts.length)===last&&c.audit.length===c.receipts.length,'receipt/audit coverage');
 const requestIds=new Set<string>();
 for(let index=0;index<c.receipts.length;index++){
  const receipt=c.receipts[index],audit=c.audit[index];
  requireCondition(receipt&&audit&&revision(receipt.sequence)===BigInt(index)+1n,'receipt sequence continuity');
  requireCondition(receipt.scene_id===c.sceneId&&audit.workspace_id===c.workspaceId,'history scope');
  requireCondition(typeof receipt.request_id==='string'&&receipt.request_id.length>0&&receipt.request_id.length<=128&&!requestIds.has(receipt.request_id),'receipt request identity');requestIds.add(receipt.request_id);
  requireCondition(['user','public','agent'].includes(String(receipt.principal_type))&&typeof receipt.principal_id==='string'&&receipt.principal_id.length>0,'principal');
  requireCondition(['scene:update','scene:command','version:restore'].includes(String(receipt.action))&&typeof receipt.payload_hash==='string'&&DIGEST.test(receipt.payload_hash),'receipt action/hash');
  requireCondition(typeof receipt.state_vector==='string'&&Buffer.from(receipt.state_vector,'base64').toString('base64')===receipt.state_vector,'receipt state vector');
  for(const row of [receipt,audit])requireCondition(typeof row.created_at==='string'&&!Number.isNaN(Date.parse(row.created_at)),'history timestamp');
  revision(audit.acl_version);
  for(const field of ['scene_id','request_id','principal_type','principal_id','payload_hash','action','sequence'])requireCondition(receipt[field]===audit[field],'audit receipt binding: '+field);
 }
 const doc=new Y.Doc({gc:false});
 try{
  Y.applyUpdate(doc,decode(c.snapshot,c.snapshotHash));let sequence=snapshotSequence;
  for(const update of c.updates){requireCondition(revision(update.sequence)===sequence+1n,'ordered update continuity');Y.applyUpdate(doc,decode(update.bytes,update.hash));sequence++;}
  requireCondition(sequence===last,'missing durable updates');
  if(c.receipts.length)requireCondition(c.receipts.at(-1)!.state_vector===Buffer.from(Y.encodeStateVector(doc)).toString('base64'),'latest receipt state vector mismatch');
  const content=materializeScene(doc);requireCondition(canonical(content)===canonical(c.content),'materialized content mismatch');return content;
 }finally{doc.destroy();}
}
/** Seal the complete export, including original promotion JSON and compacted receipts. */
export function sealCheckpoint(checkpoint:Record<string,unknown>):RecoveryArchive {
 const normalized=JSON.parse(JSON.stringify(checkpoint)) as RecoveryCheckpoint;
 validate(normalized);return {format:ARCHIVE_FORMAT,sha256:digest(normalized),checkpoint:normalized};
}
export function verifyArchive(input:unknown):SceneContent {
 const archive=input as RecoveryArchive;
 requireCondition(archive&&archive.format===ARCHIVE_FORMAT&&typeof archive.sha256==='string'&&DIGEST.test(archive.sha256),'archive format/hash');
 requireCondition(digest(archive.checkpoint)===archive.sha256,'archive digest mismatch');return validate(archive.checkpoint);
}
/** Exclusive creation prevents an operator from overwriting a known recovery point. */
export async function writeArchive(path:string,checkpoint:Record<string,unknown>):Promise<RecoveryArchive> {
 const archive=sealCheckpoint(checkpoint),bytes=JSON.stringify(archive)+'\n';
 requireCondition(Buffer.byteLength(bytes)<=MAX_ARCHIVE_BYTES,'archive too large');
 const file=await open(path,'wx',0o600);
 try{await file.writeFile(bytes,'utf8');await file.sync();}finally{await file.close();}
 const directory=await open(dirname(path),'r');try{await directory.sync();}finally{await directory.close();}
 const persisted=await readArchive(path);requireCondition(persisted.sha256===archive.sha256,'archive read-back mismatch');return persisted;
}
export async function readArchive(path:string):Promise<RecoveryArchive> {
 requireCondition((await stat(path)).size<=MAX_ARCHIVE_BYTES,'archive too large');
 const archive=JSON.parse(await readFile(path,'utf8')) as RecoveryArchive;verifyArchive(archive);return archive;
}
