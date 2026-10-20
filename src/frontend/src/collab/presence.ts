import type {Control,TransportState} from './transport';
import {canFollow,validViewport,type Viewport} from './follow';
export type Participant={connectionId:string;principalId:string;principalType:string;displayName:string;access:'view'|'edit';following:string|null;voiceJoined:boolean;voiceMuted:boolean;screenJoined?:boolean;screenPublisher?:boolean;screenSessionId?:string};
export type Cursor={cursor?:{x:number;y:number}|null;selectedIds?:string[]};
export type PresenceTransport={getSnapshot:()=>TransportState;subscribe:(f:()=>void)=>()=>void;subscribeControl:(f:(c:Control)=>void)=>()=>void;sendControl:(type:string,payload:Record<string,unknown>)=>string|undefined;sendAwareness:(value:{cursor?:{x:number;y:number};selectedIds?:string[]})=>void};
export type PresenceState={connectionId?:string;participants:Participant[];cursors:Record<string,Cursor>;following:string|null;error?:string};
const initial=():PresenceState=>({connectionId:undefined,participants:[],cursors:{},following:null,error:undefined});
export class Presence {
 private state:PresenceState=initial();private listeners=new Set<()=>void>();private viewports=new Set<(v:Viewport)=>void>();private cleanup:(()=>void)[]=[];private announced?:string;private lastViewport=0;private lastSelection='';private lastCamera?:Viewport;
 constructor(readonly transport:PresenceTransport,private name:()=>string){}
 getSnapshot=()=>this.state;
 subscribe=(f:()=>void)=>{this.listeners.add(f);return()=>{this.listeners.delete(f);};};
 subscribeViewport=(f:(v:Viewport)=>void)=>{this.viewports.add(f);return()=>{this.viewports.delete(f);};};
 private set(update:Partial<PresenceState>){this.state={...this.state,...update};for(const f of this.listeners)f();}
 start(){if(this.cleanup.length)return;this.cleanup=[this.transport.subscribe(this.changed),this.transport.subscribeControl(this.control)];this.changed();}
 stop(){for(const clean of this.cleanup)clean();this.cleanup=[];this.announced=undefined;this.lastCamera=undefined;this.set(initial());}
 private changed=()=>{
  const s=this.transport.getSnapshot();const id=['offline','revoked','degraded','legacy'].includes(s.status)?undefined:s.connectionId;
  if(id!==this.state.connectionId){this.announced=undefined;this.lastSelection='';this.lastCamera=undefined;this.set({...initial(),connectionId:id});}
  if(id&&id!==this.announced&&['synced','pending','readonly'].includes(s.status))if(this.transport.sendControl('presence',{displayName:(this.name().trim()||'Guest').slice(0,80)}))this.announced=id;
 };
 private control=(control:Control)=>{
  if(!this.state.connectionId)return;const p=control.payload;
  if(control.type==='roster'&&Array.isArray(p.participants)){
   const participants=p.participants.filter((v):v is Participant=>!!v&&typeof v.connectionId==='string'&&typeof v.displayName==='string'&&['edit','view'].includes(v.access));
   const ids=new Set(participants.map(v=>v.connectionId));const following=participants.find(v=>v.connectionId===this.state.connectionId)?.following??null;
   const priorFollowers=this.state.participants.filter(v=>v.following===this.state.connectionId).length;
   this.set({participants,cursors:Object.fromEntries(Object.entries(this.state.cursors).filter(([id])=>ids.has(id))),following:following&&ids.has(following)?following:null});
   if(this.lastCamera&&participants.filter(v=>v.following===this.state.connectionId).length>priorFollowers)this.viewport(this.lastCamera,true);
  }else if(control.type==='awareness'&&typeof p.connectionId==='string'&&this.state.participants.some(v=>v.connectionId===p.connectionId)){
   this.set({cursors:{...this.state.cursors,[p.connectionId]:{...this.state.cursors[p.connectionId],...(p.cursor!==undefined?{cursor:p.cursor as Cursor['cursor']}:{}),...(Array.isArray(p.selectedIds)?{selectedIds:p.selectedIds as string[]}: {})}}});
  }else if(control.type==='follow'&&p.connectionId===this.state.connectionId){this.set({following:typeof p.targetConnectionId==='string'?p.targetConnectionId:null,error:undefined});}
  else if(control.type==='viewport'&&p.connectionId===this.state.following&&validViewport(p))for(const f of this.viewports)f({scrollX:p.scrollX,scrollY:p.scrollY,zoom:p.zoom});
  else if(control.type==='error'&&['INVALID_FOLLOW','INVALID_PRESENCE'].includes(String(p.code)))this.set({error:String(p.message??'Collaboration action unavailable')});
 };
 follow(target:string|null){if(!canFollow(this.state.connectionId,target,this.state.participants)){this.set({error:'Choose another available collaborator to follow.'});return false;}const sent=this.transport.sendControl('follow',{targetConnectionId:target});if(sent&&this.state.error)this.set({error:undefined});return !!sent;}
 viewport(value:Viewport,force=false){if(!validViewport(value))return;this.lastCamera=value;if(!this.state.participants.some(p=>p.following===this.state.connectionId))return;if(!force&&Date.now()-this.lastViewport<150)return;this.lastViewport=Date.now();this.transport.sendControl('viewport',value);}
 selection(ids:string[]){const clean=ids.slice(0,256),key=JSON.stringify(clean);if(key===this.lastSelection)return;this.lastSelection=key;this.transport.sendAwareness({selectedIds:clean});}
}
