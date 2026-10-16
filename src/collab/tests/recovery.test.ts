import {afterEach,expect,it} from 'vitest';
import {mkdtemp,readFile,rm,stat,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import * as Y from 'yjs';
import {createSceneDoc} from '../src/document.js';
import {hash} from '../src/auth.js';
import {sealCheckpoint,verifyArchive,writeArchive,readArchive} from '../src/recovery.js';
const paths:string[]=[];
afterEach(async()=>{for(const path of paths.splice(0))await rm(path,{recursive:true,force:true});});
function checkpoint(){
 const doc=createSceneDoc({elements:[],appState:{theme:'light'},files:{}});
 const snapshot=Buffer.from(Y.encodeStateAsUpdate(doc));const vector=Y.encodeStateVector(doc);
 doc.getMap('appState').set('theme','dark');const bytes=Buffer.from(Y.encodeStateAsUpdate(doc,vector));const stateVector=Buffer.from(Y.encodeStateVector(doc)).toString('base64');doc.destroy();
 const receipt={scene_id:'11111111-1111-4111-8111-111111111111',request_id:'acknowledged',principal_type:'user',principal_id:'user',payload_hash:hash(bytes),action:'scene:update',sequence:'1',state_vector:stateVector,created_at:'2026-10-16T00:00:00.000Z'};
 return {format:1,workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:receipt.scene_id,documentId:'33333333-3333-4333-8333-333333333333',sequence:'1',snapshotSequence:'0',snapshot:snapshot.toString('base64'),snapshotHash:hash(snapshot),updates:[{sequence:'1',bytes:bytes.toString('base64'),hash:hash(bytes)}],receipts:[receipt],audit:[{...receipt,workspace_id:'22222222-2222-4222-8222-222222222222',acl_version:'2'}],promotionCheckpoint:{content:{elements:[],appState:{theme:'light'},files:{}},contentVersion:'7'},content:{elements:[],appState:{theme:'dark'},files:{}}};
}
it('reconstructs latest durable content instead of promotion JSON',()=>{expect(verifyArchive(sealCheckpoint(checkpoint())).appState.theme).toBe('dark');});
it.each(['snapshot','update','content','receipt','audit','promotion'])('rejects %s corruption without mutating the source',kind=>{
 const archive=sealCheckpoint(checkpoint());const before=JSON.stringify(archive);const corrupted=JSON.parse(before);
 if(kind==='snapshot')corrupted.checkpoint.snapshot='AA==';if(kind==='update')corrupted.checkpoint.updates[0].bytes='AA==';if(kind==='content')corrupted.checkpoint.content.appState.theme='light';if(kind==='receipt')corrupted.checkpoint.receipts=[];if(kind==='audit')corrupted.checkpoint.audit=[];if(kind==='promotion')corrupted.checkpoint.promotionCheckpoint.contentVersion='8';
 expect(()=>verifyArchive(corrupted)).toThrow();expect(JSON.stringify(archive)).toBe(before);
});
it.each(['missing update','missing receipt','missing audit','wrong snapshot hash','wrong update hash','wrong materialization','wrong audit binding'])('refuses sealing %s',kind=>{
 const c=checkpoint();if(kind==='missing update')c.updates=[];if(kind==='missing receipt')c.receipts=[];if(kind==='missing audit')c.audit=[];if(kind==='wrong snapshot hash')c.snapshotHash='0'.repeat(64);if(kind==='wrong update hash')c.updates[0].hash='0'.repeat(64);if(kind==='wrong materialization')c.content.appState.theme='light';if(kind==='wrong audit binding')c.audit[0].principal_id='different';expect(()=>sealCheckpoint(c)).toThrow();
});
it('writes a private exclusive archive and verifies the file read back',async()=>{
 const dir=await mkdtemp(join(tmpdir(),'collab-recovery-'));paths.push(dir);const path=join(dir,'checkpoint.json');const c=checkpoint();
 await writeArchive(path,c);expect((await stat(path)).mode&0o777).toBe(0o600);expect(verifyArchive(await readArchive(path)).appState.theme).toBe('dark');
 const original=await readFile(path,'utf8');await expect(writeArchive(path,c)).rejects.toThrow();expect(await readFile(path,'utf8')).toBe(original);
 await writeFile(path,original.replace('acknowledged','corrupted'));await expect(readArchive(path)).rejects.toThrow();
});

it('rejects a latest receipt that cannot acknowledge the reconstructed state',()=>{const c=checkpoint();c.receipts[0].state_vector='AA==';expect(()=>sealCheckpoint(c)).toThrow(/state vector/);});
