import {createServer} from 'node:http';
const keyId='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
createServer((req,res)=>{
 res.setHeader('Content-Type','application/json');
 if(req.url==='/healthz'){res.end('{}');return;}
 if(req.headers.authorization!=='Bearer packaging-fixture'){res.writeHead(401).end('{}');return;}
 if(req.url==='/api/v1/agent/auth'){res.end(JSON.stringify({principalId:'agent:'+keyId,keyId}));return;}
 if(req.url.endsWith('/export-projection')){res.end(JSON.stringify({workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333',sequence:'0',content:{elements:[],appState:{},files:{}}}));return;}
 if(req.url.startsWith('/api/v1/agent/workspaces')){res.end(JSON.stringify({items:[{id:'22222222-2222-4222-8222-222222222222',name:'Packaging'}]}));return;}
 res.writeHead(404).end('{}');
}).listen(8080,'0.0.0.0',()=>import('/app/mcp/dist/http.js'));
