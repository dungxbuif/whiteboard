#!/usr/bin/env node
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {join} from 'node:path';
const root=fileURLToPath(new URL('../../',import.meta.url));
const database=process.env.COLLAB_TEST_DATABASE_URL;
try {
 const url=new URL(database??'');
 if(!['postgres:','postgresql:'].includes(url.protocol)||url.hostname!=='127.0.0.1'||url.port!=='55439'||url.pathname!=='/devtool_a1_test'||[...url.searchParams].some(([key,value])=>key!=='sslmode'||value!=='disable'))throw new Error();
} catch {
 console.error('Disposable recovery requires COLLAB_TEST_DATABASE_URL exactly at 127.0.0.1:55439/devtool_a1_test; connection overrides are refused.');
 process.exit(2);
}
for(const [cwd,args] of [
 [root,[join(root,'src/node_modules/typescript/bin/tsc'),'-p','src/collab/tsconfig.json']],
 [join(root,'src/collab'),[join(root,'src/node_modules/vitest/vitest.mjs'),'run','tests/recovery.test.ts','tests/recovery.integration.test.ts','--silent=false','--reporter=verbose']]
]) {
 const result=spawnSync(process.execPath,args,{cwd,env:process.env,stdio:'inherit'});
 if(result.error)throw result.error;
 if(result.status!==0)process.exit(result.status??1);
}
