import {randomUUID} from 'node:crypto';
import {expect,test} from './fixtures';

const rectangle=(id='source-shape')=>({id,type:'rectangle',x:0,y:0,width:100,height:100,angle:0,strokeColor:'#111111',backgroundColor:'transparent',fillStyle:'solid',strokeWidth:1,strokeStyle:'solid',roughness:1,opacity:100,groupIds:[],frameId:null,roundness:null,seed:1,version:1,versionNonce:1,index:null,isDeleted:false,boundElements:null,updated:1,link:null,locked:false});

test('import, insert, reload-safe receipt, version and export preserve workspace authorization',async({stack})=>{
 const scene=await stack.scene('Library target');await stack.pool.query('UPDATE workspace_feature_policies SET libraries_enabled=true,versions_enabled=true WHERE workspace_id=$1',[stack.owner.workspaceId]);
 const content={type:'excalidrawlib',version:2,source:'https://whiteboard.example/libraries/export',libraryItems:[{id:'native-item',status:'published',created:1,name:'Flow',elements:[rectangle()],files:{}}]};
 const imported=await stack.json(`/workspaces/${stack.owner.workspaceId}/libraries/import`,stack.owner.token,stack.owner.workspaceId,'POST',{requestId:randomUUID(),name:'Flow',content});expect(imported.revision.revision).toBe('1');
 expect((await stack.request(`/workspaces/${stack.owner.workspaceId}/libraries/import`,stack.editor.token,stack.owner.workspaceId,'POST',{requestId:randomUUID(),name:'Denied',content})).status).toBe(403);
 const listed=await stack.json(`/workspaces/${stack.owner.workspaceId}/libraries`,stack.viewer.token,stack.owner.workspaceId);expect(listed.items.map((item:any)=>item.name)).toContain('Flow');
 const requestId=randomUUID(),insertion={requestId,libraryItemId:imported.item.id,nativeItemId:'native-item',revision:1,expectedSequence:'0'};const first=await stack.json(`/workspaces/${stack.owner.workspaceId}/scenes/${scene.id}/library-insertions`,stack.owner.token,stack.owner.workspaceId,'POST',insertion);const replay=await stack.json(`/workspaces/${stack.owner.workspaceId}/scenes/${scene.id}/library-insertions`,stack.owner.token,stack.owner.workspaceId,'POST',insertion);expect(replay).toEqual(first);expect(first.elementIds).toHaveLength(1);expect(first.elementIds[0]).not.toBe('source-shape');
 const version=await stack.json(`/scenes/${scene.id}/versions`,stack.owner.token,stack.owner.workspaceId,'POST',{requestId:randomUUID(),expectedSequence:first.sequence,label:'After library insert',pinned:true});expect(version.sequence).toBe(first.sequence);
 const exported=await stack.json(`/workspaces/${stack.owner.workspaceId}/libraries/${imported.item.id}/revisions/1/export`,stack.owner.token,stack.owner.workspaceId);expect(exported).toEqual(content);
 expect((await stack.request(`/workspaces/${stack.owner.workspaceId}/libraries`,stack.outsider.token,stack.owner.workspaceId)).status).toBeGreaterThanOrEqual(403);
 await stack.pool.query('DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2',[stack.owner.workspaceId,stack.editor.id]);expect((await stack.request(`/workspaces/${stack.owner.workspaceId}/scenes/${scene.id}/library-insertions`,stack.editor.token,stack.owner.workspaceId,'POST',{...insertion,requestId:randomUUID(),expectedSequence:first.sequence})).status).toBeGreaterThanOrEqual(403);
});
