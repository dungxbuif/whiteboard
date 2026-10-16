import {it,expect} from 'vitest';
import * as Y from 'yjs';
import {createSceneDoc} from './document.js';
import {runValidationJob,VALIDATION_DEADLINE_MS,VALIDATION_RESOURCE_LIMITS} from './validation-job.js';

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
