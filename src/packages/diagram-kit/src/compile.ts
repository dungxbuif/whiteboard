import {validateDiagram} from './catalog.js';
import {layoutDiagram} from './layout.js';
import {toNativeElements} from './native.js';
import {DiagramError,type DiagramInput,type CompiledDiagram} from './types.js';
export async function compileDiagram(value:unknown):Promise<CompiledDiagram>{
 const validation=validateDiagram(value);if(!validation.valid)throw new DiagramError('INVALID_DIAGRAM',validation.errors.map(e=>`${e.path}: ${e.message}`).join('\n'),validation.errors);
 const input=structuredClone(value) as DiagramInput;
 const layout=await layoutDiagram(input);const elements=toNativeElements(layout,input);
 const points=elements.flatMap(e=>e.type==='arrow'||e.type==='line'?e.points.map(([x,y])=>[e.x+x,e.y+y]):[[e.x,e.y],[e.x+e.width,e.y+e.height]]);
 const x=Math.min(...points.map(p=>p[0])),y=Math.min(...points.map(p=>p[1]));
 return {elements,files:[],bounds:{x,y,width:Math.max(...points.map(p=>p[0]))-x,height:Math.max(...points.map(p=>p[1]))-y},warnings:validation.warnings};
}
