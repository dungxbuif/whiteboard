import {graphSchema,graphExample} from './presets/graph.js';
import {sequenceSchema,sequenceExample} from './presets/sequence.js';
import {erdSchema,erdExample} from './presets/erd.js';
import {classSchema,classExample} from './presets/class.js';
import {stateSchema,stateExample} from './presets/state.js';
import {treeSchema,mindmapExample,orgExample} from './presets/tree.js';
import {timelineSchema,timelineExample} from './presets/timeline.js';
import {journeySchema,journeyExample} from './presets/journey.js';
import {swimlaneSchema,swimlaneExample} from './presets/swimlane.js';
import {object,str,enumeration,checkSchema} from './schema.js';
import {DiagramError,LIMITS,type Definition,type DiagramInput,type Diagnostic,type PresetId,type GraphData,type ValidationResult} from './types.js';

const rows:[PresetId,string,string,string][]=[
 ['flowchart','Flowchart','Branching processes with decisions and outcomes.','graph'],['sequence','Sequence','Ordered calls, replies and participant lifelines.','sequence'],
 ['architecture','Architecture','Services, stores and editable system boundaries.','graph'],['deployment','Deployment','Hosts and deployment communication.','graph'],['infrastructure','Infrastructure','Gateways, compute and storage topology.','graph'],
 ['c4-context','C4 context','People, the system and external dependencies.','graph'],['c4-container','C4 container','Applications, databases and technology labels.','graph'],['c4-component','C4 component','Internal components and their dependencies.','graph'],
 ['erd','Entity relationship','Entities, typed fields, keys and cardinality.','erd'],['class','Class','Members, methods and relationship semantics.','class'],['state','State machine','States and labeled transitions.','state'],
 ['mindmap','Mind map','Ideas connected through parent relationships.','tree'],['timeline','Timeline','Ordered dated events and milestones.','timeline'],['user-journey','User journey','Actor actions, stages and experience scores.','journey'],['swimlane','Swimlane','Ordered steps assigned to responsibility lanes.','swimlane'],['org-chart','Organization chart','People, roles and reporting lines.','tree'],
 ['network','Network','Network devices, hosts and connection labels.','graph'],['data-flow','Data flow','Inputs, processes, stores and outputs.','graph'],['agent-workflow','Agent workflow','Agents, models, tools and reviewed outputs.','graph']
];
const special:Record<string,[Definition['schema'],Definition['example']]>={sequence:[sequenceSchema,sequenceExample],erd:[erdSchema,erdExample],class:[classSchema,classExample],state:[stateSchema,stateExample],mindmap:[treeSchema,mindmapExample],'org-chart':[treeSchema,orgExample],timeline:[timelineSchema,timelineExample],'user-journey':[journeySchema,journeyExample],swimlane:[swimlaneSchema,swimlaneExample]};
export const definitions:Definition[]=rows.map(([id,title,description,family])=>({descriptor:{id,title,description,family,schemaVersion:1,editable:true,schemaUri:`whiteboard://diagrams/${id}/schema`},schema:special[id]?.[0]??graphSchema,example:special[id]?.[1]??graphExample(id)}));
export function definition(id:string):Definition{const found=definitions.find(p=>p.descriptor.id===id);if(!found)throw new DiagramError('UNKNOWN_PRESET',`Unknown preset ${id}; call listDiagramTypes().`);return found;}
export function listDiagramTypes(){return structuredClone(definitions.map(d=>d.descriptor));}
export function getDiagramTemplate(id:string){const d=definition(id);return structuredClone({schema:d.schema,example:d.example});}
export function getDiagramInputSchema(preset:string){return object({preset:{const:preset,type:'string'},instanceId:str(LIMITS.instance),schemaVersion:{const:1,type:'number'},theme:enumeration('sketch','clean','pastel'),direction:enumeration('LR','TB'),data:definition(preset).schema});}

