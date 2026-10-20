import {describe,expect,it} from 'vitest';
import {remapLibraryItem,validateLibrary,type IdAllocator} from './library.js';

const base=(id:string,type:string)=>({id,type,x:0,y:0,width:100,height:80,angle:0,strokeColor:'#1e293b',backgroundColor:'transparent',fillStyle:'solid',strokeWidth:2,strokeStyle:'solid',roughness:0,opacity:100,roundness:{type:3},seed:1,version:1,versionNonce:1,index:null,isDeleted:false,groupIds:['group-old'],frameId:null,boundElements:null,updated:0,link:null,locked:false});
const item=()=>({
 id:'item-old',status:'published' as const,created:1,name:'Flow',
 elements:[
  {...base('frame-old','frame'),name:'Frame',boundElements:null},
  {...base('box-old','rectangle'),frameId:'frame-old',boundElements:[{id:'text-old',type:'text'},{id:'arrow-old',type:'arrow'}]},
  {...base('text-old','text'),width:80,height:24,roundness:null,text:'Box',originalText:'Box',fontSize:18,fontFamily:1,textAlign:'center',verticalAlign:'middle',containerId:'box-old',autoResize:false,lineHeight:1.2},
  {...base('image-old','image'),fileId:'file-old',status:'saved',scale:[1,1],crop:null},
  {...base('arrow-old','arrow'),height:0,roundness:{type:2},points:[[0,0],[100,0]],lastCommittedPoint:null,startBinding:{elementId:'box-old',focus:0,gap:1},endBinding:{elementId:'image-old',focus:0,gap:1},startArrowhead:null,endArrowhead:'arrow',elbowed:false},
 ],
 files:{'file-old':{id:'file-old',mimeType:'image/png',created:1,assetId:'asset-old'}},
 presentationOrder:['frame-old'],
});

describe('native reusable libraries',()=>{
 it('libraryRemapsEveryReference',()=>{
  let ordinal=0;const ids:IdAllocator={element:()=>`element-${++ordinal}`,group:()=>`group-${++ordinal}`,file:()=>`file-${++ordinal}`};
  const scene=remapLibraryItem(item(),ids);
  const [frame,box,text,image,arrow]=scene.elements as any[];
  expect(new Set(scene.elements.map(element=>element.id)).size).toBe(5);
  expect(box.frameId).toBe(frame.id);
  expect(text.containerId).toBe(box.id);
  expect(box.boundElements.map((entry:any)=>entry.id)).toEqual([text.id,arrow.id]);
  expect(arrow.startBinding.elementId).toBe(box.id);
  expect(arrow.endBinding.elementId).toBe(image.id);
  expect(image.fileId).not.toBe('file-old');
  expect(scene.files[image.fileId].id).toBe(image.fileId);
  expect(scene.files[image.fileId].assetId).toBe('asset-old');
  expect(scene.presentationOrder).toEqual([frame.id]);
  expect(scene.elements.every(element=>element.groupIds[0]===box.groupIds[0])).toBe(true);
 });

 it('libraryRejectsUnsupportedOrOversize',()=>{
  const native={type:'excalidrawlib',version:2,source:'https://whiteboard.example/libraries/export',libraryItems:[item()]};
  expect(validateLibrary(native).libraryItems).toHaveLength(1);
  const nativeWithoutAssets={...item(),elements:[item().elements[0]]} as any;delete nativeWithoutAssets.files;
  expect(validateLibrary({...native,libraryItems:[nativeWithoutAssets]}).libraryItems[0].files).toEqual({});
  expect(()=>validateLibrary({...native,version:1})).toThrow(/version/i);
  expect(()=>validateLibrary({...native,source:'https://user:secret@whiteboard.example/export'})).toThrow(/source/i);
  expect(()=>validateLibrary({...native,libraryItems:Array.from({length:501},item)})).toThrow(/500/);
  const svg=item();svg.files['file-old'].mimeType='image/svg+xml';
  expect(()=>validateLibrary({...native,libraryItems:[svg]})).toThrow(/raster/i);
  const huge=item();huge.name='x'.repeat(25*1024*1024);
  expect(()=>validateLibrary({...native,libraryItems:[huge]})).toThrow(/25 MiB/i);
 });
});
