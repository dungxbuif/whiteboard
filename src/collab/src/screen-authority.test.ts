import {createServer} from 'node:http';
import {once} from 'node:events';
import {expect,it} from 'vitest';
import {requireScreenFeature} from './screen-authority.js';

it('binds screen publish to the current room token and private authority',async()=>{
 let request:Record<string,unknown>|undefined;
 const server=createServer((incoming,response)=>{expect(incoming.url).toBe('/internal/authority/features');expect(incoming.headers.authorization).toBe('Bearer '+ 's'.repeat(32));const chunks:Buffer[]=[];incoming.on('data',chunk=>chunks.push(chunk));incoming.on('end',()=>{request=JSON.parse(Buffer.concat(chunks).toString());response.writeHead(204).end();});});
 server.listen(0,'127.0.0.1');await once(server,'listening');
 try{const address=server.address();await requireScreenFeature({config:{url:`http://127.0.0.1:${typeof address==='object'&&address?address.port:0}`,serviceSecret:'s'.repeat(32)}},'room-token',{workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333'});expect(request).toEqual({roomToken:'room-token',workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333',action:'screen:publish'});}finally{server.close();await once(server,'close');}
});

it('fails closed when Go denies the current feature decision',async()=>{
 const server=createServer((_incoming,response)=>response.writeHead(403).end());server.listen(0,'127.0.0.1');await once(server,'listening');
 try{const address=server.address();await expect(requireScreenFeature({config:{url:`http://127.0.0.1:${typeof address==='object'&&address?address.port:0}`,serviceSecret:'s'.repeat(32)}},'room-token',{workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333'})).rejects.toMatchObject({code:'FORBIDDEN',retryable:false});}finally{server.close();await once(server,'close');}
});
