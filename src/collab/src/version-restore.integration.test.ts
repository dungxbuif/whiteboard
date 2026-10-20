import {randomUUID} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {afterEach,expect,it} from 'vitest';
import * as Y from 'yjs';
import {hash} from './auth.js';
import {Persistence} from './persistence.js';
import {canonicalRestorePayload,type RestoreVersionInput} from './version-restore.js';
import {canonicalSceneBytes,canonicalVersionPayload,type CaptureVersionInput} from './versions.js';
import {fixture,grantSecret,rectangle,signed} from '../tests/fixture.js';

const cleanup:Array<()=>Promise<void>>=[];afterEach(async()=>{for(const close of cleanup.splice(0).reverse())await close();});
async function setup(){const f=await fixture();cleanup.push(()=>f.close());for(const migration of ['000014_enterprise_feature_policy.up.sql','000015_workspace_events.up.sql','000018_scene_versions.up.sql','000023_version_restore_receipts.up.sql'])await f.pool.query(await readFile(new URL('../../backend/migrations/'+migration,import.meta.url),'utf8'));await f.pool.query('UPDATE workspace_feature_policies SET versions_enabled=true WHERE workspace_id=$1',[f.ref.workspaceId]);const store=new Persistence(f.pool,f.authority,{promoteV2:true});const state=await store.read(f.ref,f.principal);return {...f,store,state};}
async function token(f:Awaited<ReturnType<typeof setup>>,action:string,requestId:string,expectedSequence:string,payload:Uint8Array){const now=Math.floor(Date.now()/1000),aclVersion=(await f.pool.query('SELECT acl_version::text value FROM workspaces WHERE id=$1',[f.ref.workspaceId])).rows[0].value;return signed({iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:randomUUID(),sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:`membership:${f.ref.workspaceId}:${f.principal.principalId}`,access:'edit',aclVersion,action,requestId,payloadHash:hash(payload),expectedSequence},grantSecret);}

it('staleRestoreCommitsNothing and retry creates one recovery checkpoint',async()=>{
 const f=await setup(),create:CaptureVersionInput={requestId:randomUUID(),expectedSequence:f.state.sequence,pinned:false};
 const version=await f.store.captureVersion(f.ref,await token(f,'version:create',create.requestId,create.expectedSequence,canonicalVersionPayload(create)),create);
 const stale:RestoreVersionInput={versionId:version.id,requestId:randomUUID(),expectedSequence:'9'};
 await expect(f.store.restoreVersion(f.ref,await token(f,'version:restore',stale.requestId,stale.expectedSequence,canonicalRestorePayload(f.ref,stale)),stale)).rejects.toMatchObject({code:'CONFLICT'});
 expect((await f.pool.query('SELECT count(*) FROM collab_updates')).rows[0].count).toBe('0');
 expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('1');
 const input:RestoreVersionInput={...stale,requestId:randomUUID(),expectedSequence:f.state.sequence};const grant=await token(f,'version:restore',input.requestId,input.expectedSequence,canonicalRestorePayload(f.ref,input));
 const first=await f.store.restoreVersion(f.ref,grant,input),retry=await f.store.restoreVersion(f.ref,grant,input);
 expect(retry).toEqual(first);expect(first.sequence).toBe('1');
 expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('2');
 expect((await f.pool.query('SELECT count(*) FROM collab_receipts WHERE action=\'version:restore\'')).rows[0].count).toBe('1');
});

it('offline replay after restore is a later edit and document ID remains stable',async()=>{
 const f=await setup(),create:CaptureVersionInput={requestId:randomUUID(),expectedSequence:f.state.sequence,pinned:false};
 const version=await f.store.captureVersion(f.ref,await token(f,'version:create',create.requestId,create.expectedSequence,canonicalVersionPayload(create)),create);
 const input:RestoreVersionInput={versionId:version.id,requestId:randomUUID(),expectedSequence:f.state.sequence};
 const restored=await f.store.restoreVersion(f.ref,await token(f,'version:restore',input.requestId,input.expectedSequence,canonicalRestorePayload(f.ref,input)),input);
 const client=new Y.Doc({gc:false});Y.applyUpdate(client,Y.encodeStateAsUpdate(f.state.doc));const vector=Y.encodeStateVector(client);const element=new Y.Map<unknown>();for(const [key,value] of Object.entries({...rectangle('offline'),x:77}))element.set(key,value);client.getMap<Y.Map<unknown>>('elements').set('offline',element);client.getArray<string>('order').push(['offline']);const offline=Y.encodeStateAsUpdate(client,vector);
 const receipt=await f.store.appendUpdate(f.ref,f.principal,randomUUID(),offline);
 expect(BigInt(receipt.sequence)).toBe(BigInt(restored.sequence)+1n);
 const state=await f.store.read(f.ref,f.principal);expect(state.documentId).toBe(f.state.documentId);expect(state.content.elements[0]).toMatchObject({id:'offline',x:77});
 client.destroy();state.doc.destroy();
});

it('rejects a restore request ID already used by a checkpoint without partial writes',async()=>{
 const f=await setup(),requestId=randomUUID(),create:CaptureVersionInput={requestId,expectedSequence:f.state.sequence,pinned:false};
 const version=await f.store.captureVersion(f.ref,await token(f,'version:create',create.requestId,create.expectedSequence,canonicalVersionPayload(create)),create);
 const input:RestoreVersionInput={versionId:version.id,requestId,expectedSequence:f.state.sequence};
 await expect(f.store.restoreVersion(f.ref,await token(f,'version:restore',input.requestId,input.expectedSequence,canonicalRestorePayload(f.ref,input)),input)).rejects.toMatchObject({code:'REQUEST_ID_REUSED'});
 expect((await f.pool.query("SELECT count(*) FROM collab_receipts WHERE action='version:restore'")).rows[0].count).toBe('0');
 expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('1');
});

it('restores a valid version whose trusted delta exceeds one socket frame',async()=>{
 const f=await setup(),versionId=randomUUID();
 const label={...rectangle('large-label'),type:'text',fontSize:20,fontFamily:1,text:'x'.repeat(1024*1024+64),originalText:'',textAlign:'left',verticalAlign:'top',containerId:null,autoResize:true,lineHeight:1.25};
 const content={elements:[label],appState:{},files:{}},bytes=canonicalSceneBytes(content as any);
 await f.pool.query(`INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,pinned,kind,creator_kind,creator_id,request_id,payload_hash)
  VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,false,'manual','user',$10,$11,$12)`,[versionId,f.ref.workspaceId,f.ref.sceneId,f.state.documentId,f.state.sequence,JSON.stringify(content),Buffer.from(bytes),hash(bytes),bytes.length,f.principal.principalId,randomUUID(),hash(new Uint8Array())]);
 const input:RestoreVersionInput={versionId,requestId:randomUUID(),expectedSequence:f.state.sequence};let accepted=0;f.store.on('committed',({update})=>accepted=update.byteLength);
 const result=await f.store.restoreVersion(f.ref,await token(f,'version:restore',input.requestId,input.expectedSequence,canonicalRestorePayload(f.ref,input)),input);
 expect(accepted).toBeGreaterThan(1024*1024);expect(result.sequence).toBe('1');const state=await f.store.read(f.ref,f.principal);expect((state.content.elements[0] as any).text).toHaveLength(1024*1024+64);state.doc.destroy();
});
