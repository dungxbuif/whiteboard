import {expect,it} from 'vitest';
import {once} from 'node:events';
import WebSocket from 'ws';
import * as Y from 'yjs';
import {encodeFrame} from '@collab/contracts';
import {fixture,signed} from './fixture.js';
import {Persistence} from '../src/persistence.js';
import {startServer} from '../src/server.js';

it('full voice capacity leaves durable editing available and ephemeral controls never enter storage',async()=>{
 const f=await fixture(),sockets:WebSocket[]=[];
 let server:Awaited<ReturnType<typeof startServer>>|undefined;
 try {
  const store=new Persistence(f.pool,f.authority,{promoteV2:true});
  server=await startServer(store,{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4*1024*1024});
  const now=Math.floor(Date.now()/1000),token=await signed({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'presence',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:'2'});
  async function connect(){
   const socket=new WebSocket(server!.url+'/collab',{origin:'http://localhost:3000'});sockets.push(socket);
   const messages:any[]=[];socket.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});await once(socket,'open');
   const send=(type:string,payload:unknown={},id=crypto.randomUUID())=>{socket.send(JSON.stringify({type,payload,requestId:id}));return id;};
   const wait=async(type:string,id?:string)=>{const deadline=Date.now()+5000;while(Date.now()<deadline){const message=messages.find(m=>m.type===type&&(!id||m.requestId===id));if(message)return message;await new Promise(r=>setTimeout(r,10));}throw new Error('No '+type+' '+id);};
   send('auth',{token});const ready=await wait('ready');await wait('sync_chunk');return {socket,send,wait,id:ready.payload.connectionId};
  }
  const clients=[];for(let i=0;i<11;i++)clients.push(await connect());
  expect(new Set(clients.map(c=>c.id)).size).toBe(11);
  for(const client of clients.slice(0,10))expect((await client.wait('voice_admission',client.send('voice_join'))).payload.admitted).toBe(true);
  const editor=clients[10];expect((await editor.wait('voice_admission',editor.send('voice_join'))).payload).toMatchObject({admitted:false,reason:'VOICE_FULL'});
  await editor.wait('follow',editor.send('follow',{targetConnectionId:clients[0].id}));
  const camera=clients[0].send('viewport',{scrollX:123,scrollY:456,zoom:2});await editor.wait('viewport',camera);
  expect((await f.pool.query('SELECT count(*) FROM collab_updates')).rows[0].count).toBe('0');
  const state=await store.read(f.ref,f.principal),copy=new Y.Doc();Y.applyUpdate(copy,Y.encodeStateAsUpdate(state.doc));copy.getMap('appState').set('viewBackgroundColor','#abcdef');
  const bytes=Y.encodeStateAsUpdate(copy,Y.encodeStateVector(state.doc));state.doc.destroy();copy.destroy();
  editor.socket.send(encodeFrame({channel:3,updateId:'voice-full-edit',payload:bytes}));expect((await editor.wait('ack','voice-full-edit')).payload.sequence).toBe('1');
  expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('1');
  await store.compactScene(f.ref);const persisted=(await f.pool.query('SELECT content FROM scenes WHERE id=$1',[f.ref.sceneId])).rows[0].content;
  expect(persisted.appState.viewBackgroundColor).toBe('#abcdef');expect(JSON.stringify(persisted)).not.toMatch(/scrollX|scrollY|following|voiceJoined/);
  const closed=once(clients[0].socket,'close');clients[0].socket.close();await closed;await editor.wait('roster','server');
  expect((await editor.wait('voice_admission',editor.send('voice_join'))).payload.admitted).toBe(true);
 } finally {for(const socket of sockets)socket.terminate();await server?.close();await f.close();}
},20_000);
