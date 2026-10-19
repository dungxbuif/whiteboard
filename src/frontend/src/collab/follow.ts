export type Viewport = {scrollX:number;scrollY:number;zoom:number};
export function validViewport(value:Record<string,unknown>):value is Record<string,unknown>&Viewport {
 return [value.scrollX,value.scrollY].every(v=>typeof v==='number'&&Number.isFinite(v)&&Math.abs(v)<=10_000_000)&&typeof value.zoom==='number'&&Number.isFinite(value.zoom)&&value.zoom>=0.1&&value.zoom<=30;
}
export function canFollow(self:string|undefined,target:string|null,participants:readonly {connectionId:string;following:string|null}[]) {
 if(!self)return false;if(target===null)return true;
 const map=new Map(participants.map(p=>[p.connectionId,p]));if(target===self||!map.has(target))return false;
 const seen=new Set([self]);let next:string|null=target;
 while(next){if(seen.has(next))return false;seen.add(next);next=map.get(next)?.following??null;}
 return true;
}
