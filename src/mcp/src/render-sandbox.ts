import {readFile,readdir} from 'node:fs/promises';
import {join,relative,extname} from 'node:path';
import type {Browser} from 'playwright';
export const RENDER_ORIGIN='https://renderer.invalid';
const CSP="default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src data: blob:; connect-src 'self'; worker-src 'self' blob:; base-uri 'none'; form-action 'none'; frame-src 'none'";
async function localFiles(root:string):Promise<Map<string,Buffer>>{const files=new Map<string,Buffer>();async function walk(dir:string){for(const f of await readdir(dir,{withFileTypes:true})){const p=join(dir,f.name);if(f.isDirectory())await walk(p);else files.set('/'+relative(root,p).split('\\').join('/'),await readFile(p));}}await walk(root);return files;}

export async function createRenderContext(browser:Browser,root:string){
const files=await localFiles(root);
  const context=await browser.newContext({serviceWorkers:'block',acceptDownloads:false});
  await context.route('**/*',async route=>{
   const url=new URL(route.request().url()),path=url.pathname==='/'?'/index.html':url.pathname;
   const body=url.origin===RENDER_ORIGIN&&!url.search?files.get(path):undefined;
   if(!body){await route.abort('blockedbyclient');return;}
   const types:Record<string,string>={'.html':'text/html','.js':'text/javascript','.css':'text/css','.woff2':'font/woff2','.woff':'font/woff'};
   await route.fulfill({status:200,body,contentType:types[extname(path)]??'application/octet-stream',headers:{'Content-Security-Policy':CSP}});
  });
  await context.routeWebSocket('**/*',socket=>socket.close());

return context;
}
