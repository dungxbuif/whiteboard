import type {ExcalidrawImperativeAPI,LibraryItems} from '@excalidraw/excalidraw/types';
import {libraryClient,type LibraryClient} from '@/lib/library-api';

type NativeLibrary={type:'excalidrawlib';version:2;libraryItems:LibraryItems};
function native(value:unknown):value is NativeLibrary{return !!value&&typeof value==='object'&&(value as NativeLibrary).type==='excalidrawlib'&&(value as NativeLibrary).version===2&&Array.isArray((value as NativeLibrary).libraryItems)}

/** Loads the authorized workspace view into Excalidraw without treating personal changes as publications. */
export async function updateWorkspaceLibrary(api:Pick<ExcalidrawImperativeAPI,'updateLibrary'>,workspaceId:string,client:LibraryClient=libraryClient):Promise<void>{
 const page=await client.list(workspaceId),items:LibraryItems[number][]=[];
 for(const summary of page.items){const revision=await client.revision(workspaceId,summary.id,Number(summary.currentRevision));if(!native(revision.content))throw new Error('Invalid workspace library revision');items.push(...revision.content.libraryItems)}
 await api.updateLibrary({libraryItems:items,merge:true,prompt:false});
}

/** Deliberately local: workspace publication happens only through the managed panel. */
export async function onPersonalLibraryChange(_items:LibraryItems):Promise<void>{}
