import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';

const root=resolve(import.meta.dirname,'../..');
const prefix='whiteboard-feature-worker-'+process.pid;
const network=prefix+'-net';
const containers=[];
const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',stdio:['ignore','pipe','pipe']}).trim();
const pause=ms=>new Promise(resolve=>setTimeout(resolve,ms));
const run=(suffix,args)=>{const name=prefix+'-'+suffix;containers.push(name);docker('run','-d','--name',name,'--network',network,...args);return name;};
const workerArgs=id=>[
  '--init','--read-only','--tmpfs','/tmp:size=32m','--memory','256m','--pids-limit','64',
  '--cap-drop','ALL','--security-opt','no-new-privileges:true',
  '--entrypoint','/app/feature-worker',
  '-e','DATABASE_URL=postgres://fixture:fixture@db:5432/packaging_test?sslmode=disable',
  '-e','FEATURE_WORKER_ID='+id,
  '-e','FEATURE_WORKER_ADDR=127.0.0.1:8090',
  'whiteboard-enterprise-backend:local'
];
const waitReady=async container=>{
  for(let attempt=0;attempt<60;attempt++){
    try{
      docker('exec',container,'wget','-qO-','http://127.0.0.1:8090/readyz');
      return;
    }catch{await pause(250);}
  }
  throw new Error('worker did not become ready: '+docker('logs',container));
};

const compose=readFileSync(resolve(root,'docker-compose.prod.yml'),'utf8');
const workerCompose=compose.match(/\n  feature-worker:\n([\s\S]*?)\n  collab:/)?.[1] ?? '';
assert.doesNotMatch(workerCompose,/\benv_file:/,'worker must not inherit unrelated production secrets');
assert.match(workerCompose,/DATABASE_URL:\s*\$\{DATABASE_URL:\?database required\}/,'worker must receive only its explicit database credential');

try{
  docker('network','create',network);
  const db=run('db',['--network-alias','db','-e','POSTGRES_PASSWORD=fixture','-e','POSTGRES_USER=fixture','-e','POSTGRES_DB=packaging_test','postgres:16-alpine']);
  for(let attempt=0;attempt<60;attempt++){
    try{docker('exec',db,'pg_isready','-U','fixture','-d','packaging_test');break;}
    catch(error){if(attempt===59)throw error;await pause(250);}
  }
  const migrate=run('migrate',['-v',root+'/src/backend/migrations:/migrations:ro','migrate/migrate:v4.17.1','-path=/migrations','-database=postgres://fixture:fixture@db:5432/packaging_test?sslmode=disable','up']);
  for(let attempt=0;attempt<60;attempt++){
    const state=docker('inspect','-f','{{.State.Status}} {{.State.ExitCode}}',migrate);
    if(state.startsWith('exited ')){
      assert.equal(state,'exited 0',docker('logs',migrate));
      break;
    }
    if(attempt===59)throw new Error('migration did not finish');
    await pause(250);
  }
  const first=run('worker-a',workerArgs('fixture-a'));
  await waitReady(first);
  assert.notEqual(docker('exec',first,'id','-u'),'0');
  const firstEnv=JSON.parse(docker('inspect','-f','{{json .Config.Env}}',first));
  assert.equal(firstEnv.some(value=>/^(JWT|TURN|AI|COLLAB|AUTHORITY|MUTATION|S3)_/.test(value)),false,'worker received an unrelated secret');
  docker('exec',db,'psql','-U','fixture','-d','packaging_test','-v','ON_ERROR_STOP=1','-c',
    "INSERT INTO feature_jobs(kind,payload,available_at,lease_owner,lease_expires_at,attempts) VALUES ('fixture.unknown','{}',now()-interval '2 minutes','fixture-a',now()+interval '2 seconds',1)");
  docker('rm','-f',first);
  containers.splice(containers.indexOf(first),1);

  const replacement=run('worker-b',workerArgs('fixture-b'));
  await waitReady(replacement);
  let deadLettered=docker('exec',db,'psql','-U','fixture','-d','packaging_test','-Atc',"SELECT count(*) FROM feature_jobs WHERE dead_lettered_at IS NOT NULL");
  assert.equal(deadLettered,'0','replacement stole an unexpired lease from the stopped worker');
  for(let attempt=0;attempt<40;attempt++){
    deadLettered=docker('exec',db,'psql','-U','fixture','-d','packaging_test','-Atc',"SELECT count(*) FROM feature_jobs WHERE dead_lettered_at IS NOT NULL");
    if(deadLettered==='1')break;
    await pause(250);
  }
  assert.equal(deadLettered,'1','expired lease was not resumed after worker restart');
  console.log(JSON.stringify({pass:true,checks:['nonroot read-only worker','least-privilege environment','database readiness','process restart','unexpired lease exclusion','expired lease recovery','unknown-kind safe dead letter']}));
}finally{
  for(const container of containers.reverse()){
    try{docker('rm','-f',container);}catch{}
  }
  try{docker('network','rm',network);}catch{}
}
