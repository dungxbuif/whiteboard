import {compileDiagram} from './compile.js';
import {DiagramError,LIMITS,type DiagramInput,type GraphData,type SequenceData} from './types.js';
const unsupported=(detail:string):never=>{throw new DiagramError('UNSUPPORTED_NATIVE_DIAGRAM',`${detail} Use getDiagramTemplate('flowchart') or getDiagramTemplate('sequence') for the typed native alternative.`);};
export const MERMAID_SUPPORT={flowchart:'Simple LR/TB/TD graph headers, ID[Label], ID{Decision}, ID(Label), A --> B and A -->|label| B; one statement per line.',sequence:'sequenceDiagram, participant/actor ID as Label, A->>B: message, A-->>B: return; one statement per line.'} as const;
export async function compileMermaid(source:string,options:Partial<Pick<DiagramInput,'instanceId'|'theme'|'direction'>>={} ){
 if(typeof source!=='string'||source.length>LIMITS.source)throw new DiagramError('INVALID_DIAGRAM',`Mermaid source must fit ${LIMITS.source} characters.`);
 if(/%%\s*\{/.test(source))unsupported('Mermaid configuration directives are not accepted.');
 const lines=source.split(/\r?\n/).map(l=>l.trim()).filter(l=>l&&!l.startsWith('%%'));
 const header=lines.shift()??'';const common={instanceId:options.instanceId??'mermaid',theme:options.theme??'clean',schemaVersion:1 as const,direction:options.direction??'LR'};
 if(/^(flowchart|graph) (LR|TB|TD)$/.test(header)){
  const data:GraphData={nodes:[],edges:[]};const map=new Map<string,GraphData['nodes'][number]>();
  const node=(token:string)=>{const match=/^([\w-]+)(?:\[([^\[\]<>]+)\]|\{([^{}<>]+)\}|\(([^()<>]+)\))?$/.exec(token.trim());if(!match)unsupported(`Unsupported flowchart token: ${token}.`);const [,id,box,diamond,round]=match!;const old=map.get(id);const label=box??diamond??round??old?.label??id;const item={id,label,kind:diamond?'decision':round?'start':'process'};if(!old){map.set(id,item);data.nodes.push(item);}else if(box||diamond||round){Object.assign(old,item);}return id;};
  for(const [i,line] of lines.entries()){const match=/^(.+?)\s*-->(?:\|([^|<>]+)\|)?\s*(.+)$/.exec(line);if(match){data.edges.push({id:`edge-${i}`,from:node(match[1]),to:node(match[3]),...(match[2]?{label:match[2]}:{})});}else node(line);}
  return compileDiagram({...common,direction:options.direction??(header.endsWith('LR')?'LR':'TB'),preset:'flowchart',data});
 }
 if(header==='sequenceDiagram'){
  const data:SequenceData={participants:[],messages:[]};const ids=new Set<string>();
  const participant=(id:string,label=id,kind:'actor'|'participant'='participant')=>{if(!ids.has(id)){ids.add(id);data.participants.push({id,label,kind});}else if(label!==id){const existing=data.participants.find(p=>p.id===id)!;existing.label=label;existing.kind=kind;}};
  for(const [i,line] of lines.entries()){const p=/^(participant|actor) ([\w-]+)(?: as ([^<>]+))?$/.exec(line);if(p){participant(p[2],p[3]??p[2],p[1] as 'actor'|'participant');continue;}const m=/^([\w-]+)(-->>|->>)([\w-]+):\s*([^<>]+)$/.exec(line);if(!m)unsupported(`Unsupported sequence statement at line ${i+2}.`);participant(m![1]);participant(m![3]);data.messages.push({id:`message-${i}`,from:m![1],to:m![3],label:m![4],kind:m![2]==='-->>'?'return':'call'});}
  return compileDiagram({...common,preset:'sequence',data});
 }
 return unsupported(`Diagram header ${header||'(empty)'} is not supported natively.`);
}
