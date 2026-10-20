import { describe, it, expect } from 'vitest';
import * as Y from 'yjs';
import { ExcalidrawAdapter, LOCAL_ORIGIN } from './excalidraw-adapter';
import { rectangle, scene } from './fixtures';
describe('collab adapter', () => {
  it('remote updates render without echo and preserve native binding fields', () => {
    const a = new ExcalidrawAdapter(); const b = new ExcalidrawAdapter();
    const node = { ...rectangle(), boundElements: [{ id: 'label', type: 'text' as const }] };
    const label = { ...rectangle('label'), type: 'text' as const, fontSize: 20, fontFamily: 1, text: 'Hello', originalText: 'Hello', textAlign: 'left' as const, verticalAlign: 'top' as const, containerId: 'shape', autoResize: true, lineHeight: 1.25 };
    a.applyLocalScene(scene([node, label]));
    let echoes = 0; b.doc.on('update', (_bytes, origin) => { if (origin === LOCAL_ORIGIN) echoes++; });
    b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    b.applyLocalScene(b.content());
    expect(b.content().elements).toEqual([node, label]); expect(echoes).toBe(0);
  });
  it('concurrent field edits, tombstones and reorder converge without losing independent fields', () => {
    const a = new ExcalidrawAdapter(); a.applyLocalScene(scene([rectangle('one'), rectangle('two')]));
    const b = new ExcalidrawAdapter(); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    const left = a.content(); left.elements[0].x = 75; left.elements.reverse(); a.applyLocalScene(left);
    const right = b.content(); right.elements[0].y = 44; right.elements[1].isDeleted = true; b.applyLocalScene(right);
    a.applyRemote(Y.encodeStateAsUpdate(b.doc)); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    expect(a.content()).toEqual(b.content());
    expect(a.content().elements.find(e => e.id === 'one')).toMatchObject({ x: 75, y: 44 });
    expect(a.content().elements.find(e => e.id === 'two')?.isDeleted).toBe(true);
  });
  it('causally reapplies offline image intent after restore regardless of Yjs client tie-break',()=>{
    const image={...rectangle('image'),type:'image' as const,fileId:'file',status:'saved' as const,scale:[1,1] as [number,number],crop:null};
    const base=scene([image]);base.files={file:{id:'file',assetId:'00000000-0000-4000-8000-000000000001',mimeType:'image/png',created:1}};
    for(let run=0;run<32;run++){
      const seed=new ExcalidrawAdapter();seed.applyLocalScene(base);const offline=new ExcalidrawAdapter(),restored=new ExcalidrawAdapter();offline.applyRemote(Y.encodeStateAsUpdate(seed.doc));restored.applyRemote(Y.encodeStateAsUpdate(seed.doc));
      const desired=offline.content();desired.elements[0].x=88;desired.elements[0].isDeleted=false;offline.applyLocalScene(desired);
      restored.doc.clientID=0xfffffffe;const vector=Y.encodeStateVector(restored.doc);restored.doc.getMap<Y.Map<unknown>>('elements').get('image')!.set('isDeleted',true);const restoreDelta=Y.encodeStateAsUpdate(restored.doc,vector);
      const authoritative=new ExcalidrawAdapter();authoritative.doc.clientID=1;authoritative.applyRemote(Y.encodeStateAsUpdate(seed.doc));authoritative.applyRemote(restoreDelta);const causal=[];for(let x=81;x<=88;x++){const step=structuredClone(desired);step.elements[0].x=x;causal.push(authoritative.reapplyIntent(base,step));}const server=new ExcalidrawAdapter();server.applyRemote(Y.encodeStateAsUpdate(seed.doc));server.applyRemote(restoreDelta);server.applyRemote(Y.mergeUpdates(causal));
      expect(server.content().elements[0]).toMatchObject({x:88,isDeleted:false,fileId:'file'});expect(server.content().files.file).toBeDefined();
      for(const adapter of [seed,offline,restored,authoritative,server])adapter.doc.destroy();
    }
  });
});
