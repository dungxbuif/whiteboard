import {createHash} from 'node:crypto';
import * as Y from 'yjs';
import {assertSerializable,type MutationReceipt,type SceneContent,type SceneRef} from '@collab/contracts';
import {remapLibraryItem,validateLibrary,type IdAllocator} from '@collab/diagram-kit';
import {CollabError,requestID,revision} from './auth.js';

export type LibraryInsertion=SceneRef&{requestId:string;expectedSequence:string;nativeItemId?:string;content:unknown};
export type LibraryInsertionResult=MutationReceipt&{elementIds:string[]};

export function parseLibraryInsertion(bytes:Uint8Array):LibraryInsertion{
 try{
  const value=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes));assertSerializable(value);
  if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!['workspaceId','sceneId','requestId','expectedSequence','nativeItemId','content'].includes(key))||typeof value.workspaceId!=='string'||typeof value.sceneId!=='string'||!revision(value.expectedSequence)||value.nativeItemId!==undefined&&typeof value.nativeItemId!=='string')throw new Error();
  requestID(value.requestId);validateLibrary(value.content);return value as LibraryInsertion;
 }catch(error){if(error instanceof CollabError)throw error;throw new CollabError('INVALID_UPDATE','Invalid native library insertion',400);}
}
function fresh(requestId:string,kind:string,oldId:string):string{return `l_${createHash('sha256').update(`${requestId}:${kind}:${oldId}`).digest('hex').slice(0,30)}`;}
export function materializeLibraryInsertion(input:LibraryInsertion):SceneContent{
 const library=validateLibrary(input.content),item=input.nativeItemId?library.libraryItems.find(entry=>entry.id===input.nativeItemId):library.libraryItems[0];if(!item)throw new CollabError('INVALID_UPDATE','Unknown library item',400);
 const ids:IdAllocator={element:id=>fresh(input.requestId,'element',id),group:id=>fresh(input.requestId,'group',id),file:id=>fresh(input.requestId,'file',id)};
 try{return remapLibraryItem(item,ids);}catch(error){throw new CollabError('INVALID_UPDATE',error instanceof Error?error.message:'Invalid library item',400);}
}
function fieldMap(value:Record<string,unknown>):Y.Map<unknown>{const map=new Y.Map<unknown>();for(const [key,field]of Object.entries(value))map.set(key,structuredClone(field));return map;}
export function libraryInsertionUpdate(current:Y.Doc,content:SceneContent):Uint8Array{
 const doc=new Y.Doc({gc:false});Y.applyUpdate(doc,Y.encodeStateAsUpdate(current));const vector=Y.encodeStateVector(doc);
 try{doc.transact(()=>{const elements=doc.getMap<Y.Map<unknown>>('elements'),files=doc.getMap<Y.Map<unknown>>('files');for(const element of content.elements){if(elements.has(element.id))throw new Error('Element already exists');elements.set(element.id,fieldMap(element as unknown as Record<string,unknown>));doc.getArray('order').push([element.id]);}for(const [id,file]of Object.entries(content.files)){if(files.has(id))throw new Error('File already exists');files.set(id,fieldMap(file as unknown as Record<string,unknown>));}if(content.presentationOrder?.length)doc.getArray('presentationOrder').push(content.presentationOrder);});return Y.encodeStateAsUpdate(doc,vector);}catch(error){throw new CollabError('INVALID_UPDATE',error instanceof Error?error.message:'Invalid library insertion',400);}finally{doc.destroy();}
}
