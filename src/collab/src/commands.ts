import * as Y from 'yjs';
import {assertSerializable,validateElementPatch,type SceneMutation,type SceneRef,type MutationReceipt} from '@collab/contracts';
import {CollabError,requestID,revision} from './auth.js';
import type {Persistence} from './persistence.js';
export function parseMutation(bytes:Uint8Array):SceneMutation {
 try{
  const mutation=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes));assertSerializable(mutation);
  if(!mutation||Object.keys(mutation).some(key=>!['workspaceId','sceneId','requestId','expectedSequence','commands'].includes(key))||!Array.isArray(mutation.commands)||mutation.commands.length<1||mutation.commands.length>1000||!revision(mutation.expectedSequence))throw new Error();
  requestID(mutation.requestId);return mutation;
 }catch{throw new CollabError('INVALID_UPDATE','Invalid command envelope',400);}
}
export function commandUpdate(current:Y.Doc,mutation:SceneMutation):Uint8Array {
 const doc=new Y.Doc({gc:false});Y.applyUpdate(doc,Y.encodeStateAsUpdate(current));const vector=Y.encodeStateVector(doc);
 try{doc.transact(()=>{
  const elements=doc.getMap<Y.Map<unknown>>('elements');
  for(const command of mutation.commands){
   if(command.op==='add'){
    if(Object.keys(command).some(k=>!['op','elements'].includes(k))||!Array.isArray(command.elements))throw new Error();
    for(const element of command.elements){if(elements.has(element.id))throw new Error('Element already exists');const fields=new Y.Map<unknown>();for(const [key,value]of Object.entries(element))fields.set(key,value);elements.set(element.id,fields);doc.getArray('order').push([element.id]);}
   }else if(command.op==='patch'){
    if(Object.keys(command).some(k=>!['op','patches'].includes(k))||!Array.isArray(command.patches))throw new Error();
    for(const patch of command.patches){if(Object.keys(patch).some(k=>!['id','fields'].includes(k)))throw new Error();validateElementPatch(patch.fields);const element=elements.get(patch.id);if(!element)throw new Error('Unknown element');for(const [key,value]of Object.entries(patch.fields))element.set(key,value);}
   }else if(command.op==='delete'){
    if(Object.keys(command).some(k=>!['op','ids'].includes(k))||!Array.isArray(command.ids))throw new Error();
    for(const id of command.ids){const element=elements.get(id);if(!element)throw new Error('Unknown element');element.set('isDeleted',true);}
   }else throw new Error('Unknown command');
  }
 });return Y.encodeStateAsUpdate(doc,vector);}catch(error){throw new CollabError('INVALID_UPDATE',error instanceof Error?error.message:'Invalid command',400);}finally{doc.destroy();}
}
/** Preserve original bytes; callers must not parse and reserialize signed commands. */
export function applyCommands(store:Persistence,grant:string,mutation:Uint8Array):Promise<MutationReceipt>{return store.applyCommands(grant,mutation);}
