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
});
