import {describe,it,expect} from 'vitest';
import {compileMermaid,compileDiagram,getDiagramTemplate,convertWithExcalidraw} from './index';
import {canonicalFixtures} from './fixtures/canonical';
describe('bounded native Mermaid capability',()=>{
 it('compiles declared decisions and labeled connections as native editable elements',async()=>{
  const source='flowchart LR\nA[Request] --> B{Approved?}\nB -->|Yes| C(Ship)';
  const a=await compileMermaid(source,{instanceId:'native-mermaid'}),b=await compileMermaid(source,{instanceId:'native-mermaid'});
  expect(a.elements).toEqual(b.elements);expect(a.files).toEqual([]);expect(a.elements.some(e=>e.type==='diamond')).toBe(true);expect(a.elements.some(e=>e.type==='text'&&e.originalText==='Yes')).toBe(true);
 });
 it('compiles ordered participant calls and replies without image fallback',async()=>{
  const result=await compileMermaid('sequenceDiagram\nactor User as User\nparticipant API as Scene API\nUser->>API: Open scene\nAPI-->>User: Scene ready');
  expect(result.elements.filter(e=>e.customData?.diagramRole==='connector')).toHaveLength(2);
  expect(result.elements.some(e=>e.type==='arrow'&&e.strokeStyle==='dashed')).toBe(true);expect(result.elements.every(e=>e.type!=='image')).toBe(true);
 });
 for(const source of ['pie\n"A": 10','flowchart LR\nclick A "https://example.com"','flowchart LR\nA[<img src=x>]','flowchart LR\n%%{init: {}}%%\nA --> B','sequenceDiagram\nloop forever\nA->>B: test\nend'])it(`rejects unsupported syntax: ${source.split('\n')[0]}`,async()=>{await expect(compileMermaid(source)).rejects.toMatchObject({code:'UNSUPPORTED_NATIVE_DIAGRAM'});});
 it('rejects oversized source before layout',async()=>{await expect(compileMermaid('flowchart LR\n'+'A'.repeat(50000))).rejects.toMatchObject({code:'INVALID_DIAGRAM'});});
});
describe('native normalization and adversarial identifiers',()=>{
 it('all presets compile in both directions and all themes with strict scene validation',async()=>{
  for(const fixture of canonicalFixtures())for(const direction of ['LR','TB'] as const)for(const theme of ['sketch','clean','pastel'] as const){const result=await compileDiagram({...fixture,direction,theme});expect(result.elements.every(e=>Number.isFinite(e.x)&&Number.isFinite(e.y))).toBe(true);}
 });
 it('separates generated headers/lifelines from user semantic names',async()=>{
  const common={instanceId:'đồng-bộ 🤖',schemaVersion:1,theme:'clean',direction:'LR'};
  const lane=await compileDiagram({...common,preset:'swimlane',data:{lanes:[{id:'same',label:'Lane'}],steps:[{id:'same',label:'Step',lane:'same'}],edges:[]}});
  expect(new Set(lane.elements.map(e=>e.id)).size).toBe(lane.elements.length);
  const sequence=await compileDiagram({...common,preset:'sequence',data:{participants:[{id:'a',label:'A'},{id:'b',label:'B'}],messages:[{id:'a',from:'a',to:'b',label:'Call'}]}});
  expect(new Set(sequence.elements.map(e=>e.id)).size).toBe(sequence.elements.length);
 });
 it('does not expose mutable catalog definitions',()=>{const before=getDiagramTemplate('flowchart');const mutable=getDiagramTemplate('flowchart');(mutable.example as {nodes:{label:string}[]}).nodes[0].label='Changed';expect(getDiagramTemplate('flowchart')).toEqual(before);});
 it('official converter adapter rejects ID regeneration',async()=>{const fixture=canonicalFixtures()[0];const result=await compileDiagram(fixture);expect(()=>convertWithExcalidraw(result.elements,elements=>elements.map(e=>({...e,id:'regenerated'})))).toThrow('semantic IDs');});
 it('rejects duplicate semantic IDs and missing journey actors with actionable paths',async()=>{
  const base=canonicalFixtures()[0];await expect(compileDiagram({...base,data:{nodes:[{id:'a',label:'A'},{id:'a',label:'B'}],edges:[]}})).rejects.toMatchObject({code:'INVALID_DIAGRAM',diagnostics:expect.arrayContaining([expect.objectContaining({code:'DUPLICATE_ID'})])});
  await expect(compileDiagram({...base,preset:'user-journey',data:{actors:[{id:'a',label:'A'}],stages:[{id:'s',label:'S',steps:[{id:'t',label:'Action',actor:'missing'}]}]}})).rejects.toMatchObject({code:'INVALID_DIAGRAM',diagnostics:expect.arrayContaining([expect.objectContaining({code:'MISSING_ACTOR'})])});
 });
 it('keeps grouped boundaries separate from nonmember nodes',async()=>{
  const base=canonicalFixtures()[0];
  const result=await compileDiagram({...base,data:{nodes:[{id:'a',label:'A',group:'g'},{id:'outside',label:'Outside'},{id:'c',label:'C',group:'g'}],edges:[{id:'a',from:'a',to:'outside'},{id:'c',from:'outside',to:'c'}],groups:[{id:'g',label:'Group'}]}});
  const boundary=result.elements.find(e=>e.customData?.diagramRole==='boundary')!,outside=result.elements.find(e=>e.customData?.semanticId==='outside'&&e.customData?.diagramRole==='node')!;
  expect(boundary.x+boundary.width<=outside.x||outside.x+outside.width<=boundary.x||boundary.y+boundary.height<=outside.y||outside.y+outside.height<=boundary.y).toBe(true);
 });
 it('places connector labels between nodes without covering their boxes',async()=>{
  const result=await compileDiagram({...canonicalFixtures()[0],data:{nodes:[{id:'a',label:'Request'},{id:'b',label:'Response'}],edges:[{id:'call',from:'a',to:'b',label:'HTTPS'}]}});
  const edgeIds=new Set(result.elements.filter(e=>e.type==='arrow').map(e=>e.id));const labels=result.elements.filter(e=>e.type==='text'&&e.containerId&&edgeIds.has(e.containerId));const nodes=result.elements.filter(e=>e.customData?.diagramRole==='node');
  for(const label of labels)for(const node of nodes)expect(label.x+label.width<=node.x||node.x+node.width<=label.x||label.y+label.height<=node.y||node.y+node.height<=label.y).toBe(true);
 });
});
