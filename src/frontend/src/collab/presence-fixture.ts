import type {Control,TransportState} from './transport';
export class PresenceTransportFixture {
 state:TransportState={status:'synced',canEdit:true,pending:0,recovery:false,connectionId:'a'};
 sent:Control[]=[];controls=new Set<(c:Control)=>void>();listeners=new Set<()=>void>();
 subscribe=(f:()=>void)=>{this.listeners.add(f);return()=>{this.listeners.delete(f);};};
 subscribeControl=(f:(c:Control)=>void)=>{this.controls.add(f);return()=>{this.controls.delete(f);};};
 getSnapshot=()=>this.state;
 sendControl(type:string,payload:Record<string,unknown>){const requestId=crypto.randomUUID();this.sent.push({type,payload,requestId});return requestId;}
 sendAwareness(){}
 emit(type:string,payload:Record<string,unknown>,requestId='server'){for(const f of this.controls)f({type,payload,requestId});}
 status(status:TransportState['status'],connectionId?:string){this.state={...this.state,status,connectionId:arguments.length===1?this.state.connectionId:connectionId};for(const f of this.listeners)f();}
}
export const participant=(id:string,following:string|null=null,voiceJoined=false)=>({connectionId:id,principalId:'same-user',principalType:'user',displayName:id,access:'edit' as const,following,voiceJoined,voiceMuted:false});
