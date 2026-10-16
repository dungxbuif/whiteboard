import {test,expect} from 'vitest';
import {spawnSync} from 'node:child_process';
test('built diagram-kit imports in ordinary Node ESM',()=>{
 const result=spawnSync(process.execPath,['--input-type=module','-e',"const m=await import('./dist/index.js');if(m.listDiagramTypes().length!==19)process.exit(2)"],{cwd:process.cwd(),encoding:'utf8'});
 expect(result.stderr).toBe('');expect(result.status).toBe(0);
});
