import {z} from 'zod';
import type {Tool} from '@modelcontextprotocol/sdk/types.js';
import {listDiagramTypes,getDiagramTemplate,validateDiagram,DiagramError} from '@collab/diagram-kit';
import {RenderWorker,RenderError} from './render-worker.js';
const renderer=new RenderWorker();
import {AccessError} from './auth.js';
import type {GoClient} from './go-client.js';
import {mutationFields} from './scene-tools.js';
import {mutationResult,submitMutation} from './mutation-client.js';
const input={preset:z.string().max(80),data:z.unknown(),instanceId:z.string().min(1).max(100),schemaVersion:z.literal(1).default(1),theme:z.enum(['sketch','clean','pastel']).default('clean'),direction:z.enum(['LR','TB']).default('LR')};
export const diagramSchemas={compile_mermaid:z.object({source:z.string().max(50000)}).strict(),list_diagram_types:z.object({}).strict(),get_diagram_template:z.object({preset:z.string().max(80)}).strict(),validate_diagram:z.object(input).strict(),generate_diagram:z.object({...input,...mutationFields,dryRun:z.boolean().default(false)}).strict()};
export const diagramTools:Tool[]=Object.entries(diagramSchemas).map(([name,schema])=>({name,description:name==='compile_mermaid'?'Compile the documented flowchart/sequence Mermaid subset into editable native elements. No image fallback. Unsupported syntax gives typed catalog alternatives. Does not mutate scenes.':name==='generate_diagram'?'Compile native editable elements and apply one durable mutation. Requires scene:write. Stable instanceId/requestId/expectedSequence required. dryRun only compiles.':'Discover and validate native diagram inputs; no scene content read.',inputSchema:z.toJSONSchema(schema) as Tool['inputSchema'],annotations:{readOnlyHint:name!=='generate_diagram',destructiveHint:false,idempotentHint:true,openWorldHint:false}}));
export async function callDiagramTool(name:string,args:Record<string,unknown>,api:GoClient):Promise<Record<string,unknown>|undefined>{
 try{
  if(name==='compile_mermaid'){const a=diagramSchemas.compile_mermaid.parse(args);return {...await renderer.compileMermaid(a.source)};}
  if(name==='list_diagram_types'){diagramSchemas.list_diagram_types.parse(args);return {items:listDiagramTypes()};}
  if(name==='get_diagram_template'){const a=diagramSchemas.get_diagram_template.parse(args);return getDiagramTemplate(a.preset);}
  if(name==='validate_diagram'){const a=diagramSchemas.validate_diagram.parse(args);return {...validateDiagram(a)};}
  if(name==='generate_diagram'){
   const a=diagramSchemas.generate_diagram.parse(args);
   const diagram=await renderer.compileDiagram({preset:a.preset,data:a.data,instanceId:a.instanceId,schemaVersion:a.schemaVersion,theme:a.theme,direction:a.direction});
   const createdIds=diagram.elements.map(e=>e.id);
   if(a.dryRun)return {dryRun:true,createdIds,warnings:diagram.warnings,bounds:diagram.bounds,elements:diagram.elements};
   const receipt=await submitMutation(api,{...a.sceneRef,requestId:a.requestId,expectedSequence:a.expectedSequence,commands:[{op:'add',elements:diagram.elements}]},name);
   return mutationResult(api,a.sceneRef,receipt,createdIds,diagram.warnings);
  }
 }catch(error){if(error instanceof RenderError)throw new AccessError(error.code==='RENDER_BUSY'?429:400,error.code);if(error instanceof DiagramError)throw new AccessError(400,error.code==='UNSUPPORTED_NATIVE_DIAGRAM'?error.code:'INVALID_INPUT');throw error;}
}
export function diagramResources(){return [{uri:'whiteboard://diagrams/catalog',name:'Native diagram catalog',mimeType:'application/json'},...listDiagramTypes().map(p=>({uri:p.schemaUri,name:p.title+' schema',mimeType:'application/json'}))];}
export function readDiagramResource(uri:string):unknown{
 if(uri==='whiteboard://diagrams/catalog')return listDiagramTypes();
 const preset=listDiagramTypes().find(p=>p.schemaUri===uri);if(!preset)throw new AccessError(404,'NOT_FOUND');return getDiagramTemplate(preset.id);
}
