import {afterEach,describe,expect,it} from 'vitest';
import {Persistence} from '../src/persistence.js';
import {fixture,grantSecret,rectangle,signed} from './fixture.js';
import {createHash,randomUUID} from 'node:crypto';

const opened:Awaited<ReturnType<typeof fixture>>[]=[];
afterEach(async()=>{while(opened.length)await opened.pop()!.close();});

describe('native library insertion',()=>{
 it('libraryInsertUndoAndExport uses one expected-sequence durable receipt and fresh IDs',async()=>{
  const f=await fixture();opened.push(f);await f.pool.query('ALTER TABLE collab_receipts DROP CONSTRAINT collab_receipts_action_check');await f.pool.query("ALTER TABLE collab_receipts ADD CONSTRAINT collab_receipts_action_check CHECK(action IN ('scene:update','scene:command','library:insert'))");
  const store=new Persistence(f.pool,f.authority,{promoteV2:true});const requestId=randomUUID();const content={type:'excalidrawlib',version:2,libraryItems:[{id:'native-item',status:'published',created:1,elements:[rectangle('source-shape')],files:{}}]};
  const bytes=Buffer.from(JSON.stringify({...f.ref,requestId,expectedSequence:'0',content}));const now=Math.floor(Date.now()/1000),payloadHash=createHash('sha256').update(bytes).digest('hex');
  const aclVersion=(await f.pool.query('SELECT acl_version::text FROM workspaces WHERE id=$1',[f.ref.workspaceId])).rows[0].acl_version;const grant=await signed({iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:randomUUID(),sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion,action:'library:insert',requestId,payloadHash,expectedSequence:'0'},grantSecret);
  const first=await store.applyLibraryInsertion(grant,bytes);expect(first.sequence).toBe('1');expect(first.elementIds).toHaveLength(1);expect(first.elementIds[0]).not.toBe('source-shape');
  expect(await store.applyLibraryInsertion(grant,bytes)).toEqual(first);
  const state=await store.read(f.ref,f.principal);expect(state.content.elements.map(element=>element.id)).toEqual(first.elementIds);
  expect((await f.pool.query('SELECT action FROM collab_receipts WHERE request_id=$1',[requestId])).rows[0].action).toBe('library:insert');
 });
});
