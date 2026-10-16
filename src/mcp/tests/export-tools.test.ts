import {test,expect} from 'vitest';
import {ExportService} from '../src/export-tools.js';
const ref={workspaceId:'w',sceneId:'s'};
test('JSON grant is scoped and revoked principal cannot download or create',async()=>{
 let revoked=false;const authority={principalId:'agent:k',async projection(){if(revoked)throw new Error('ACCESS_DENIED');return {...ref,sequence:'9007199254740993',content:{elements:[],appState:{},files:{}}};},async asset(){throw new Error('unexpected asset');}};
 const service=new ExportService();const grant=await service.exportScene(authority,ref,'json');expect(grant.mimeType).toBe('application/json');expect(grant.revision).toBe('9007199254740993');expect(new Date(grant.expiresAt).getTime()-Date.now()).toBeLessThanOrEqual(300000);
 expect(JSON.parse((await service.download(authority,grant.id)).bytes.toString()).type).toBe('excalidraw');
 await expect(service.download({...authority,principalId:'agent:other'},grant.id)).rejects.toMatchObject({code:'ARTIFACT_NOT_FOUND'});
 revoked=true;await expect(service.download(authority,grant.id)).rejects.toThrow('ACCESS_DENIED');await expect(service.exportScene(authority,ref,'json')).rejects.toThrow('ACCESS_DENIED');
});

test('artifact cache expires and does not issue a grant after rendering-time revocation',async()=>{
 let time=1000,calls=0;const authority={principalId:'agent:k',async projection(){calls++;if(calls===2)throw new Error('ACCESS_DENIED');return {...ref,sequence:'0',content:{elements:[],appState:{},files:{}}};},async asset(){throw new Error('unexpected');}};
 const service=new ExportService(undefined,()=>time);await expect(service.exportScene(authority,ref,'json')).rejects.toThrow('ACCESS_DENIED');calls=10;const grant=await service.exportScene(authority,ref,'json');time+=300001;await expect(service.download(authority,grant.id)).rejects.toMatchObject({code:'ARTIFACT_NOT_FOUND'});
});


test('rejects artifacts above 20 MiB without issuing a grant',async()=>{
 const authority={principalId:'agent:k',async projection(){return {...ref,sequence:'0',content:{elements:[],appState:{},files:{}}};},async asset(){throw new Error('unexpected');}};
 const renderer={async render(){return {mimeType:'image/png',bytes:Buffer.alloc(20*1024*1024+1)};}};
 await expect(new ExportService(renderer as any).exportScene(authority,ref,'png')).rejects.toMatchObject({code:'RENDER_LIMIT'});
});
