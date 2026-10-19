import {describe,it,expect} from 'vitest';
import {listDiagramTypes,getDiagramTemplate,compileDiagram,validateDiagram} from './index';

const presets=['flowchart','sequence','architecture','deployment','infrastructure','c4-context','c4-container','c4-component','erd','class','state','mindmap','timeline','user-journey','swimlane','org-chart','network','data-flow','agent-workflow'];
const input=(preset:string,instanceId='fixture')=>({preset,instanceId,schemaVersion:1,theme:'pastel',direction:'LR',data:getDiagramTemplate(preset).example});
describe('native catalog contract',()=>{
 it('discovers every requested preset with schema and canonical example',()=>{
  expect(listDiagramTypes().map(p=>p.id)).toEqual(presets);
  for(const preset of presets){expect(getDiagramTemplate(preset).schema).toHaveProperty('type','object');expect(validateDiagram(input(preset)).valid).toBe(true);}
 });
 for(const preset of presets) it(`${preset}: native, stable, disjoint, bound and finite`,async()=>{
  const first=await compileDiagram(input(preset));const repeated=await compileDiagram(input(preset));const other=await compileDiagram(input(preset,'other'));
  expect(first.elements.length).toBeGreaterThan(2);
  expect(first.elements.every((e:any)=>['rectangle','diamond','ellipse','text','arrow','line'].includes(e.type))).toBe(true);
  expect(first.elements.map((e:any)=>e.id)).toEqual(repeated.elements.map((e:any)=>e.id));
  const ids=new Set(first.elements.map((e:any)=>e.id));expect(other.elements.filter((e:any)=>ids.has(e.id))).toEqual([]);
  expect(ids.size).toBe(first.elements.length);
  for(const e of first.elements){expect(Number.isFinite(e.x)&&Number.isFinite(e.y)&&Number.isFinite(e.width)&&Number.isFinite(e.height)).toBe(true);if(e.type==='arrow'||e.type==='line')for(const b of [e.startBinding,e.endBinding])if(b)expect(ids.has(b.elementId)).toBe(true);if(e.type==='text'&&e.containerId)expect(ids.has(e.containerId)).toBe(true);for(const b of e.boundElements??[])expect(ids.has(b.id)).toBe(true);}
  const shapes=first.elements.filter((e:any)=>e.customData?.diagramRole==='node');
  for(let a=0;a<shapes.length;a++)for(let b=a+1;b<shapes.length;b++){const x=shapes[a],y=shapes[b];expect(x.x+x.width<=y.x||y.x+y.width<=x.x||x.y+x.height<=y.y||y.y+y.height<=x.y).toBe(true);}
 });
 it('rejects missing endpoints before producing output',async()=>{
  const bad={...input('flowchart'),data:{nodes:[{id:'a',label:'A'}],edges:[{id:'e',from:'a',to:'missing'}]}};
  expect(validateDiagram(bad)).toMatchObject({valid:false,errors:[expect.objectContaining({code:'MISSING_NODE',path:expect.any(String)})]});
  await expect(compileDiagram(bad)).rejects.toMatchObject({code:'INVALID_DIAGRAM'});
 });
 it('rejects unknown fields and bounded input',()=>{
  expect(validateDiagram({...input('flowchart'),execute:'oops'}).valid).toBe(false);
  expect(validateDiagram({...input('flowchart'),data:{nodes:[{id:'a',label:'x'.repeat(2001)}],edges:[]}}).valid).toBe(false);
 });
 it('reports graph cycles and rejects parent cycles',async()=>{
  const cycle={...input('flowchart'),data:{nodes:[{id:'a',label:'A'},{id:'b',label:'B'}],edges:[{id:'ab',from:'a',to:'b'},{id:'ba',from:'b',to:'a'}]}};
  expect((await compileDiagram(cycle)).warnings).toEqual(expect.arrayContaining([expect.objectContaining({code:'CYCLE'})]));
  expect(validateDiagram({...input('mindmap'),data:{nodes:[{id:'a',label:'A',parent:'b'},{id:'b',label:'B',parent:'a'}]}}).valid).toBe(false);
 });
 it('retains Unicode and wraps long labels with actionable warnings',async()=>{
  const label='Đồng bộ 日本語 🤖 '+('long label '.repeat(12));
  const result=await compileDiagram({...input('flowchart'),data:{nodes:[{id:'unicode',label}],edges:[]}});
  expect(result.elements.some((e:any)=>e.type==='text'&&e.originalText===label)).toBe(true);
  expect(result.warnings).toEqual(expect.arrayContaining([expect.objectContaining({code:'LONG_LABEL'})]));
 });
});
