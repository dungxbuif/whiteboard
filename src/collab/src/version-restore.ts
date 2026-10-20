import * as Y from 'yjs';
import {validateSceneContent,type SceneContent} from '@collab/contracts';

export type SceneDiff={addedIds:string[];removedIds:string[];changedIds:string[];changedAppStateFields:string[]};

const same=(a:unknown,b:unknown)=>JSON.stringify(a)===JSON.stringify(b);

export function diffVersion(before:SceneContent,after:SceneContent):SceneDiff{
 const left=validateSceneContent(before),right=validateSceneContent(after);
 const a=new Map(left.elements.map(element=>[element.id,element]));
 const b=new Map(right.elements.map(element=>[element.id,element]));
 const addedIds=[...b.keys()].filter(id=>!a.has(id)).sort();
 const removedIds=[...a.keys()].filter(id=>!b.has(id)).sort();
 const changedIds=[...a.keys()].filter(id=>b.has(id)&&!same(a.get(id),b.get(id))).sort();
	const leftApp=left.appState as Record<string,unknown>,rightApp=right.appState as Record<string,unknown>;
	const appKeys=new Set([...Object.keys(leftApp),...Object.keys(rightApp)]);
	const changedAppStateFields=[...appKeys].filter(key=>!same(leftApp[key],rightApp[key])).sort();
 return {addedIds,removedIds,changedIds,changedAppStateFields};
}

function replaceFields(map:Y.Map<unknown>,value:Record<string,unknown>):void{
 for(const key of [...map.keys()])if(!(key in value))map.delete(key);
 for(const [key,field] of Object.entries(value))map.set(key,structuredClone(field));
}

function newFieldMap(value:Record<string,unknown>):Y.Map<unknown>{
 const map=new Y.Map<unknown>();replaceFields(map,value);return map;
}

/** Build one semantic Yjs update without mutating the live document. */
export function restoreUpdate(current:Y.Doc,targetInput:SceneContent):Uint8Array{
 const target=validateSceneContent(targetInput);
 const candidate=new Y.Doc({gc:false});Y.applyUpdate(candidate,Y.encodeStateAsUpdate(current));
 const vector=Y.encodeStateVector(candidate);
 try{
  candidate.transact(()=>{
   const elements=candidate.getMap<Y.Map<unknown>>('elements');
   const wanted=new Set(target.elements.map(element=>element.id));
   for(const [id,map] of elements){
    if(!(map instanceof Y.Map))throw new Error('Element must remain a field map');
    const next=target.elements.find(element=>element.id===id);
    if(!next){map.set('isDeleted',true);continue;}
    if(map.get('type')!==next.type)throw new Error('Restore cannot change an element type');
    replaceFields(map,next as unknown as Record<string,unknown>);
   }
   for(const element of target.elements)if(!elements.has(element.id))elements.set(element.id,newFieldMap(element as unknown as Record<string,unknown>));
   const order=candidate.getArray<string>('order');order.delete(0,order.length);order.insert(0,target.elements.map(element=>element.id));
   const app=candidate.getMap<unknown>('appState');replaceFields(app,target.appState);
   const files=candidate.getMap<Y.Map<unknown>>('files');
   for(const id of [...files.keys()])if(!(id in target.files))files.delete(id);
   for(const [id,file] of Object.entries(target.files)){
    const existing=files.get(id);
    if(existing instanceof Y.Map)replaceFields(existing,file as unknown as Record<string,unknown>);
    else files.set(id,newFieldMap(file as unknown as Record<string,unknown>));
   }
   const presentation=candidate.getArray<string>('presentationOrder');presentation.delete(0,presentation.length);
   if(target.presentationOrder?.length)presentation.insert(0,target.presentationOrder);
   // Ensure every target element was addressed; keeps the tombstone loop honest.
   if(wanted.size!==target.elements.length)throw new Error('Duplicate restore element ID');
  });
  return Y.encodeStateAsUpdate(candidate,vector);
 }finally{candidate.destroy();}
}

export type RestoreVersionInput={versionId:string;requestId:string;expectedSequence:string};
export type RestoreVersionResult={requestId:string;sequence:string;stateVector:string;preRestoreVersionId:string};
export function canonicalRestorePayload(ref:{workspaceId:string;sceneId:string},input:RestoreVersionInput):Uint8Array{
 return new TextEncoder().encode(JSON.stringify({...ref,versionId:input.versionId,requestId:input.requestId,expectedSequence:input.expectedSequence}));
}
