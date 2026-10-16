import {validateSceneContent} from '@collab/contracts';
import {DiagramError,type NativeElement,type DiagramInput,type LayoutResult,type LayoutNode,type LayoutEdge} from './types';
import {wrapText} from './layout';

type MutableElement=Record<string,any>;
const FONT_SIZE=18,LINE_HEIGHT=1.3333333333333333,TEXT_INSET=16;
const COLORS=['#d0ebff','#d3f9d8','#fff3bf','#e5dbff','#ffe8cc','#ffdeeb'];
const fingerprint=(text:string)=>[0,1,2,3].map(salt=>hash(`${salt}:${text}`).toString(16).padStart(8,'0')).join('');
export const semanticId=(instance:string,kind:string,id:string)=>`d_${fingerprint(instance)}_${kind}_${fingerprint(id)}`;
function hash(text:string):number{let h=2166136261;for(const ch of text)h=Math.imul(h^ch.codePointAt(0)!,16777619);return (h>>>0)&0x7fffffff;}
function base(id:string,type:string,input:DiagramInput,ordinal:number):MutableElement{return {id,type,x:0,y:0,width:0,height:0,angle:0,strokeColor:'#1e293b',backgroundColor:input.theme==='pastel'?COLORS[ordinal%COLORS.length]:'transparent',fillStyle:input.theme==='sketch'?'hachure':'solid',strokeWidth:2,strokeStyle:'solid',roundness:input.theme==='sketch'?null:{type:3},roughness:input.theme==='sketch'?1:0,opacity:100,seed:hash(id),version:1,versionNonce:hash(id+'v1'),index:null,isDeleted:false,groupIds:[],frameId:null,boundElements:null,updated:0,link:null,locked:false};}
function label(baseElement:MutableElement,text:string,original:string,box:{x:number;y:number;width:number;height:number},containerId:string|null,input:DiagramInput):MutableElement{
 return {...base(baseElement.id+'_text','text',input,0),...box,text,originalText:original,fontSize:FONT_SIZE,fontFamily:input.theme==='sketch'?1:2,textAlign:'center',verticalAlign:'middle',containerId,autoResize:false,lineHeight:LINE_HEIGHT,backgroundColor:'transparent',strokeWidth:1,roundness:null,groupIds:[...baseElement.groupIds],customData:{diagramRole:'label',preset:input.preset,instanceId:input.instanceId}};
}
function nodeElements(n:LayoutNode,input:DiagramInput,index:number):MutableElement[]{
 const id=semanticId(input.instanceId,n.role??'node',n.id),element:MutableElement={...base(id,n.shape??'rectangle',input,index),x:n.x,y:n.y,width:n.width,height:n.height,groupIds:n.group?[semanticId(input.instanceId,'group',n.group)]:[],customData:{diagramRole:n.role??'node',semanticId:n.id,preset:input.preset,instanceId:input.instanceId,kind:n.kind??'process'}};
 if(n.role==='boundary'){element.backgroundColor='transparent';element.strokeStyle='dashed';element.strokeColor='#64748b';}
 const original=n.role==='boundary'?n.label:(n.kind&&n.kind!=='process'&&n.kind!=='state'?`«${n.kind}»\n`:'')+n.label+(n.technology?`\n[${n.technology}]`:'')+(n.description?`\n${n.description}`:'');
 const text=n.text??wrapText(original);if(!text)return [element];
 const height=text.split('\n').length*FONT_SIZE*LINE_HEIGHT;
 const diamond=n.shape==='diamond'||n.shape==='ellipse';const textWidth=diamond?n.width*.6:n.width-2*TEXT_INSET;
 const textElement=label(element,text,original,{x:n.x+(n.width-textWidth)/2,y:n.role==='boundary'?n.y+10:n.y+(n.height-height)/2,width:textWidth,height},n.role==='boundary'?null:id,input);
 if(n.role!=='boundary')element.boundElements=[{id:textElement.id,type:'text'}];else textElement.id=semanticId(input.instanceId,'boundary-label',n.id);
 return [element,textElement];
}
function routeMidpoint(points:[number,number][]):[number,number]{
 const lengths=points.slice(1).map((p,i)=>Math.hypot(p[0]-points[i][0],p[1]-points[i][1]));let remaining=lengths.reduce((a,b)=>a+b,0)/2;
 for(let i=0;i<lengths.length;i++){if(remaining<=lengths[i]){const ratio=lengths[i]?remaining/lengths[i]:0;return [points[i][0]+(points[i+1][0]-points[i][0])*ratio,points[i][1]+(points[i+1][1]-points[i][1])*ratio];}remaining-=lengths[i];}return points[0];
}
function edgeElements(e:LayoutEdge,input:DiagramInput):MutableElement[]{
 const id=semanticId(input.instanceId,e.role??'edge',e.id),[x,y]=e.points[0];const points=e.points.map(([px,py])=>[px-x,py-y]);
 const element:MutableElement={...base(id,'arrow',input,0),x,y,width:Math.max(...points.map(p=>p[0]))-Math.min(...points.map(p=>p[0])),height:Math.max(...points.map(p=>p[1]))-Math.min(...points.map(p=>p[1])),points,startBinding:e.bind===false?null:{elementId:semanticId(input.instanceId,'node',e.from),focus:0,gap:1},endBinding:e.bind===false?null:{elementId:semanticId(input.instanceId,'node',e.to),focus:0,gap:1},startArrowhead:e.startHead??null,endArrowhead:e.endHead===undefined?'arrow':e.endHead,elbowed:false,lastCommittedPoint:null,backgroundColor:'transparent',roundness:{type:2},strokeStyle:e.dashed?'dashed':'solid',customData:{diagramRole:e.role??'connector',semanticId:e.id,preset:input.preset,instanceId:input.instanceId}};
 if(!e.label)return [element];
 const midpoint=routeMidpoint(e.points);const text=wrapText(e.label),height=text.split('\n').length*FONT_SIZE*LINE_HEIGHT,width=Math.min(260,Math.max(80,...text.split('\n').map(t=>Array.from(t).length*10)));
 const textElement=label(element,text,e.label,{x:midpoint[0]-width/2,y:midpoint[1]-height-8,width,height},id,input);
 element.boundElements=[{id:textElement.id,type:'text'}];return [element,textElement];
}
export function repairBindings(elements:MutableElement[]):void{
 const byId=new Map(elements.map(e=>[e.id,e]));
 for(const element of elements)for(const binding of [element.startBinding,element.endBinding])if(binding){const target=byId.get(binding.elementId);if(!target)throw new DiagramError('UNRESOLVED_BINDING',`Missing native binding ${binding.elementId}.`);target.boundElements??=[];if(!target.boundElements.some((b:{id:string})=>b.id===element.id))target.boundElements.push({id:element.id,type:'arrow'});}
}
export function toNativeElements(layout:LayoutResult,input:DiagramInput):NativeElement[]{
 const elements=layout.nodes.flatMap((n,i)=>nodeElements(n,input,i)).concat(layout.edges.flatMap(e=>edgeElements(e,input)));
 if(new Set(elements.map(e=>e.id)).size!==elements.length)throw new DiagramError('ID_COLLISION','Semantic IDs collide; change the colliding input identifier.');
 repairBindings(elements);
 validateSceneContent({elements:elements as NativeElement[],appState:{},files:{}});
 return elements as NativeElement[];
}
/** Browser callers inject the official convertToExcalidrawElements export; Node never imports its React bundle. */
export function convertWithExcalidraw(elements:NativeElement[],convert:(skeletons:any[],options:{regenerateIds:boolean})=>any[]):NativeElement[]{
 const converted=convert(structuredClone(elements),{regenerateIds:false}) as MutableElement[];
 if(converted.length!==elements.length)throw new DiagramError('CONVERSION_MISMATCH','Converter changed the native element count.');
 const ids=new Set(elements.map(e=>e.id));if(converted.some(e=>!ids.has(e.id)))throw new DiagramError('CONVERSION_MISMATCH','Converter changed semantic IDs.');
 repairBindings(converted);validateSceneContent({elements:converted as NativeElement[],appState:{},files:{}});return converted as NativeElement[];
}
