import {useCallback,useEffect,useState} from 'react';
import {Download,LibraryBig,Plus,Upload} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {libraryClient,type LibraryClient,type LibraryInsertResult,type LibraryItem} from '@/lib/library-api';

export function LibraryPanel({workspaceId,sceneId,currentSequence,canManage,api=libraryClient,onInserted}:{workspaceId:string;sceneId:string;currentSequence:string;canManage:boolean;api?:LibraryClient;onInserted:(result:LibraryInsertResult)=>void}){
 const [items,setItems]=useState<LibraryItem[]>([]),[filter,setFilter]=useState(''),[error,setError]=useState(''),[busy,setBusy]=useState('');
 const load=useCallback(async()=>{try{setError('');setItems((await api.list(workspaceId,filter.trim())).items)}catch{setError('Workspace libraries could not be loaded.')}},[api,filter,workspaceId]);
 useEffect(()=>{const timer=setTimeout(()=>void load(),filter?150:0);return()=>clearTimeout(timer)},[filter,load]);
 const insert=async(item:LibraryItem)=>{setBusy(item.id);try{onInserted(await api.insert(workspaceId,sceneId,item.id,Number(item.currentRevision),currentSequence))}catch{setError('Library item could not be inserted.')}finally{setBusy('')}};
 const importFile=async(file:File)=>{if(file.size>25*1024*1024){setError('Library files must be 25 MiB or smaller.');return}setBusy('import');try{const content=JSON.parse(await file.text());await api.import(workspaceId,file.name.replace(/\.excalidrawlib$/i,'')||'Imported library',content);await load()}catch{setError('Choose a valid native .excalidrawlib file.')}finally{setBusy('')}};
 return <div className="grid gap-3">
  <div className="flex items-center gap-2"><Input aria-label="Search workspace libraries" onChange={event=>setFilter(event.target.value)} placeholder="Search workspace libraries" value={filter}/>{canManage?<Button asChild size="icon" title="Import native library" variant="outline"><label><Upload/><input accept=".excalidrawlib,application/json" className="sr-only" disabled={!!busy} onChange={event=>{const file=event.target.files?.[0];if(file)void importFile(file)}} type="file"/></label></Button>:null}</div>
  {error?<p className="text-sm text-destructive" role="alert">{error}</p>:null}
  <div className="grid gap-2">{items.map(item=><article className="flex items-center gap-3 rounded-lg border bg-background p-3" key={item.id}><LibraryBig className="h-5 w-5 text-primary"/><div className="min-w-0 flex-1"><p className="truncate font-medium">{item.name}</p><p className="text-xs text-muted-foreground">Revision {item.currentRevision}</p></div><Button aria-label={`Insert ${item.name}`} disabled={!!busy} onClick={()=>void insert(item)} size="sm"><Plus/> Insert</Button><Button aria-label={`Export ${item.name}`} onClick={async()=>{const content=await api.export(workspaceId,item.id,Number(item.currentRevision));const blob=new Blob([JSON.stringify(content)],{type:'application/json'}),url=URL.createObjectURL(blob),anchor=document.createElement('a');anchor.href=url;anchor.download=`${item.name}.excalidrawlib`;anchor.click();URL.revokeObjectURL(url)}} size="icon" variant="ghost"><Download/></Button></article>)}</div>
  {!items.length&&!error?<p className="py-6 text-center text-sm text-muted-foreground">No workspace libraries found.</p>:null}
 </div>;
}
