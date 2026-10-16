import * as Y from 'yjs';
import {MAX_FRAME_BYTES,validateSceneContent,assertSerializable,type SceneContent} from '@collab/contracts';
import {runValidationJob} from './validation-job.js';
export type ValidatedUpdate = {update:Uint8Array;content:SceneContent;stateVector:Uint8Array};
export const MAX_SNAPSHOT_BYTES = 64 * 1024 * 1024;
export class SnapshotHistoryLimitError extends Error {
  readonly code='SNAPSHOT_HISTORY_LIMIT';
  constructor(){super('Snapshot history byte limit exceeded; controlled recovery required without discarding queued or acknowledged updates');this.name='SnapshotHistoryLimitError';}
}
const roots=['elements','order','appState','files','presentationOrder'];
function fieldMap(value:Record<string,unknown>):Y.Map<unknown> {
  const map=new Y.Map<unknown>();for(const [key,field] of Object.entries(value))map.set(key,structuredClone(field));return map;
}
export function createSceneDoc(input:SceneContent):Y.Doc {
  const content=validateSceneContent(input),doc=new Y.Doc({gc:false});
  doc.transact(()=>{
    doc.getMap('appState');doc.getMap('files');doc.getArray('presentationOrder');
    const elements=doc.getMap<Y.Map<unknown>>('elements');
    for(const element of content.elements)elements.set(element.id,fieldMap(element));
    doc.getArray<string>('order').push(content.elements.map(element=>element.id));
    for(const [key,value] of Object.entries(content.appState))doc.getMap('appState').set(key,value);
    for(const [key,value] of Object.entries(content.files))doc.getMap('files').set(key,fieldMap(value));
    if(content.presentationOrder?.length)doc.getArray('presentationOrder').push(content.presentationOrder);
  });
  return doc;
}
function plainMap(map:Y.Map<unknown>):Record<string,unknown> {
  const value=Object.fromEntries(map.entries());assertSerializable(value);return value;
}
function orderedIDs(order:Y.Array<unknown>,known:Set<string>):string[] {
  const result:string[]=[],seen=new Set<string>();
  for(const id of order.toArray()){
    if(typeof id!=='string'||!known.has(id))throw new Error('Invalid order reference');
    if(!seen.has(id)){result.push(id);seen.add(id);}
  }
  return result;
}
export function materializeScene(doc:Y.Doc):SceneContent {
  for(const key of doc.share.keys())if(!roots.includes(key))throw new Error('Unknown document root');
  if(doc.store.pendingStructs||doc.store.pendingDs)throw new Error('Unresolved update dependencies');
  const elements=doc.getMap<Y.Map<unknown>>('elements');
  const plainElements=new Map<string,Record<string,unknown>>();
  for(const [id,element] of elements){
    if(!(element instanceof Y.Map))throw new Error('Element must be a field map');
    const value=plainMap(element);if(value.id!==id)throw new Error('Element ID mismatch');plainElements.set(id,value);
  }
  const order=orderedIDs(doc.getArray('order'),new Set(plainElements.keys()));
  const ordered=new Set(order);
  const omitted=[...plainElements.keys()].filter(id=>!ordered.has(id)).sort();
  const files:Record<string,unknown>=Object.create(null);
  for(const [id,file] of doc.getMap('files')){
    if(!(file instanceof Y.Map))throw new Error('File must be a field map');files[id]=plainMap(file);
  }
  const presentationOrder=orderedIDs(doc.getArray('presentationOrder'),new Set(plainElements.keys()));
  return validateSceneContent({elements:[...order,...omitted].map(id=>plainElements.get(id)),appState:plainMap(doc.getMap('appState')),files,...(presentationOrder.length?{presentationOrder}:{})});
}
export async function validateCandidate(current:Y.Doc,update:Uint8Array):Promise<ValidatedUpdate> {
  if(!(update instanceof Uint8Array)||update.byteLength>MAX_FRAME_BYTES)throw new Error('Update byte limit exceeded');
  const snapshot=Y.encodeStateAsUpdate(current);
  if(snapshot.byteLength>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  return runValidationJob({snapshot,update:update.slice()});
}
