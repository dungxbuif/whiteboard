import type {SceneContent} from '@collab/contracts';

export const MAX_VERSION_BYTES=25*1024*1024;

export type CaptureVersionInput={
 requestId:string;
 expectedSequence:string;
 label?:string;
 pinned:boolean;
};

export type VersionCreator={kind:'user'|'system';id:string};
export type VersionSummary={
 id:string;
 documentId:string;
 sequence:string;
 hash:string;
 label?:string;
 pinned:boolean;
 kind:'manual'|'automatic';
 metadataRevision:string;
 createdAt:string;
 updatedAt:string;
 creator:VersionCreator;
};

export function canonicalVersionPayload(input:CaptureVersionInput):Uint8Array{
 return new TextEncoder().encode(JSON.stringify({requestId:input.requestId,expectedSequence:input.expectedSequence,...(input.label===undefined?{}:{label:input.label}),pinned:input.pinned}));
}

export function canonicalSceneBytes(content:SceneContent):Uint8Array{
 return new TextEncoder().encode(JSON.stringify(content));
}

export function referencedAssetIDs(content:SceneContent):string[]{
 const found=new Set<string>();
 for(const file of Object.values(content.files??{}) as Array<Record<string,unknown>>){
  if(typeof file.assetId==='string'&&file.assetId)found.add(file.assetId);
 }
 return [...found].sort();
}
