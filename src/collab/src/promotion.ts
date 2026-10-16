import type {PoolClient} from 'pg';
import {validateSceneContent,type SceneRef,type SceneContent} from '@collab/contracts';
import {CollabError} from './auth.js';
const durableAppState=['viewBackgroundColor','gridSize','gridStep','gridModeEnabled','theme'];
/** Only legacy promotion may remove renderer-local fields. Original JSON is checkpointed atomically. */
export async function promoteLegacy(client:PoolClient,ref:SceneRef,raw:unknown):Promise<SceneContent>{
 try{
  if(!raw||typeof raw!=='object'||Array.isArray(raw))throw new Error('Legacy content must be an object');
  const original=raw as Record<string,any>;
  if(Object.keys(original).some(k=>!['elements','appState','files','presentationOrder'].includes(k)))throw new Error('Unknown legacy content field');
  const appState:Record<string,unknown>={};
  if(original.appState!==undefined&&(!original.appState||typeof original.appState!=='object'||Array.isArray(original.appState)))throw new Error('Invalid legacy app state');
  for(const key of durableAppState)if(original.appState?.[key]!==undefined)appState[key]=original.appState[key];
  const files:Record<string,unknown>=Object.create(null),rawFiles=original.files??{};
  if(typeof rawFiles!=='object'||Array.isArray(rawFiles))throw new Error('Invalid legacy files');
  const ids=Object.keys(rawFiles);
  if(ids.length){
   const rows=await client.query("SELECT id,file_id,mime_type FROM scene_assets WHERE workspace_id=$1 AND scene_id=$2 AND deleted_at IS NULL AND asset_kind='excalidraw_file' AND file_id=ANY($3::text[])",[ref.workspaceId,ref.sceneId,ids]);
   const assets=new Map(rows.rows.map(row=>[row.file_id,row]));
   for(const id of ids){
    const file=rawFiles[id],asset=assets.get(id);
    if(!file||typeof file!=='object'||Array.isArray(file)||file.id!==id||!asset||asset.mime_type!==file.mimeType||(file.assetId!==undefined&&file.assetId!==asset.id))throw new Error('Legacy asset is missing or outside this scene: '+id);
    if(Object.keys(file).some(key=>!['id','mimeType','created','dataURL','assetId','lastRetrieved','version'].includes(key)))throw new Error('Unknown legacy file field');
    files[id]={id,mimeType:file.mimeType,created:file.created,assetId:asset.id,...(file.lastRetrieved!==undefined?{lastRetrieved:file.lastRetrieved}:{}),...(file.version!==undefined?{version:file.version}:{})};
   }
  }
  return validateSceneContent({elements:original.elements??[],appState,files,...(original.presentationOrder!==undefined?{presentationOrder:original.presentationOrder}:{})});
 }catch(error){if(error instanceof CollabError)throw error;throw new CollabError('PROMOTION_REQUIRED','Legacy scene requires repair before promotion: '+(error instanceof Error?error.message:'invalid content'),409);}
}
