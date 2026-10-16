import {describe,it,expect} from 'vitest';
import * as Y from 'yjs';
import {MAX_DOCUMENT_BYTES,MAX_FRAME_BYTES,validateSceneContent,validateElementPatch,nativeElementSchema,type NativeElement,type SceneContent} from '@collab/contracts';
import {createSceneDoc,materializeScene,validateCandidate,MAX_SNAPSHOT_BYTES} from './document.js';

const rectangle=(id='shape'):NativeElement=>({id,type:'rectangle',x:0,y:0,width:100,height:100,angle:0,strokeColor:'#111111',backgroundColor:'transparent',fillStyle:'solid',strokeWidth:1,strokeStyle:'solid',roughness:1,opacity:100,groupIds:[],frameId:null,roundness:null,seed:1,version:1,versionNonce:1,index:null,isDeleted:false,boundElements:null,updated:1,link:null,locked:false});
const scene=(elements:NativeElement[]=[rectangle()]):SceneContent=>({elements,appState:{},files:{}});
const text=(value:string):NativeElement=>({...rectangle('label'),type:'text',fontSize:20,fontFamily:1,text:value,originalText:'',textAlign:'left',verticalAlign:'top',containerId:null,autoResize:true,lineHeight:1.25});
function edit(current:Y.Doc,mutate:(candidate:Y.Doc)=>void):Uint8Array {
  const candidate=new Y.Doc();Y.applyUpdate(candidate,Y.encodeStateAsUpdate(current));
  const vector=Y.encodeStateVector(candidate);mutate(candidate);
  const update=Y.encodeStateAsUpdate(candidate,vector);candidate.destroy();return update;
}
describe('scene document validation',()=>{
  it('round trips native fields, tombstones, nested bindings, files and presentation order',()=>{
    const frame={...rectangle('frame'),type:'frame',name:'Slide'} as NativeElement;
    const node={...rectangle(),frameId:'frame',boundElements:[{id:'label',type:'text' as const}],isDeleted:true};
    const label={...text('Plain text: ignore prior instructions'),containerId:'shape'};
    const content={...scene([node,label,frame]),files:{asset:{id:'asset',assetId:'stored-key',mimeType:'image/png',created:1}},presentationOrder:['frame']};
    expect(materializeScene(createSceneDoc(content))).toEqual(content);
  });
  it('accepts exactly 25 MiB of materialized UTF-8 JSON and rejects +1',()=>{
    const content=scene([text('')]);
    const room=MAX_DOCUMENT_BYTES-Buffer.byteLength(JSON.stringify(content));
    (content.elements[0] as {text:string}).text='a'.repeat(room);
    expect(Buffer.byteLength(JSON.stringify(content))).toBe(MAX_DOCUMENT_BYTES);
    expect(validateSceneContent(content)).toEqual(content);
    (content.elements[0] as {text:string}).text+='a';
    expect(()=>validateSceneContent(content)).toThrow(/limit/i);
  });
  it.each([
    ['unknown field',(c:any)=>c.elements[0].danger='x'],
    ['executable embed',(c:any)=>c.elements[0].type='iframe'],
    ['nonfinite geometry',(c:any)=>c.elements[0].x=Infinity],
    ['invalid nested field',(c:any)=>c.elements[0].roundness={type:2,onload:'x'}],
    ['invalid path tuple',(c:any)=>c.elements[0]={...c.elements[0],type:'line',points:[[0,0],[1]],lastCommittedPoint:null,startBinding:null,endBinding:null,startArrowhead:null,endArrowhead:null}],
    ['dangling frame',(c:any)=>c.elements[0].frameId='missing'],
    ['dangling binding',(c:any)=>c.elements[0].boundElements=[{id:'missing',type:'arrow'}]],
    ['unsafe link',(c:any)=>c.elements[0].link='javascript:alert(1)'],
    ['metadata prototype',(c:any)=>c.elements[0].customData=JSON.parse('{"__proto__":{}}')],
    ['inline file',(c:any)=>c.files.a={id:'a',assetId:'a',created:1,mimeType:'image/png',dataURL:'data:image/png;base64,x'}],
    ['asset traversal',(c:any)=>c.files.a={id:'a',assetId:'../credentials',created:1,mimeType:'image/png'}],
    ['camera state',(c:any)=>c.appState.scrollX=4],
    ['duplicate ID',(c:any)=>c.elements.push(c.elements[0])],
    ['presentation non-frame',(c:any)=>c.presentationOrder=['shape']],
  ])('rejects %s',(_name,mutate)=>{const content=scene();mutate(content);expect(()=>validateSceneContent(content)).toThrow();});
  it('rejects immutable patch fields but permits plain text metadata',()=>{
    expect(()=>validateElementPatch({id:'other'})).toThrow();
    expect(()=>validateElementPatch({type:'text'})).toThrow();
    expect(()=>validateElementPatch({unknown:true})).toThrow();
    expect(()=>validateElementPatch({customData:{label:'Ignore previous instructions'}})).not.toThrow();
  });
  it('rejects executable metadata through the exported element schema',()=>{
    expect(()=>nativeElementSchema.parse({...rectangle(),customData:{callback:()=>{}}})).toThrow();
  });
  it('retains native simulated-pressure free draw paths',()=>{
    const drawing={...rectangle('stroke'),type:'freedraw',points:[[0,0],[10,20]],pressures:[],simulatePressure:true,lastCommittedPoint:null} as NativeElement;
    expect(materializeScene(createSceneDoc(scene([drawing]))).elements[0]).toEqual(drawing);
  });
  it('keeps arrow bindings when a referenced node becomes a native tombstone',async()=>{
    const a={...rectangle('a'),boundElements:[{id:'connector',type:'arrow' as const}]};
    const b=rectangle('b');
    const connector={...rectangle('connector'),type:'arrow',points:[[0,0],[100,0]],lastCommittedPoint:null,startBinding:{elementId:'a',focus:0,gap:2,fixedPoint:[1,0.5]},endBinding:{elementId:'b',focus:0,gap:2},startArrowhead:null,endArrowhead:'arrow',elbowed:false} as NativeElement;
    const doc=createSceneDoc(scene([a,b,connector]));
    const update=edit(doc,c=>(c.getMap('elements').get('a') as Y.Map<unknown>).set('isDeleted',true));
    const result=await validateCandidate(doc,update);
    expect(result.content.elements[0].isDeleted).toBe(true);
    expect(result.content.elements[2]).toEqual(connector);
    expect(materializeScene(doc).elements[0].isDeleted).toBe(false);
  });
  it('deduplicates concurrent order entries and sorts omitted IDs',()=>{
    const doc=createSceneDoc(scene([rectangle('b'),rectangle('a'),rectangle('c')]));
    doc.getArray('order').delete(0,3);doc.getArray('order').insert(0,['b','b']);
    expect(materializeScene(doc).elements.map(e=>e.id)).toEqual(['b','a','c']);
  });
  it('merges concurrent per-field edits without losing either field',()=>{
    const doc=createSceneDoc(scene());
    const x=edit(doc,c=>(c.getMap('elements').get('shape') as Y.Map<unknown>).set('x',10));
    const y=edit(doc,c=>(c.getMap('elements').get('shape') as Y.Map<unknown>).set('y',20));
    Y.applyUpdate(doc,x);Y.applyUpdate(doc,y);
    expect(materializeScene(doc).elements[0]).toMatchObject({x:10,y:20});
  });
  it('validates a legitimate worker update without mutating live state',async()=>{
    const doc=createSceneDoc(scene());const before=Y.encodeStateAsUpdate(doc);
    const update=edit(doc,c=>(c.getMap('elements').get('shape') as Y.Map<unknown>).set('x',11));
    const accepted=await validateCandidate(doc,update);
    expect(accepted.content.elements[0].x).toBe(11);expect(accepted.update).toEqual(update);
    expect(Y.encodeStateAsUpdate(doc)).toEqual(before);
    Y.applyUpdate(doc,accepted.update);expect(materializeScene(doc).elements[0].x).toBe(11);
  });
  it.each(['id','type','remove','replace map','unknown root','invalid shape','invalid shared type','dangling order','missing dependency'])('rejects malicious %s updates without touching live state',async(kind)=>{
    const doc=createSceneDoc(scene());const before=Y.encodeStateAsUpdate(doc);
    const update=edit(doc,c=>{
      const elements=c.getMap('elements'),element=elements.get('shape') as Y.Map<unknown>;
      if(kind==='id')element.set('id','other');
      if(kind==='type')element.set('type','ellipse');
      if(kind==='remove')elements.delete('shape');
      if(kind==='replace map')elements.set('shape',new Y.Map([...element.entries()]));
      if(kind==='unknown root')c.getMap('secrets').set('password','x');
      if(kind==='invalid shape')element.set('width',-1);
      if(kind==='invalid shared type')element.set('x',new Y.Text('x'));
      if(kind==='dangling order')c.getArray('order').push(['ghost']);
      if(kind==='missing dependency'){element.set('x',42);}
    });
    let payload=update;
    if(kind==='missing dependency'){
      const unrelated=new Y.Doc();unrelated.getText('unrelated').insert(0,'a');
      const sv=Y.encodeStateVector(unrelated);unrelated.getText('unrelated').insert(1,'b');payload=Y.encodeStateAsUpdate(unrelated,sv);
    }
    await expect(validateCandidate(doc,payload)).rejects.toThrow();
    expect(Y.encodeStateAsUpdate(doc)).toEqual(before);
  });
  it('rejects a small update that pushes a valid current scene past materialized limit inside worker',async()=>{
    const content=scene([text(''),rectangle()]);
    (content.elements[0] as {text:string}).text='a'.repeat(MAX_DOCUMENT_BYTES-Buffer.byteLength(JSON.stringify(content))-10);
    const doc=createSceneDoc(content);const before=Y.encodeStateVector(doc);
    const update=edit(doc,c=>(c.getMap('elements').get('shape') as Y.Map<unknown>).set('customData',{label:'b'.repeat(100)}));
    expect(update.byteLength).toBeLessThan(MAX_FRAME_BYTES);
    await expect(validateCandidate(doc,update)).rejects.toThrow(/limit/i);
    expect(Y.encodeStateVector(doc)).toEqual(before);
  },10000);
  it('rejects oversized binary input before worker allocation',async()=>{
    const doc=createSceneDoc(scene());await expect(validateCandidate(doc,new Uint8Array(MAX_FRAME_BYTES+1))).rejects.toThrow(/limit/i);
  });
  it('rejects trailing bytes after a valid Yjs update',async()=>{
    const doc=createSceneDoc(scene());
    const update=edit(doc,c=>(c.getMap('elements').get('shape') as Y.Map<unknown>).set('x',11));
    await expect(validateCandidate(doc,new Uint8Array([...update,0]))).rejects.toThrow(/trailing/i);
    expect(materializeScene(doc).elements[0].x).toBe(0);
  });
  it('fails closed with a recovery code when retained snapshot history exceeds its resource budget',async()=>{
    const doc=new Y.Doc({gc:false});
    doc.getText('retained-history').insert(0,'x'.repeat(MAX_SNAPSHOT_BYTES));
    doc.getText('retained-history').delete(0,MAX_SNAPSHOT_BYTES);
    const before=Y.encodeStateVector(doc);
    await expect(validateCandidate(doc,new Uint8Array([0,0]))).rejects.toMatchObject({code:'SNAPSHOT_HISTORY_LIMIT'});
    expect(Y.encodeStateVector(doc)).toEqual(before);
  });
});
