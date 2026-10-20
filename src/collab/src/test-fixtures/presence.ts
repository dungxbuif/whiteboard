import { EventEmitter, once } from 'node:events';
import { randomUUID } from 'node:crypto';
import WebSocket, { WebSocketServer } from 'ws';
import * as Y from 'yjs';
import { Rooms } from '../rooms.js';
import { CollabError } from '../auth.js';
import type { Persistence } from '../persistence.js';
export type Message = {type:string;requestId:string;payload:any};
export async function roomFixture(options: {heartbeatIntervalMs?:number} = {}) {
 const hooks:{beforeRead?:(principalId:string)=>Promise<void>;access?:string;featureEnabled:boolean;featureError?:CollabError;featureChecks:string[];featurePrincipals:string[];readKinds:string[]}={featureEnabled:true,featureChecks:[],featurePrincipals:[],readKinds:[]}; const source = new EventEmitter(); const writes: unknown[]=[]; const denied=new Set<string>();
 const store=Object.assign(source, {
  authority:{verifyRoom:async(token:string)=>JSON.parse(token),verifyPresentation:async(token:string)=>JSON.parse(token),requireFeature:async(token:string,_ref:unknown,action:string)=>{hooks.featureChecks.push(action);hooks.featurePrincipals.push(JSON.parse(token).principalId);if(hooks.featureError)throw hooks.featureError;if(!hooks.featureEnabled)throw new CollabError('FORBIDDEN');}},
  read:async(_ref:unknown,principal:{principalId:string})=>{hooks.readKinds.push('room:'+principal.principalId);await hooks.beforeRead?.(principal.principalId);if(denied.has(principal.principalId))throw new CollabError('FORBIDDEN');return {doc:new Y.Doc(),documentId:'document',sequence:'0',stateVector:'AA==',access:hooks.access??'edit'};},
  readPresentation:async(_ref:unknown,principal:{principalId:string})=>{hooks.readKinds.push('presentation:'+principal.principalId);await hooks.beforeRead?.(principal.principalId);if(denied.has(principal.principalId))throw new CollabError('FORBIDDEN');return {doc:new Y.Doc(),documentId:'document',sequence:'0',stateVector:'AA==',access:'view'};},
  appendUpdate:async(_ref:unknown,_principal:unknown,id:string,update:Uint8Array)=>{writes.push(update);return {requestId:id,sequence:String(writes.length),stateVector:'AA=='};}
 });
 const rooms=new Rooms(store as unknown as Persistence,{updatesPerMinute:120,maxBufferedBytes:2*1024*1024,...options});
 const server=new WebSocketServer({host:'127.0.0.1',port:0});server.on('connection',socket=>rooms.attach(socket,'127.0.0.1'));await once(server,'listening');
 const sockets:WebSocket[]=[];const ref={workspaceId:'workspace',sceneId:'scene'};
 async function connect(principalId='user',sceneId=ref.sceneId,autoPong=true){
  const socket=new WebSocket('ws://127.0.0.1:'+(server.address() as {port:number}).port,{autoPong});sockets.push(socket);
  const messages:Message[]=[];socket.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});await once(socket,'open');
  const send=(type:string,payload:unknown={},requestId=randomUUID())=>{socket.send(JSON.stringify({type,payload,requestId}));return requestId;};
  const wait=async(type:string,id?:string)=>{const end=Date.now()+1500;while(Date.now()<end){const message=messages.find(m=>m.type===type&&(!id||m.requestId===id));if(message)return message;await new Promise(resolve=>setTimeout(resolve,5));}throw new Error('Missing '+type+' '+id+'; got '+JSON.stringify(messages));};
  const token=JSON.stringify({...ref,sceneId,principalId,principalType:'user',exp:Math.floor(Date.now()/1000)+600});
  send('auth',{token});const ready=await wait('ready');await wait('sync_chunk');
  return {socket,messages,send,wait,connectionId:ready.payload.connectionId,principalId};
 }
 async function connectPresentation(token:Record<string,unknown>){
  const socket=new WebSocket('ws://127.0.0.1:'+(server.address() as {port:number}).port);sockets.push(socket);const messages:Message[]=[];socket.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});await once(socket,'open');const send=(type:string,payload:unknown={},requestId=randomUUID())=>{socket.send(JSON.stringify({type,payload,requestId}));return requestId;};const wait=async(type:string,id?:string)=>{const end=Date.now()+1500;while(Date.now()<end){const message=messages.find(m=>m.type===type&&(!id||m.requestId===id));if(message)return message;await new Promise(resolve=>setTimeout(resolve,5));}throw new Error('Missing '+type+' '+id+'; got '+JSON.stringify(messages));};const requestId=send('presentation_auth',{token:JSON.stringify(token)});const waiting=await wait('presentation_waiting',requestId);return {socket,messages,send,wait,connectionId:waiting.payload.connectionId};
 }
 return {rooms,connect,connectPresentation,writes,denied,hooks,async close(){for(const socket of sockets)socket.terminate();await rooms.close();await new Promise<void>(resolve=>server.close(()=>resolve()));}};
}
