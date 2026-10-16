import ELK from 'elkjs/lib/elk.bundled.js';
import type {ElkNode} from 'elkjs/lib/elk-api';
import {asGraph} from './catalog';
import {DiagramError,LIMITS,type DiagramInput,type GraphNode,type LayoutNode,type LayoutEdge,type LayoutResult,type SequenceData,type TimelineData,type JourneyData,type SwimlaneData} from './types';

const GAP=90,PADDING=40,NODE_WIDTH=240,LINE_HEIGHT=24,TEXT_COLUMNS=24;
export function wrapText(text:string,columns=TEXT_COLUMNS):string{return text.split('\n').map(line=>{const chars=Array.from(line),lines:string[]=[];while(chars.length>columns){let at=chars.slice(0,columns+1).lastIndexOf(' ');if(at<columns/2)at=columns;lines.push(chars.splice(0,at).join(''));if(chars[0]===' ')chars.shift();}lines.push(chars.join(''));return lines.join('\n');}).join('\n');}
export function measureNode(n:GraphNode):LayoutNode{
 const semantic=n.kind&&n.kind!=='process'&&n.kind!=='state'?`«${n.kind}»\n`:'';
 const text=wrapText(semantic+n.label+(n.technology?`\n[${n.technology}]`:'')+(n.description?`\n${n.description}`:''));
 const shape=n.kind==='decision'?'diamond':['person','start','end','external','network'].includes(n.kind??'')?'ellipse':'rectangle';
 const factor=shape==='rectangle'?1:1.6;
 return {...n,x:0,y:0,width:NODE_WIDTH*factor,height:Math.max(88,(text.split('\n').length*LINE_HEIGHT+32)*factor),text,shape,role:'node'};
}
function connectors(edges:import('./types').GraphEdge[],nodes:LayoutNode[],direction:'LR'|'TB'):LayoutEdge[]{const map=new Map(nodes.map(n=>[n.id,n]));return edges.map(e=>{const a=map.get(e.from)!,b=map.get(e.to)!;const p:[number,number]=direction==='LR'?[a.x+a.width,a.y+a.height/2]:[a.x+a.width/2,a.y+a.height];const q:[number,number]=direction==='LR'?[b.x,b.y+b.height/2]:[b.x+b.width/2,b.y];return {...e,points:[p,...(direction==='LR'?[[ (p[0]+q[0])/2,p[1] ],[(p[0]+q[0])/2,q[1]]]:[[p[0],(p[1]+q[1])/2],[q[0],(p[1]+q[1])/2]]) as [number,number][],q]};});}
async function layered(input:DiagramInput):Promise<LayoutResult>{
 const graph=asGraph(input),nodes=graph.nodes.map(measureNode);
 const nodeIds=new Map(nodes.map((n,i)=>[n.id,`n${i}`])),edgeIds=new Map(graph.edges.map((e,i)=>[e.id,`e${i}`]));
 const groupIds=new Map((graph.groups??[]).map((g,i)=>[g.id,`g${i}`]));
 const elkNodes=nodes.map(n=>({id:nodeIds.get(n.id)!,width:n.width,height:n.height}));
 const children:ElkNode[]=[...elkNodes.filter((_,i)=>!nodes[i].group),...(graph.groups??[]).map(g=>({id:groupIds.get(g.id)!,layoutOptions:{'elk.padding':'[top=56,left=24,bottom=24,right=24]'},children:elkNodes.filter((_,i)=>nodes[i].group===g.id)}))];
 const elk=new ELK();let timer:ReturnType<typeof setTimeout>|undefined;
 const layout=elk.layout({id:'root',layoutOptions:{'elk.algorithm':'layered','elk.hierarchyHandling':'INCLUDE_CHILDREN','elk.direction':input.direction==='LR'?'RIGHT':'DOWN','elk.edgeRouting':'ORTHOGONAL','elk.spacing.nodeNode':String(GAP),'elk.layered.spacing.nodeNodeBetweenLayers':String(GAP+30),'elk.randomSeed':'1','elk.padding':`[top=${PADDING},left=${PADDING},bottom=${PADDING},right=${PADDING}]`},children,edges:graph.edges.map(e=>({id:edgeIds.get(e.id)!,sources:[nodeIds.get(e.from)!],targets:[nodeIds.get(e.to)!],labels:e.label?[{text:e.label,width:Math.min(240,Array.from(e.label).length*9),height:24}]:[]}))});
 let result:ElkNode;try{result=await Promise.race([layout,new Promise<never>((_,reject)=>{timer=setTimeout(()=>reject(new DiagramError('LAYOUT_TIMEOUT','Layout exceeded 10 seconds; reduce the graph.')),LIMITS.layoutMs);})]);}finally{clearTimeout(timer);}
 const boundaries:LayoutNode[]=[],routes=new Map<string,{points:[number,number][]}>();
 const collect=(parent:ElkNode,ox=0,oy=0)=>{for(const edge of parent.edges??[]){const route=edge.sections?.[0];if(route)routes.set(edge.id,{points:[route.startPoint,...route.bendPoints??[],route.endPoint].map(p=>[p.x+ox,p.y+oy])});}for(const child of parent.children??[]){const x=ox+(child.x??0),y=oy+(child.y??0);const n=nodes.find(n=>nodeIds.get(n.id)===child.id);if(n){n.x=x;n.y=y;}else{const group=graph.groups?.find(g=>groupIds.get(g.id)===child.id);if(group)boundaries.push({id:group.id,label:group.label,text:group.label,x,y,width:child.width??0,height:child.height??0,role:'boundary',group:group.id,shape:'rectangle'});}collect(child,x,y);}};
 collect(result);
 const edges:LayoutEdge[]=graph.edges.map(e=>{const points=routes.get(edgeIds.get(e.id)!)?.points??connectors([e],nodes,input.direction)[0].points;
  return {...e,points,dashed:e.kind==='dependency'||e.kind==='async',startHead:e.kind==='composition'?'diamond':e.kind==='many-to-many'?'crowfoot_many':e.kind?.startsWith('one-to')?'crowfoot_one':null,endHead:e.kind==='inheritance'?'triangle_outline':e.kind==='many-to-many'||e.kind==='one-to-many'?'crowfoot_many':e.kind==='one-to-one'?'crowfoot_one':'arrow'};
 });
 return {nodes:[...boundaries,...nodes],edges};
}
function sequence(data:SequenceData):LayoutResult{
 const nodes=data.participants.map(p=>measureNode({...p,kind:p.kind==='actor'?'person':'participant'}));
 const longest=Math.max(240,...data.messages.map(m=>Math.min(600,Array.from(m.label).length*10+40)));let x=PADDING;
 for(const n of nodes){n.x=x;n.y=PADDING;x+=n.width+longest;}
 const top=Math.max(...nodes.map(n=>n.height+n.y))+GAP;let y=top;
 const byId=new Map(nodes.map(n=>[n.id,n]));const edges:LayoutEdge[]=[];
 for(const m of data.messages){const a=byId.get(m.from)!,b=byId.get(m.to)!;const ax=a.x+a.width/2,bx=b.x+b.width/2;edges.push({...m,points:ax===bx?[[ax,y],[ax+80,y],[ax+80,y+40],[ax,y+40]]:[[ax,y],[bx,y]],dashed:m.kind==='return',bind:false,endHead:m.kind==='async'?'arrow':'triangle'});y+=Math.max(GAP,wrapText(m.label).split('\n').length*LINE_HEIGHT+40);}
 for(const n of nodes)edges.unshift({id:n.id,from:n.id,to:n.id,points:[[n.x+n.width/2,n.y+n.height],[n.x+n.width/2,y]],dashed:true,endHead:null,bind:false,role:'lifeline'});
 return {nodes,edges};
}
function timeline(data:TimelineData,input:DiagramInput):LayoutResult{
 const nodes=data.events.map(e=>measureNode({id:e.id,label:e.label,description:[e.date,e.description].filter(Boolean).join('\n'),kind:'milestone'}));let offset=PADDING;
 for(const n of nodes){n.x=input.direction==='LR'?offset:PADDING;n.y=input.direction==='TB'?offset:PADDING;offset+=(input.direction==='LR'?n.width:n.height)+GAP;}
 return {nodes,edges:connectors(nodes.slice(1).map((n,i)=>({id:`next:${n.id}`,from:nodes[i].id,to:n.id})),nodes,input.direction)};
}
function journey(data:JourneyData):LayoutResult{
 const nodes:LayoutNode[]=[],colWidth=NODE_WIDTH+GAP;
 const actorLabels=new Map(data.actors.map(a=>[a.id,a.label]));
 const cards=new Map(data.stages.flatMap(s=>s.steps.map(step=>[step.id,measureNode({id:step.id,label:step.label,description:`${actorLabels.get(step.actor)}${step.score?`\nExperience: ${step.score}/5`:''}`})] as const)));
 const rowHeights=data.actors.map(a=>Math.max(180,...data.stages.flatMap(s=>s.steps.filter(t=>t.actor===a.id).map(t=>cards.get(t.id)!.height+50))));
 const rows:number[]=[];let y=150;for(const h of rowHeights){rows.push(y);y+=h+GAP;}
 let x=PADDING;
 for(const s of data.stages){const maxPerActor=Math.max(1,...data.actors.map(a=>s.steps.filter(t=>t.actor===a.id).length));nodes.push({...measureNode({id:`stage:${s.id}`,label:s.label}),x,y:PADDING,role:'header',height:80});for(const a of data.actors){const actorSteps=s.steps.filter(t=>t.actor===a.id);actorSteps.forEach((step,k)=>nodes.push({...cards.get(step.id)!,x:x+k*colWidth,y:rows[data.actors.indexOf(a)]}));}x+=colWidth*maxPerActor;}
 return {nodes,edges:[]};
}
function swimlane(data:SwimlaneData,input:DiagramInput):LayoutResult{
 const nodes=data.steps.map(s=>({...measureNode({...s}),group:s.lane}));const maxHeight=Math.max(...nodes.map(n=>n.height));const row=maxHeight+GAP,stepWidth=NODE_WIDTH*1.6+GAP;
 for(const [i,n] of nodes.entries()){const lane=data.lanes.findIndex(l=>l.id===data.steps[i].lane);n.x=PADDING+260+i*stepWidth;n.y=PADDING+70+lane*row;}
 const headers:LayoutNode[]=data.lanes.map((l,i)=>({...measureNode(l),x:PADDING,y:PADDING+70+i*row,width:200,height:80,role:'header'}));
 const boundaries:LayoutNode[]=data.lanes.map((l,i)=>({id:`lane:${l.id}`,label:l.label,text:'',x:PADDING-16,y:PADDING+40+i*row,width:300+nodes.length*stepWidth,height:row,role:'boundary',group:l.id}));
 const edges=connectors(data.edges,nodes,'LR');
 if(input.direction==='TB'){for(const n of [...nodes,...headers,...boundaries]){[n.x,n.y]=[n.y,n.x];[n.width,n.height]=[n.height,n.width];}for(const e of edges)e.points=e.points.map(([x,y])=>[y,x]);}
 return {nodes:[...boundaries,...headers,...nodes],edges};
}
export async function layoutDiagram(input:DiagramInput):Promise<LayoutResult>{
 switch(input.preset){case 'sequence':return sequence(input.data as SequenceData);case 'timeline':return timeline(input.data as TimelineData,input);case 'user-journey':return journey(input.data as JourneyData);case 'swimlane':return swimlane(input.data as SwimlaneData,input);default:return layered(input);}
}