export function asGraph(input:DiagramInput):GraphData{
 const data=input.data;
 if('entities' in data)return {nodes:data.entities.map(e=>({id:e.id,label:e.label,kind:'database',description:e.fields.map(f=>`${f.key?f.key+' ':''}${f.name}: ${f.type}`).join('\n')})),edges:data.relations.map(r=>({id:r.id,from:r.from,to:r.to,label:[r.label,r.cardinality].filter(Boolean).join(' · '),kind:r.cardinality}))};
 if('classes' in data)return {nodes:data.classes.map(c=>({id:c.id,label:c.label,kind:'component',description:[c.members.join('\n'),'──────────',c.methods.join('\n')].filter(Boolean).join('\n')})),edges:data.relations};
 if('nodes' in data){if(input.preset==='mindmap'||input.preset==='org-chart'){const nodes=data.nodes as import('./types.js').TreeData['nodes'];return {nodes:nodes.map(n=>({id:n.id,label:n.label,description:n.role,kind:input.preset==='org-chart'?'person':'process'})),edges:nodes.filter(n=>n.parent).map(n=>({id:`parent:${n.id}`,from:n.parent!,to:n.id}))};}return data as GraphData;}
 if('steps' in data)return {nodes:data.steps.map(s=>({id:s.id,label:s.label,kind:s.kind})),edges:data.edges};
 return {nodes:[],edges:[]};
}
function hasCycle(nodes:string[],edges:{from:string;to:string}[]):boolean{
 const outgoing=new Map(nodes.map(id=>[id,[] as string[]]));for(const e of edges)outgoing.get(e.from)?.push(e.to);
 const visiting=new Set<string>(),visited=new Set<string>();
 const walk=(id:string):boolean=>{if(visiting.has(id))return true;if(visited.has(id))return false;visiting.add(id);for(const next of outgoing.get(id)??[])if(walk(next))return true;visiting.delete(id);visited.add(id);return false;};
 return nodes.some(walk);
}
export function validateDiagram(value:unknown):ValidationResult{
 const errors:Diagnostic[]=[],warnings:Diagnostic[]=[];
 const error=(code:string,path:string,message:string)=>errors.push({code,path,message});
 let serialized:string;try{serialized=JSON.stringify(value);}catch{error('SCHEMA','$','Input must be acyclic JSON data.');return {valid:false,errors,warnings};}
 if(!serialized||serialized.length>LIMITS.source){error('LIMIT','$',`Input must fit ${LIMITS.source} JSON characters.`);return {valid:false,errors,warnings};}
 const preset=(value as {preset?:unknown}|null)?.preset;
 if(typeof preset!=='string'||!definitions.some(d=>d.descriptor.id===preset)){error('UNKNOWN_PRESET','$.preset','Choose a preset returned by listDiagramTypes().');return {valid:false,errors,warnings};}
 checkSchema(value,getDiagramInputSchema(preset),'$',errors);if(errors.length)return {valid:false,errors,warnings};
 const input=value as DiagramInput,data=input.data;
 const unique=(items:{id:string}[],path:string)=>{const seen=new Set<string>();for(const [i,item] of items.entries()){if(seen.has(item.id))error('DUPLICATE_ID',`${path}[${i}].id`,`Duplicate ID ${item.id}; use unique semantic IDs.`);seen.add(item.id);}return seen;};
 const refs=(items:{from:string;to:string}[],ids:Set<string>,path:string)=>{for(const [i,e] of items.entries())for(const side of ['from','to'] as const)if(!ids.has(e[side]))error('MISSING_NODE',`${path}[${i}].${side}`,`Node ${e[side]} does not exist; declare it before connecting.`);};
 const graph=asGraph(input);
 if(graph.nodes.length){const ids=unique(graph.nodes,'$.data.nodes');unique(graph.edges,'$.data.edges');refs(graph.edges,ids,'$.data.edges');if(hasCycle([...ids],graph.edges)){if(input.preset==='mindmap'||input.preset==='org-chart')error('PARENT_CYCLE','$.data.nodes','Parent relationships contain a cycle; remove a parent link.');else warnings.push({code:'CYCLE',path:'$.data.edges',message:'Graph contains a cycle; ELK will route feedback edges. Check transition direction if unexpected.'});}
  const groups=unique(graph.groups??[],'$.data.groups');for(const [i,n] of graph.nodes.entries())if(n.group&&!groups.has(n.group))error('MISSING_GROUP',`$.data.nodes[${i}].group`,`Declare group ${n.group} in groups.`);
 }
 if('participants' in data){const ids=unique(data.participants,'$.data.participants');unique(data.messages,'$.data.messages');refs(data.messages,ids,'$.data.messages');}
 if('events' in data)unique(data.events,'$.data.events');
 if('actors' in data){const actors=unique(data.actors,'$.data.actors');unique(data.stages,'$.data.stages');const steps=data.stages.flatMap(s=>s.steps);unique(steps,'$.data.stages.steps');if(steps.length>LIMITS.nodes)error('LIMIT','$.data.stages',`Use at most ${LIMITS.nodes} total journey steps.`);for(const s of steps)if(!actors.has(s.actor))error('MISSING_ACTOR','$.data.stages.steps.actor',`Declare actor ${s.actor}.`);}
 if('lanes' in data){const lanes=unique(data.lanes,'$.data.lanes');for(const [i,s] of data.steps.entries())if(!lanes.has(s.lane))error('MISSING_LANE',`$.data.steps[${i}].lane`,`Declare lane ${s.lane}.`);}
 const labels=(v:unknown,path:string)=>{if(typeof v==='string'&&v.length>80)warnings.push({code:'LONG_LABEL',path,message:'Label exceeds 80 characters and will wrap; shorten it for a more compact diagram.'});else if(Array.isArray(v))v.forEach((x,i)=>labels(x,`${path}[${i}]`));else if(v&&typeof v==='object')for(const [k,x]of Object.entries(v))labels(x,`${path}.${k}`);};labels(data,'$.data');
 return {valid:errors.length===0,errors,warnings};
}
