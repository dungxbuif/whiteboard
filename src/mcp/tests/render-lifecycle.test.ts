import {test,expect} from 'vitest';
import {mkdtemp,writeFile,readFile,access,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {existsSync} from 'node:fs';
import {join} from 'node:path';
import {RenderWorker} from '../src/render-worker.js';

function alive(pid:number){try{process.kill(pid,0);return true;}catch{return false;}}
function observed(extra:Record<string,unknown>={}){
 const directories:string[]=[],browsers:number[]=[],workers:number[]=[],profileReady:boolean[]=[];
 const worker=new RenderWorker({...extra,onTempDirectory:(dir:string)=>directories.push(dir),onBrowserProcess:(pid:number)=>browsers.push(pid),onProcess:(pid:number)=>{workers.push(pid);profileReady.push(existsSync(join(directories.at(-1)!,'profile','DevToolsActivePort')));}} as any);
 return {worker,directories,browsers,workers,profileReady};
}
async function cleaned(job:ReturnType<typeof observed>){
 expect(job.directories.length).toBeGreaterThan(0);
 for(const pid of [...job.browsers,...job.workers]){expect(alive(pid),`process ${pid} survived`).toBe(false);expect(alive(-pid),`process group ${pid} survived`).toBe(false);}
 for(const directory of job.directories)await expect(access(directory),directory).rejects.toThrow();
}

test('owns and reaps a stalled executable before browser startup resolves, including subsequent jobs',async()=>{
 const root=await mkdtemp(join(tmpdir(),'d4-startup-test-')),executable=join(root,'stall.sh'),pidfile=join(root,'pid');
 await writeFile(executable,`#!/bin/sh
echo $$ > '${pidfile}'
exec /bin/sleep 30
`,{mode:0o700});
 const job=observed({browserExecutable:executable,deadlineMs:1500});
 try{
  for(let i=0;i<2;i++){
   await expect(job.worker.compileMermaid('flowchart LR\nA')).rejects.toMatchObject({code:'RENDER_TIMEOUT'});
   const pid=Number(await readFile(pidfile,'utf8'));expect(job.browsers).toContain(pid);expect(alive(pid)).toBe(false);await cleaned(job);
  }
 }finally{for(const pid of job.browsers){try{process.kill(-pid,'SIGKILL');}catch{}}await rm(root,{recursive:true,force:true});}
},10000);

test('repeated success and parser errors close and reap real Chromium and remove exact job storage',async()=>{
 const job=observed();
 for(let i=0;i<2;i++){
  expect((await job.worker.compileMermaid('flowchart LR\nA[Clean] --> B[Next]')).elements.length).toBeGreaterThan(0);await cleaned(job);
  await expect(job.worker.compileMermaid('erDiagram\n A ||--o{ B : has')).rejects.toMatchObject({code:'UNSUPPORTED_NATIVE_DIAGRAM'});await cleaned(job);
 }
 expect(job.browsers).toHaveLength(4);expect(job.workers).toHaveLength(4);expect(job.profileReady).toEqual([true,true,true,true]);
},20000);

test('child crash closes the owned browser and removes exact job storage before retry',async()=>{
 const root=await mkdtemp(join(tmpdir(),'d4-crash-test-')),child=join(root,'crash.cjs');await writeFile(child,'process.once("message",()=>process.exit(17));');const job=observed({childPath:child});
 try{for(let i=0;i<2;i++){await expect(job.worker.compileMermaid('flowchart LR\nA')).rejects.toMatchObject({code:'RENDER_FAILED'});await cleaned(job);}}finally{await rm(root,{recursive:true,force:true});}
},15000);

test('active browser infinite loop is hard-killed and exact job storage is removed before retry',async()=>{
 const root=await mkdtemp(join(tmpdir(),'d4-active-test-')),child=join(root,'loop.mjs'),marker=join(root,'loop-started');
 await writeFile(child,`import{chromium}from ${JSON.stringify(import.meta.resolve('playwright'))};import{writeFileSync}from'node:fs';process.once('message',async({browserEndpoint})=>{const browser=await chromium.connectOverCDP(browserEndpoint);const page=await browser.newPage();writeFileSync(${JSON.stringify(marker)},'started');await page.evaluate(()=>{while(true){}});});`);
 const job=observed({childPath:child,deadlineMs:3000});
 try{for(let i=0;i<2;i++){await rm(marker,{force:true});await expect(job.worker.compileMermaid('flowchart LR\nA')).rejects.toMatchObject({code:'RENDER_TIMEOUT'});expect(await readFile(marker,'utf8')).toBe('started');expect(job.browsers.length).toBe(i+1);expect(job.workers.length).toBe(i+1);await cleaned(job);}}finally{await rm(root,{recursive:true,force:true});}
},10000);


test('failed browser spawn removes its job directory and releases the slot',async()=>{
 const job=observed({browserExecutable:'/nonexistent-whiteboard-chromium'});
 for(let i=0;i<2;i++){await expect(job.worker.compileMermaid('flowchart LR\nA')).rejects.toMatchObject({code:'RENDER_FAILED'});await cleaned(job);}
 expect(job.browsers).toHaveLength(0);expect(job.workers).toHaveLength(0);
});
