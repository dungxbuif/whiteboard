import {parentPort,workerData} from 'node:worker_threads';
import * as Y from 'yjs';
import {createDecoder,hasContent} from 'lib0/decoding';
import {materializeScene} from './document.js';
import type {ValidationInput} from './validation-job.js';

function validate(input:ValidationInput){
const doc=new Y.Doc({gc:false});
try {
  const {snapshot,update}=input;
  Y.applyUpdate(doc,snapshot);
  // Capture map identity as well as immutable values; replacing an entire map
  // would destroy independent-field CRDT semantics even if its ID/type matched.
  const original=new Map([...doc.getMap<Y.Map<unknown>>('elements')].map(([id,map])=>[id,{map,type:map.get('type')}]));
  const decoder=createDecoder(update);
  Y.readUpdate(decoder,doc);
  if(hasContent(decoder))throw new Error('Trailing bytes after Yjs update');
  const elements=doc.getMap<Y.Map<unknown>>('elements');
  for(const [id,previous] of original){
    const element=elements.get(id);
    if(!element||element!==previous.map||element.get('id')!==id||element.get('type')!==previous.type)throw new Error('Immutable element ID/type/map changed or tombstone removed');
  }
  const content=materializeScene(doc);
  parentPort!.postMessage({ok:true,result:{update,content,stateVector:Y.encodeStateVector(doc)}});
} catch(error){
  parentPort!.postMessage({ok:false,error:error instanceof Error?error.message:'Invalid candidate update'});
} finally {doc.destroy();}

}
parentPort!.on('message',validate);
validate(workerData as ValidationInput);
