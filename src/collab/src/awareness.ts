import type {SceneRef} from '@collab/contracts';
import {CollabError} from './auth.js';

const MAX_NAME_LENGTH = 80;
const MAX_COORDINATE = 10_000_000;
const MAX_SELECTION = 256;
const IDENTITY_FIELDS = ['connectionId', 'principalId', 'principalType', 'from', 'access', 'role'];
export type Participant = {
 connectionId:string; principalId:string; principalType:string; displayName:string;
 access:string; following:string|null;
};
export const roomKey = (ref:SceneRef) => ref.workspaceId+':'+ref.sceneId;
export function fields(value:unknown, allowed:string[], code='INVALID_PRESENCE'):Record<string,unknown> {
 if(!value || typeof value!=='object' || Array.isArray(value) || Object.keys(value).some(key=>!allowed.includes(key)&&!IDENTITY_FIELDS.includes(key)))throw new CollabError(code,'Invalid ephemeral control',400);
 return value as Record<string,unknown>;
}
function coordinate(value:unknown):value is number {return typeof value==='number'&&Number.isFinite(value)&&Math.abs(value)<=MAX_COORDINATE;}
export function parseAwareness(value:unknown) {
 const input=fields(value,['cursor','selectedIds']);const output:{cursor?:{x:number;y:number}|null;selectedIds?:string[]}={};
 if(input.cursor!==undefined){if(input.cursor===null)output.cursor=null;else{const cursor=fields(input.cursor,['x','y']);if(!coordinate(cursor.x)||!coordinate(cursor.y))throw new CollabError('INVALID_PRESENCE','Invalid cursor',400);output.cursor={x:cursor.x,y:cursor.y};}}
 if(input.selectedIds!==undefined){if(!Array.isArray(input.selectedIds)||input.selectedIds.length>MAX_SELECTION||input.selectedIds.some(id=>typeof id!=='string'||!id.length||id.length>256))throw new CollabError('INVALID_PRESENCE','Invalid selection',400);output.selectedIds=input.selectedIds;}
 return output;
}
export function parseViewport(value:unknown) {
 const input=fields(value,['scrollX','scrollY','zoom']);
 if(!coordinate(input.scrollX)||!coordinate(input.scrollY)||typeof input.zoom!=='number'||!Number.isFinite(input.zoom)||input.zoom<0.1||input.zoom>30)throw new CollabError('INVALID_PRESENCE','Invalid viewport',400);
 return {scrollX:input.scrollX,scrollY:input.scrollY,zoom:input.zoom};
}
export class Awareness {
 private participants=new Map<string,{room:string;participant:Participant}>();
 register(ref:SceneRef,identity:Omit<Participant,'displayName'|'following'>) {
  const previous=this.participants.get(identity.connectionId)?.participant;
  this.participants.set(identity.connectionId,{room:roomKey(ref),participant:{...identity,displayName:previous?.displayName??(identity.principalType==='public'?'Guest':'Participant'),following:previous?.following??null}});
 }
 list(ref:SceneRef):Participant[] {return [...this.participants.values()].filter(p=>p.room===roomKey(ref)).map(p=>({...p.participant}));}
 update(connectionId:string,value:unknown) {
  const input=fields(value,['displayName']);
  if(typeof input.displayName!=='string'||input.displayName.length>MAX_NAME_LENGTH)throw new CollabError('INVALID_PRESENCE','Invalid display name',400);
  const displayName=input.displayName.replace(/[\u0000-\u001f\u007f-\u009f\u202a-\u202e\u2066-\u2069]/g,'').trim();
  if(!displayName)throw new CollabError('INVALID_PRESENCE','Display name required',400);
  const entry=this.participants.get(connectionId);if(entry)entry.participant.displayName=displayName;
 }
 follow(connectionId:string,target:unknown) {
  const entry=this.participants.get(connectionId),destination=typeof target==='string'?this.participants.get(target):undefined;
  if(!entry || (target!==null&&(!destination||destination.room!==entry.room||target===connectionId)))throw new CollabError('INVALID_FOLLOW','Target is not available',400);
  let next=destination?.participant;
  while(next){if(next.connectionId===connectionId)throw new CollabError('INVALID_FOLLOW','Follow cycle is not allowed',400);next=next.following?this.participants.get(next.following)?.participant:undefined;}
  entry.participant.following=target as string|null;
 }
 remove(connectionId:string):string[] {
  this.participants.delete(connectionId);const cleared:string[]=[];
  for(const {participant} of this.participants.values())if(participant.following===connectionId){participant.following=null;cleared.push(participant.connectionId);}
  return cleared;
 }
}
