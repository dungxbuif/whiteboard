import {it,expect} from 'vitest';
import * as Y from 'yjs';
import {createSceneDoc} from './document.js';
import {ValidationPool,runValidationJob,VALIDATION_DEADLINE_MS,VALIDATION_RESOURCE_LIMITS} from './validation-job.js';

it('terminates a noncooperative worker at the two-second deadline without changing live state',async()=>{
  const current=createSceneDoc({elements:[],appState:{},files:{}}),snapshot=Y.encodeStateAsUpdate(current);
  const start=performance.now();
  await expect(runValidationJob({snapshot,update:new Uint8Array()},new URL('./test-fixtures/hung-worker.mjs',import.meta.url))).rejects.toThrow(/deadline/);
  expect(VALIDATION_DEADLINE_MS).toBe(2000);
  expect(performance.now()-start).toBeGreaterThanOrEqual(1900);
  expect(performance.now()-start).toBeLessThan(4000);
  expect(Y.encodeStateAsUpdate(current)).toEqual(snapshot);
},5000);

it('terminates worker heap exhaustion and leaves the live document unchanged',async()=>{
  const current=createSceneDoc({elements:[],appState:{},files:{}}),snapshot=Y.encodeStateAsUpdate(current);
  expect(VALIDATION_RESOURCE_LIMITS).toEqual({maxOldGenerationSizeMb:256,maxYoungGenerationSizeMb:16,stackSizeMb:4});
  await expect(runValidationJob({snapshot,update:new Uint8Array()},new URL('./test-fixtures/memory-worker.mjs',import.meta.url))).rejects.toThrow(/resource\/error.*memory/i);
  expect(Y.encodeStateAsUpdate(current)).toEqual(snapshot);
},5000);

it('reuses a bounded worker and replaces it after a deadline; queue stays bounded',async()=>{
 const pool=new ValidationPool(new URL('./test-fixtures/reusable-worker.mjs',import.meta.url),1,1);
 const input={snapshot:new Uint8Array(),update:new Uint8Array()};
 try {
  expect((await pool.run(input)).stateVector[0]).toBe(1);
  expect((await pool.run(input)).stateVector[0]).toBe(2);
  const hung=pool.run({...input,update:new Uint8Array([1])});
  const queued=pool.run(input);
  const failures=Promise.allSettled([hung,queued]);
  await expect(pool.run(input)).rejects.toThrow(/queue/);
  expect((await failures).every(r=>r.status==='rejected')).toBe(true);
  expect((await pool.run(input)).stateVector[0]).toBe(1);
 }finally{await pool.close();}
},6000);
