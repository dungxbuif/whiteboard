import {assertSerializable,validateSceneContent,type AssetReference,type NativeElement,type SceneContent} from '@collab/contracts';

const MAX_ITEMS=500;
const MAX_BYTES=25*1024*1024;
const RASTER_TYPES=new Set(['image/png','image/jpeg','image/gif','image/webp','image/avif']);
const SAFE_ID=/^[A-Za-z0-9_-]{1,256}$/;

export type LibraryItem={
 id:string;
 status:'published'|'unpublished';
 created:number;
 name?:string;
 elements:NativeElement[];
 files:Record<string,AssetReference>;
 presentationOrder?:string[];
};
export type NativeLibrary={type:'excalidrawlib';version:2;source?:string;libraryItems:LibraryItem[]};
export type IdAllocator={element:(oldId:string)=>string;group:(oldId:string)=>string;file:(oldId:string)=>string};

function plain(value:unknown):value is Record<string,unknown>{return !!value&&typeof value==='object'&&!Array.isArray(value)&&Object.getPrototypeOf(value)===Object.prototype;}
function keys(value:Record<string,unknown>,allowed:string[]):void{if(Object.keys(value).some(key=>!allowed.includes(key)))throw new Error('Unsupported library field');}
function source(value:unknown):string|undefined{
 if(value===undefined)return undefined;
 if(typeof value!=='string'||value.length>4096)throw new Error('Invalid library source');
 let url:URL;try{url=new URL(value);}catch{throw new Error('Invalid library source');}
 if(!['http:','https:'].includes(url.protocol)||url.username||url.password)throw new Error('Invalid library source');
 return value;
}
function item(value:unknown,index:number):LibraryItem{
 if(!plain(value))throw new Error(`Invalid library item ${index}`);
 keys(value,['id','status','created','name','elements','files','presentationOrder']);
 if(typeof value.id!=='string'||!SAFE_ID.test(value.id)||!['published','unpublished'].includes(String(value.status))||!Number.isSafeInteger(value.created)||Number(value.created)<0||value.name!==undefined&&(typeof value.name!=='string'||value.name.length>1024))throw new Error(`Invalid library item ${index}`);
 const inputFiles=value.files===undefined?{}:value.files;
 if(!plain(inputFiles))throw new Error(`Invalid library files ${index}`);
 for(const file of Object.values(inputFiles)){
  if(!plain(file)||typeof file.mimeType!=='string'||!RASTER_TYPES.has(file.mimeType))throw new Error('Library assets must be supported raster images');
 }
 const scene=validateSceneContent({elements:value.elements,appState:{},files:inputFiles,...(value.presentationOrder===undefined?{}:{presentationOrder:value.presentationOrder})});
 return {id:value.id,status:value.status as LibraryItem['status'],created:value.created as number,...(value.name===undefined?{}:{name:value.name as string}),elements:scene.elements,files:scene.files,...(scene.presentationOrder?{presentationOrder:scene.presentationOrder}:{})};
}

export function validateLibrary(input:unknown):NativeLibrary{
 assertSerializable(input);
 let bytes:number;try{bytes=new TextEncoder().encode(JSON.stringify(input)).length;}catch{throw new Error('Invalid library JSON');}
 if(bytes>MAX_BYTES)throw new Error('Library exceeds the 25 MiB limit');
 if(!plain(input)){throw new Error('Invalid native library');}
 keys(input,['type','version','source','libraryItems']);
 if(input.type!=='excalidrawlib')throw new Error('Unsupported library type');
 if(input.version!==2)throw new Error('Unsupported library version; expected version 2');
 if(!Array.isArray(input.libraryItems)||input.libraryItems.length>MAX_ITEMS)throw new Error('Library supports at most 500 items');
 const libraryItems=input.libraryItems.map(item);
 const ids=new Set<string>();for(const entry of libraryItems){if(ids.has(entry.id))throw new Error('Duplicate library item ID');ids.add(entry.id);}
 const origin=source(input.source);
 return {type:'excalidrawlib',version:2,...(origin?{source:origin}:{}),libraryItems};
}

function allocated(kind:string,oldId:string,value:string,used:Set<string>):string{
 if(!SAFE_ID.test(value)||value===oldId||used.has(value))throw new Error(`ID allocator returned an invalid or reused ${kind} ID`);
 used.add(value);return value;
}

export function remapLibraryItem(input:LibraryItem,ids:IdAllocator):SceneContent{
 const validated=item(input,0),used=new Set<string>();
 const elements=new Map(validated.elements.map(element=>[element.id,allocated('element',element.id,ids.element(element.id),used)]));
 const groups=new Map<string,string>();for(const element of validated.elements)for(const group of element.groupIds)if(!groups.has(group))groups.set(group,allocated('group',group,ids.group(group),used));
 const files=new Map(Object.keys(validated.files).map(file=>[file,allocated('file',file,ids.file(file),used)]));
 const reference=(id:string|null)=>id===null?null:elements.get(id)??(()=>{throw new Error(`Missing element reference ${id}`);})();
 const output=validated.elements.map(element=>{
  const next:any=structuredClone(element);next.id=elements.get(element.id)!;next.groupIds=element.groupIds.map(id=>groups.get(id)!);next.frameId=reference(element.frameId);
  if(element.boundElements)next.boundElements=element.boundElements.map(bound=>({...bound,id:reference(bound.id)}));
  if(element.type==='text')next.containerId=reference(element.containerId);
  if(element.type==='arrow'||element.type==='line'){
   if(element.startBinding)next.startBinding={...element.startBinding,elementId:reference(element.startBinding.elementId)};
   if(element.endBinding)next.endBinding={...element.endBinding,elementId:reference(element.endBinding.elementId)};
  }
  if(element.type==='image'&&element.fileId)next.fileId=files.get(element.fileId)!;
  return next as NativeElement;
 });
 const remappedFiles:Record<string,AssetReference>={};for(const [oldId,file] of Object.entries(validated.files)){const id=files.get(oldId)!;remappedFiles[id]={...structuredClone(file),id};}
 return validateSceneContent({elements:output,appState:{},files:remappedFiles,...(validated.presentationOrder?{presentationOrder:validated.presentationOrder.map(id=>reference(id)!)}:{})});
}
