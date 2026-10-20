import {useEffect,useState} from 'react';
import {Button} from '@/components/ui/button';
import {Card,CardContent,CardDescription,CardHeader,CardTitle} from '@/components/ui/card';
import {Input} from '@/components/ui/input';
import {scimClient,type SCIMClient,type SCIMTokenStatus} from '@/lib/scim-api';

export function SCIMSettings({workspaceId,api=scimClient}:{workspaceId:string;api?:SCIMClient}){
 const[status,setStatus]=useState<SCIMTokenStatus|null>(null);const[token,setToken]=useState('');const[busy,setBusy]=useState(false);const[message,setMessage]=useState('');const[error,setError]=useState('');
 useEffect(()=>{let active=true;api.status(workspaceId).then(value=>{if(active)setStatus(value)}).catch(()=>{if(active)setError('Could not load SCIM settings. Owner access is required.')});return()=>{active=false}},[api,workspaceId]);
 async function rotate(){setBusy(true);setError('');setMessage('');try{const value=await api.rotate(workspaceId);setToken(value.token);setStatus({configured:true,prefix:value.prefix,createdAt:value.createdAt})}catch{setError('Could not rotate the SCIM token.')}finally{setBusy(false)}}
 async function revoke(){setBusy(true);setError('');setMessage('');try{await api.revoke(workspaceId);setToken('');setStatus({configured:false});setMessage('SCIM token revoked.')}catch{setError('Could not revoke the SCIM token.')}finally{setBusy(false)}}
 async function copy(){try{await navigator.clipboard.writeText(token);setMessage('SCIM token copied.')}catch{setError('Copy was blocked. Select and copy the token manually.')}}
 const endpoint=`${window.location.origin}/api/v1/scim/v2`;
 return <Card><CardHeader><CardTitle>SCIM provisioning</CardTitle><CardDescription>Provision workspace viewers and editors from a SCIM 2.0 identity provider. SCIM can never grant owner access.</CardDescription></CardHeader><CardContent className="space-y-5">
  <label className="grid gap-2 text-sm font-medium">SCIM base URL<Input aria-label="SCIM base URL" readOnly value={endpoint}/></label>
  {status?.configured?<p className="text-sm">Active token: <code>{status.prefix}…</code>{status.createdAt?` created ${new Date(status.createdAt).toLocaleString()}`:''}</p>:<p className="text-sm text-muted-foreground">No SCIM token is active.</p>}
  {token?<section aria-label="New SCIM token" className="grid gap-2 rounded-lg border p-4"><p className="text-sm font-medium">Copy this token now. It is shown only once.</p><Input aria-label="SCIM token" readOnly value={token}/><Button onClick={()=>void copy()} variant="outline">Copy token</Button></section>:null}
  <p className="text-xs text-muted-foreground">Disabling a SCIM user ends workspace SSO sessions and removes only SCIM-managed access. Manual access and other workspaces are preserved.</p>
  {message?<p role="status" className="text-sm">{message}</p>:null}{error?<p role="alert" className="text-sm text-destructive">{error}</p>:null}
  <div className="flex flex-wrap gap-2"><Button disabled={busy} onClick={()=>void rotate()}>{status?.configured?'Rotate token':'Create token'}</Button>{status?.configured?<Button disabled={busy} onClick={()=>void revoke()} variant="destructive">Revoke token</Button>:null}</div>
 </CardContent></Card>
}
