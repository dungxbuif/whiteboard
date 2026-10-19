// Narrow CI entrypoint. No repository CI workflow exists to modify.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const cwd = fileURLToPath(new URL('../../src/', import.meta.url));
for (const args of [
  ['run','build','--workspace','./collab'],
  ['run','build','--workspace','./packages/diagram-kit'],
  ['run','build','--workspace','./mcp'],
  ['run','test:e2e:collab','--workspace','./frontend',...process.argv.slice(2)],
]) {
  const result=spawnSync('rtk',['proxy','npm',...args],{cwd,stdio:'inherit',env:process.env});
  if(result.status!==0)process.exit(result.status??1);
}
