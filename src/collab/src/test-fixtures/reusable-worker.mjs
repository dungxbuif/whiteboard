import {parentPort,workerData} from 'node:worker_threads';
let count=0;function run(input){if(input.update[0]===1)while(true){};parentPort.postMessage({ok:true,result:{update:input.update,content:{elements:[],appState:{},files:{}},stateVector:new Uint8Array([++count])}});}
parentPort.on('message',run);run(workerData);
