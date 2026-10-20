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
const mapRoots=new Set(['elements','appState','files']);
type ContainerCompartments={_start:Y.Item|null;_map:Map<string,Y.Item>};
function assertMapShape(map:ContainerCompartments):void {
  // Public map iterators omit the sequence compartment, including deleted
  // structs. Yjs persists both compartments regardless of the JS wrapper type.
  if(map._start!==null)throw new Error('Map contains forbidden sequence content');
}
function assertArrayShape(array:ContainerCompartments):void {
  if(array._map.size!==0)throw new Error('Array contains forbidden keyed content');
  for(let item=array._start;item!==null;item=item.right){
    if(item.content instanceof Y.ContentDeleted)continue;
    if(!(item.content instanceof Y.ContentAny)||item.content.getContent().some(value=>typeof value!=='string'))throw new Error('Array contains forbidden sequence content');
  }
}
function* retainedMapValues<T>(map:Y.Map<T>):Generator<unknown> {
  for(const latest of map._map.values())for(let item:Y.Item|null=latest;item!==null;item=item.left)yield* item.content.getContent();
}
function assertFieldMaps<T>(root:Y.Map<T>):void {
  // Check historical child containers too: a malicious map added and deleted in
  // the same update must not hide its sequence structs from live iteration.
  for(const value of retainedMapValues(root)){
    if(!(value instanceof Y.Map))throw new Error('Nested field container must be a map');
    assertMapShape(value);
    assertJsonMapFields(value);
  }
}
function assertJsonMapFields<T>(map:Y.Map<T>):void {
  for(const field of retainedMapValues(map)){
    try{assertSerializable(field);}catch{throw new Error('Nested map fields must be plain JSON');}
  }
}
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
  assertMapShape(map);
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
  for(const [key,type] of doc.share){
    if(!roots.includes(key))throw new Error('Unknown document root');
    if(mapRoots.has(key))assertMapShape(type);else assertArrayShape(type);
  }
  if(doc.store.pendingStructs||doc.store.pendingDs)throw new Error('Unresolved update dependencies');
  const elements=doc.getMap<Y.Map<unknown>>('elements');
  assertFieldMaps(elements);
  assertFieldMaps(doc.getMap('files'));
  assertJsonMapFields(doc.getMap('appState'));
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
async function validateCandidateLimit(current:Y.Doc,update:Uint8Array,maxUpdateBytes:number):Promise<ValidatedUpdate> {
  if(!(update instanceof Uint8Array)||update.byteLength>maxUpdateBytes)throw new Error('Update byte limit exceeded');
  const snapshot=Y.encodeStateAsUpdate(current);
  if(snapshot.byteLength>MAX_SNAPSHOT_BYTES)throw new SnapshotHistoryLimitError();
  return runValidationJob({snapshot,update:update.slice()});
}
export const validateCandidate=(current:Y.Doc,update:Uint8Array)=>validateCandidateLimit(current,update,MAX_FRAME_BYTES);
// Restore is an authenticated internal writer mutation. Its canonical content is
// capped at 25 MiB, but its Yjs delta can legitimately exceed a socket frame.
export const validateRestoreCandidate=(current:Y.Doc,update:Uint8Array)=>validateCandidateLimit(current,update,MAX_SNAPSHOT_BYTES);
