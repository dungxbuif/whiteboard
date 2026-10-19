import {beforeAll,test,expect} from 'vitest';
import {build} from 'vite';
import {compileMermaid} from '@collab/diagram-kit';
import {chromium} from 'playwright';
import {fileURLToPath} from 'node:url';
import {createRenderContext,RENDER_ORIGIN} from '../src/render-sandbox.js';
const root=fileURLToPath(new URL('./browser',import.meta.url)),out=fileURLToPath(new URL('../test-render-dist',import.meta.url));
beforeAll(async()=>{await build({configFile:false,root,logLevel:'error',build:{outDir:out,emptyOutDir:true}});},30000);
test('patched KaTeX renders normal math but never inherits trust for external resources or links',async()=>{
 const browser=await chromium.launch();try{const context=await createRenderContext(browser,out);const page=await context.newPage();await page.goto(RENDER_ORIGIN);await page.waitForFunction(()=>Boolean((globalThis as any).fixture));
 const result=await page.evaluate(()=>{const {katex}= (globalThis as any).fixture;const normal=katex.renderToString('x^2+\\frac{1}{2}');const expressions=['\\href{javascript:alert(1)}{click}','\\includegraphics{http://127.0.0.1:9/secret}','\\htmlStyle{background:url(https://example.com/x)}{x}'];const inherited=expressions.map(s=>katex.renderToString(s,Object.create({trust:true})));let polluted:string[]=[];try{(Object.prototype as any).trust=true;polluted=expressions.map(s=>katex.renderToString(s));}finally{delete (Object.prototype as any).trust;}return {version:katex.version,normal,inherited,polluted};});
 expect(result.version).toBe('0.18.2');expect(result.normal).toContain('katex');for(const html of [...result.inherited,...result.polluted]){expect(html).not.toMatch(/<a\b|<img\b|style="[^"]*background:url/i);expect(html).not.toMatch(/href="javascript:/);}
 }finally{await browser.close();}
},15000);
test('official Mermaid core with patched KaTeX preserves native flowchart and sequence conversions',async()=>{
 const browser=await chromium.launch();try{const context=await createRenderContext(browser,out);const page=await context.newPage();await page.goto(RENDER_ORIGIN);await page.waitForFunction(()=>Boolean((globalThis as any).fixture));const results=await page.evaluate(async()=>{const f=(globalThis as any).fixture;const results=[];for(const source of ['flowchart LR\nA[Hello] --> B[World]','sequenceDiagram\nA->>B: Message','flowchart LR\nA["$$x^2$$"] --> B[Math]']){const parsed=await f.parseMermaidToExcalidraw(source);const elements=f.convertToExcalidrawElements(parsed.elements);const svg=await f.exportToSvg({elements,files:parsed.files??{},appState:{}});results.push({types:elements.map((e:any)=>e.type),svg:svg.outerHTML});}return results;});for(const r of results){expect(r.types.length).toBeGreaterThan(1);expect(r.types).not.toContain('image');expect(r.svg).toContain('<svg');expect(r.svg).not.toMatch(/javascript:|<script/i);}
 }finally{await browser.close();}
},20000);

test('D2 native normalizer survives official browser conversion with stable IDs and text containers',async()=>{
 const diagram=await compileMermaid('flowchart TB\nA[Tiếng Việt 日本語 and a long multiline label that wraps] --> B[終わり]',{theme:'sketch'});const browser=await chromium.launch();try{const context=await createRenderContext(browser,out);const page=await context.newPage();await page.goto(RENDER_ORIGIN);await page.waitForFunction(()=>Boolean((globalThis as any).fixture));const result=await page.evaluate(async input=>{const f=(globalThis as any).fixture;const elements=f.convertToExcalidrawElements(input,{regenerateIds:false});const svg=await f.exportToSvg({elements,appState:{},files:{}});return {elements,svg:svg.outerHTML};},diagram.elements);expect(result.elements.map((e:any)=>e.id).sort()).toEqual(diagram.elements.map(e=>e.id).sort());expect(result.elements.some((e:any)=>e.type==='image')).toBe(false);for(const e of result.elements.filter((e:any)=>e.type==='text')){expect(e.text).toBeTruthy();expect(e.containerId).toBeTruthy();expect(result.elements.find((x:any)=>x.id===e.containerId)?.boundElements.some((x:any)=>x.id===e.id)).toBe(true);}expect(result.svg).toContain('Tiếng Việt');expect(result.svg).toContain('終わり');
 }finally{await browser.close();}
},15000);
