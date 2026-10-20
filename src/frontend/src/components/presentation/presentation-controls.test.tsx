import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { LivePresentation } from '@/collab/live-presentation';
import type { Control, TransportState } from '@/collab/transport';
import { AudienceView } from './audience-view';
import { HostControls } from './host-controls';
import { WaitingRoom } from './waiting-room';

class Transport {
 state:TransportState={status:'synced',canEdit:true,pending:0,recovery:false,connectionId:'host'};listeners=new Set<()=>void>();controls=new Set<(value:Control)=>void>();sent:{type:string;payload:Record<string,unknown>}[]=[];
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>this.listeners.delete(listener)};subscribeControl=(listener:(value:Control)=>void)=>{this.controls.add(listener);return()=>this.controls.delete(listener)};getSnapshot=()=>this.state;
 sendControl=(type:string,payload:Record<string,unknown>)=>{this.sent.push({type,payload});return crypto.randomUUID()};emit(type:string,payload:Record<string,unknown>){for(const listener of this.controls)listener({type,requestId:'server',payload})}
}
const session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const state=(overrides:Record<string,unknown>={})=>({sessionId:session,hostConnectionId:'host',hostPrincipalId:'user',revision:2,pending:[],admitted:[],frameId:null,spotlight:null,locked:false,reactions:[],...overrides});

describe('presentation overlays',()=>{
 it('keeps Escape and Leave active while the viewport is locked',()=>{
  const transport=new Transport(),live=new LivePresentation(transport);live.start();transport.state.connectionId='audience';transport.emit('presentation_state',state({locked:true,hostConnectionId:'host',frameId:'missing',admitted:[{connectionId:'audience',principalId:'presentation:a',displayName:'Audience'}]}));
  render(<AudienceView elements={[]} live={live}/>);expect(screen.getByText('The presented frame was removed.')).toBeInTheDocument();
  fireEvent.keyDown(window,{key:'Escape'});expect(transport.sent.at(-1)).toMatchObject({type:'presentation_leave'});
 });
 it('restores a locked native frame after audience camera input without blocking controls',()=>{
  const transport=new Transport(),live=new LivePresentation(transport),focus=vi.fn();live.start();transport.state.connectionId='audience';transport.emit('presentation_state',state({locked:true,frameId:'frame',admitted:[{connectionId:'audience',principalId:'presentation:a',displayName:'Audience'}]}));
  render(<AudienceView elements={[{id:'frame',type:'frame'}] as never} live={live} onFocusFrame={focus}/>);expect(focus).toHaveBeenCalledTimes(1);fireEvent.pointerUp(window);expect(focus).toHaveBeenCalledTimes(2);expect(screen.getByRole('button',{name:'Leave'})).toBeEnabled();
 });
 it('renders host admission controls as a floating region',()=>{
  const transport=new Transport(),live=new LivePresentation(transport,{create:async()=>({sessionId:session,token:'host-token',expiresAt:new Date().toISOString()}),createLink:async()=>({id:'link',token:'invite',expiresAt:new Date().toISOString()}),admit:async()=>({sessionId:session,token:'audience',expiresAt:new Date().toISOString()})});live.start();
  transport.emit('presentation_state',state({pending:[{connectionId:'guest',principalId:'presentation:g',displayName:'Guest'}]}));render(<HostControls live={live}/>);
  expect(screen.getByRole('region',{name:'Live presentation host controls'})).toHaveClass('absolute');fireEvent.click(screen.getByRole('button',{name:'Admit Guest'}));
  expect(screen.getByRole('button',{name:'End presentation'})).toBeInTheDocument();
 });
 it('shows waiting state without canvas or participant data',()=>{render(<WaitingRoom onLeave={()=>{}}/>);expect(screen.getByText('Waiting for the host')).toBeInTheDocument();expect(screen.queryByLabelText('Excalidraw canvas')).not.toBeInTheDocument();});
 it('offers a fresh waiting-room join after audience authority expires',()=>{let retried=false;render(<WaitingRoom onLeave={()=>{}} onRejoin={()=>{retried=true}} status="ended"/>);fireEvent.click(screen.getByRole('button',{name:'Rejoin waiting room'}));expect(retried).toBe(true);});
});
