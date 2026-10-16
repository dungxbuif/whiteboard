import type { ExcalidrawElement, ExcalidrawElbowArrowElement } from '@excalidraw/excalidraw/element/types';
import { z } from 'zod';
export type JsonValue = null | boolean | number | string | JsonValue[] | {[key:string]:JsonValue};
type Serializable<T> = T extends number ? number : T extends string ? T extends {_brand:string} ? string : T : T extends boolean | null | undefined ? T : { -readonly [K in keyof T]: Serializable<T[K]> };
export type NativePoint = [number,number];
// Upstream LocalPoint imports an unpublished @excalidraw/math alias. Override
// geometry explicitly so that consumers never inherit an unresolved `any`.
type NativeFields<T> = T extends unknown ? Omit<T,'customData'|'points'|'lastCommittedPoint'|'fixedSegments'> & {customData?:Record<string,JsonValue>} & (T extends {points:unknown} ? {points:NativePoint[];lastCommittedPoint:NativePoint|null} : {}) & (T extends {fixedSegments:unknown} ? {fixedSegments:{start:NativePoint;end:NativePoint;index:number}[]|null} : {}) : never;
export type NativeElement = NativeFields<Serializable<Exclude<ExcalidrawElement, {type:'selection'|'iframe'|'embeddable'}> | ExcalidrawElbowArrowElement>>;
export type AssetReference = {id:string; mimeType:string; created:number; assetId:string; lastRetrieved?:number; version?:number};
export type SceneAppState = {viewBackgroundColor?:string;gridSize?:number|null;gridStep?:number;gridModeEnabled?:boolean;theme?:'light'|'dark'};
export type SceneContent = {elements:NativeElement[];appState:SceneAppState;files:Record<string,AssetReference>;presentationOrder?:string[]};
export type SceneRef = {workspaceId:string;sceneId:string};
export type Revision = string;
export type MutationReceipt = {requestId:string;sequence:Revision;stateVector:string};
export type ElementCommand = {op:'add';elements:NativeElement[]} | {op:'patch';patches:{id:string;fields:Record<string,unknown>}[]} | {op:'delete';ids:string[]};
export type SceneMutation = SceneRef & {requestId:string;expectedSequence:Revision;commands:ElementCommand[]};
export const MAX_DOCUMENT_BYTES = 25 * 1024 * 1024;
const id = z.string().regex(/^[A-Za-z0-9_-]{1,256}$/);
const finite = z.number().finite();
const nonnegative = finite.min(0);
const integer = nonnegative.int().max(Number.MAX_SAFE_INTEGER);
const color = z.string().max(128).regex(/^(?:#[0-9a-fA-F]{3,8}|[a-zA-Z]+|(?:rgb|rgba|hsl|hsla)\([0-9.,%\s+-]+\))$/);
const point = z.tuple([finite,finite]);
const points = z.array(point);
const arrowhead = z.enum(['arrow','bar','dot','circle','circle_outline','triangle','triangle_outline','diamond','diamond_outline','crowfoot_one','crowfoot_many','crowfoot_one_or_many']).nullable();
const binding = z.strictObject({elementId:id,focus:finite.min(-1).max(1),gap:nonnegative,fixedPoint:point.optional()}).nullable();
const safeLink = z.string().max(4096).refine(value=>{
  if (/[\s\\\u0000-\u001f]/.test(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) {
    try {return !decodeURIComponent(value).split(/[/?#]/).includes('..');} catch {return false;}
  }
  try {const url=new URL(value);return ['https:','http:','mailto:'].includes(url.protocol)&&!url.username&&!url.password;} catch {return false;}
},'Unsafe link').nullable();

// Inspect before recursive schema parsing/stringification. No accessors, cycles,
// prototype keys, non-JSON values or deeply nested metadata can cross this boundary.
export function assertSerializable(input:unknown):void {
  const active = new Set<object>();
  const visit=(value:unknown,depth:number):void=>{
    if(depth>32)throw new Error('JSON nesting limit exceeded');
    if(value===null||typeof value==='string'||typeof value==='boolean')return;
    if(typeof value==='number'&&Number.isFinite(value))return;
    if(typeof value!=='object'||value===undefined)throw new Error('Non-JSON value');
    if(active.has(value))throw new Error('Cyclic JSON value');
    if(!Array.isArray(value)&&Object.getPrototypeOf(value)!==Object.prototype&&Object.getPrototypeOf(value)!==null)throw new Error('Non-JSON object');
    active.add(value);
    for(const [key,descriptor] of Object.entries(Object.getOwnPropertyDescriptors(value))){
      if(['__proto__','prototype','constructor'].includes(key)||descriptor.get||descriptor.set)throw new Error('Unsafe JSON field');
      if(Array.isArray(value)&&key==='length')continue;
      visit(descriptor.value,depth+1);
    }
    active.delete(value);
  };
  visit(input,0);
}
const metadata=z.record(z.string(),z.unknown()).superRefine((value,ctx)=>{
  try {assertSerializable(value);}catch{ctx.addIssue({code:'custom',message:'Metadata must be bounded serializable JSON'});return;}
  const visit=(entry:unknown):void=>{
    if(entry&&typeof entry==='object')for(const [key,item] of Object.entries(entry)){
      if(/^(?:html|srcdoc|script|dataurl|password|authorization|accesstoken|refreshtoken|on[a-z]+)$/i.test(key))ctx.addIssue({code:'custom',message:'Unsafe metadata field'});
      visit(item);
    }
  };visit(value);
});
const base = {
  id,x:finite,y:finite,width:nonnegative,height:nonnegative,angle:finite,
  strokeColor:color,backgroundColor:color,fillStyle:z.enum(['hachure','cross-hatch','solid','zigzag']),
  strokeWidth:nonnegative,strokeStyle:z.enum(['solid','dashed','dotted']),roughness:nonnegative,opacity:finite.min(0).max(100),
  roundness:z.strictObject({type:z.union([z.literal(1),z.literal(2),z.literal(3)]),value:finite.optional()}).nullable(),
  seed:integer,version:integer,versionNonce:integer,index:z.string().regex(/^[A-Za-z0-9]+$/).nullable(),
  isDeleted:z.boolean(),groupIds:z.array(id),frameId:id.nullable(),
  boundElements:z.array(z.strictObject({id,type:z.enum(['arrow','text'])})).nullable(),updated:integer,link:safeLink,locked:z.boolean(),customData:metadata.optional(),
};
const textFields = {fontSize:finite.positive(),fontFamily:integer,text:z.string(),textAlign:z.enum(['left','center','right']),verticalAlign:z.enum(['top','middle','bottom']),containerId:id.nullable(),originalText:z.string(),autoResize:z.boolean(),lineHeight:finite.positive()};
const linearFields = {points:points.min(2),lastCommittedPoint:point.nullable(),startBinding:binding,endBinding:binding,startArrowhead:arrowhead,endArrowhead:arrowhead};
const elbowFields = {elbowed:z.boolean().optional(),fixedSegments:z.array(z.strictObject({start:point,end:point,index:integer})).nullable().optional(),startIsSpecial:z.boolean().nullable().optional(),endIsSpecial:z.boolean().nullable().optional()};
const freeFields = {points:points.min(1),pressures:z.array(finite.min(0).max(1)),simulatePressure:z.boolean(),lastCommittedPoint:point.nullable()};
const imageFields = {fileId:id.nullable(),status:z.enum(['pending','saved','error']),scale:z.tuple([z.union([z.literal(-1),z.literal(1)]),z.union([z.literal(-1),z.literal(1)])]),crop:z.strictObject({x:nonnegative,y:nonnegative,width:nonnegative,height:nonnegative,naturalWidth:finite.positive(),naturalHeight:finite.positive()}).nullable()};
export const nativeElementSchema = z.discriminatedUnion('type',[
  z.strictObject({...base,type:z.literal('rectangle')}),z.strictObject({...base,type:z.literal('diamond')}),z.strictObject({...base,type:z.literal('ellipse')}),
  z.strictObject({...base,type:z.literal('text'),...textFields}),
  z.strictObject({...base,type:z.literal('line'),...linearFields}),z.strictObject({...base,type:z.literal('arrow'),...linearFields,...elbowFields}),
  z.strictObject({...base,type:z.literal('freedraw'),...freeFields}),z.strictObject({...base,type:z.literal('image'),...imageFields}),
  z.strictObject({...base,type:z.literal('frame'),name:z.string().nullable()}),z.strictObject({...base,type:z.literal('magicframe'),name:z.string().nullable()}),
]);
export const assetReferenceSchema = z.strictObject({id,mimeType:z.enum(['image/png','image/jpeg','image/svg+xml','image/gif','image/webp','image/avif']),created:integer,assetId:id,lastRetrieved:integer.optional(),version:integer.optional()});
export const sceneAppStateSchema = z.strictObject({viewBackgroundColor:color.optional(),gridSize:finite.positive().nullable().optional(),gridStep:finite.positive().optional(),gridModeEnabled:z.boolean().optional(),theme:z.enum(['light','dark']).optional()});
export const sceneContentSchema = z.strictObject({elements:z.array(nativeElementSchema),appState:sceneAppStateSchema,files:z.record(id,assetReferenceSchema),presentationOrder:z.array(id).optional()});
const patchSchema=z.strictObject({...base,...textFields,...linearFields,...elbowFields,...freeFields,...imageFields,name:z.string().nullable()}).omit({id:true}).partial();
export function validateElementPatch(fields:Record<string,unknown>):void {
  assertSerializable(fields);patchSchema.parse(fields);
}
export function validateSceneContent(input:unknown):SceneContent {
  assertSerializable(input);
  const content=sceneContentSchema.parse(input) as SceneContent;
  const elements=new Map<string,NativeElement>();
  for(const element of content.elements){
    if(elements.has(element.id))throw new Error('Duplicate element ID');elements.set(element.id,element);
    if(new Set(element.groupIds).size!==element.groupIds.length)throw new Error('Duplicate group ID');
  }
  const target=(reference:string,from:string):NativeElement=>{
    const value=elements.get(reference);if(!value||reference===from)throw new Error('Invalid element reference');return value;
  };
  for(const element of content.elements){
    if(element.frameId){const frame=target(element.frameId,element.id);if(!['frame','magicframe'].includes(frame.type))throw new Error('Invalid frame reference');}
    for(const ref of element.boundElements??[]){if(target(ref.id,element.id).type!==ref.type)throw new Error('Invalid bound element type');}
    if(element.type==='text'&&element.containerId&&!['rectangle','diamond','ellipse','arrow'].includes(target(element.containerId,element.id).type))throw new Error('Invalid text container');
    if(element.type==='line'||element.type==='arrow')for(const ref of [element.startBinding,element.endBinding])if(ref&&!['rectangle','diamond','ellipse','text','image','frame','magicframe'].includes(target(ref.elementId,element.id).type))throw new Error('Invalid binding reference');
    if(element.type==='freedraw'&&element.pressures.length!==element.points.length&&!(element.simulatePressure&&element.pressures.length===0))throw new Error('Pressure/path length mismatch');
    if(element.type==='image'&&element.fileId&&!content.files[element.fileId])throw new Error('Invalid file reference');
  }
  const checked=new Set<string>();
  for(const element of content.elements){
    const path=new Set<string>();let ancestor:string|null=element.id;
    while(ancestor&&!checked.has(ancestor)){
      if(path.has(ancestor))throw new Error('Cyclic frame reference');
      path.add(ancestor);ancestor=elements.get(ancestor)?.frameId??null;
    }
    for(const visited of path)checked.add(visited);
  }
  for(const [key,file] of Object.entries(content.files))if(key!==file.id)throw new Error('File ID mismatch');
  if(content.presentationOrder){
    if(new Set(content.presentationOrder).size!==content.presentationOrder.length)throw new Error('Duplicate presentation frame');
    for(const ref of content.presentationOrder)if(!['frame','magicframe'].includes(elements.get(ref)?.type??''))throw new Error('Invalid presentation reference');
  }
  if(new TextEncoder().encode(JSON.stringify(content)).length>MAX_DOCUMENT_BYTES)throw new Error('Materialized document byte limit exceeded');
  return content;
}
export * from './protocol.js';
