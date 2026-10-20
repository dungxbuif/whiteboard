import {describe,expect,it} from 'vitest';
import * as Y from 'yjs';
import type {NativeElement,SceneContent} from '@collab/contracts';
import {createSceneDoc,materializeScene,validateCandidate} from './document.js';
import {diffVersion,restoreUpdate} from './version-restore.js';

const rect=(id:string,x=0):NativeElement=>({id,type:'rectangle',x,y:0,width:100,height:100,angle:0,strokeColor:'#111111',backgroundColor:'transparent',fillStyle:'solid',strokeWidth:1,strokeStyle:'solid',roughness:1,opacity:100,groupIds:[],frameId:null,roundness:null,seed:1,version:1,versionNonce:1,index:null,isDeleted:false,boundElements:null,updated:1,link:null,locked:false});
const scene=(elements:NativeElement[],extra:Partial<SceneContent>={}):SceneContent=>({elements,appState:{viewBackgroundColor:'#ffffff'},files:{},...extra});

describe('version restore semantics',()=>{
 it('restoreRetainsDocumentAndElementMaps and tombstones later additions',async()=>{
  const current=createSceneDoc(scene([rect('kept',20),rect('later',50)]));
  const kept=current.getMap<Y.Map<unknown>>('elements').get('kept');
  const accepted=await validateCandidate(current,restoreUpdate(current,scene([rect('kept',1)])));
  Y.applyUpdate(current,accepted.update);
  expect(current.getMap('elements').get('kept')).toBe(kept);
  expect(materializeScene(current).elements).toMatchObject([{id:'kept',x:1,isDeleted:false},{id:'later',isDeleted:true}]);
  current.destroy();
 });

 it('restoreRestoresBindingsOrderAndImages and clears omitted fields',async()=>{
  const frame={...rect('frame'),type:'frame',name:'Slide'} as NativeElement;
  const node={...rect('node'),frameId:'frame',customData:{temporary:true}};
  const targetNode={...rect('node'),frameId:'frame',boundElements:[{id:'arrow',type:'arrow' as const}]};
  const arrow={...rect('arrow'),type:'arrow',points:[[0,0],[100,0]],lastCommittedPoint:null,startBinding:{elementId:'node',focus:0,gap:2},endBinding:null,startArrowhead:null,endArrowhead:'arrow',elbowed:false} as NativeElement;
  const current=createSceneDoc(scene([node,frame]));
  const target=scene([frame,targetNode,arrow],{files:{file:{id:'file',assetId:'00000000-0000-4000-8000-000000000001',mimeType:'image/png',created:1}},presentationOrder:['frame']});
  const accepted=await validateCandidate(current,restoreUpdate(current,target));Y.applyUpdate(current,accepted.update);
  expect(materializeScene(current)).toEqual(target);
  expect(materializeScene(current).elements[1]).not.toHaveProperty('customData');
  current.destroy();
 });

 it('rejects immutable type changes and reports semantic diff',()=>{
  const before=scene([rect('a'),rect('removed')],{appState:{viewBackgroundColor:'#fff'}});
  const after=scene([{...rect('a'),x:4},rect('added')],{appState:{viewBackgroundColor:'#000'}});
  expect(diffVersion(before,after)).toEqual({addedIds:['added'],removedIds:['removed'],changedIds:['a'],changedAppStateFields:['viewBackgroundColor']});
  const current=createSceneDoc(before);
  expect(()=>restoreUpdate(current,scene([{...rect('a'),type:'ellipse'} as NativeElement]))).toThrow(/type/i);
  current.destroy();
 });

 it('retains a tombstoned image file so an offline edit can revive it',async()=>{
  const file={id:'file-later',assetId:'00000000-0000-4000-8000-000000000001',mimeType:'image/png',created:1};
  const image={...rect('later-image'),type:'image',fileId:'file-later',status:'saved',scale:[1,1],crop:null} as unknown as NativeElement;
  const current=createSceneDoc(scene([rect('kept'),image],{files:{'file-later':file}}));
  const offline=new Y.Doc({gc:false});Y.applyUpdate(offline,Y.encodeStateAsUpdate(current));const offlineVector=Y.encodeStateVector(offline);
  const offlineImage=offline.getMap<Y.Map<unknown>>('elements').get('later-image')!;offlineImage.set('x',88);offlineImage.set('isDeleted',false);
  const offlineUpdate=Y.encodeStateAsUpdate(offline,offlineVector);
  const restored=await validateCandidate(current,restoreUpdate(current,scene([rect('kept')])));Y.applyUpdate(current,restored.update);
  expect(materializeScene(current).files).toHaveProperty('file-later');
  const replayed=await validateCandidate(current,offlineUpdate);Y.applyUpdate(current,replayed.update);
  expect(materializeScene(current).elements.find(element=>element.id==='later-image')).toMatchObject({x:88,isDeleted:false,fileId:'file-later'});
  offline.destroy();current.destroy();
 });
});
