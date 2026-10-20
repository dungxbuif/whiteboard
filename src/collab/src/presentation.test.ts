import {describe,it,expect} from 'vitest';
import {PresentationSessions,PRESENTATION_AUDIENCE_LIMIT,PRESENTATION_WAITING_LIMIT} from './presentation.js';

describe('presentation waiting room',()=>{
 it('keeps waiting clients metadata-only and ends on host disconnect',()=>{
  const sessions=new PresentationSessions();sessions.start('room','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','host','user');
  expect(sessions.wait('room','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',{connectionId:'guest',principalId:'presentation:x',displayName:'Guest'})).toEqual({sessionId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',status:'pending',connectionId:'guest'});
  expect(sessions.state('room')?.pending).toHaveLength(1);expect(sessions.disconnect('host')?.ended.sessionId).toBe('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa');expect(sessions.state('room')).toBeUndefined();
 });
 it('caps waiting and requires matching host revision for admission',()=>{
  const sessions=new PresentationSessions(),session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';sessions.start('room',session,'host','user');
  for(let i=0;i<PRESENTATION_WAITING_LIMIT;i++)sessions.wait('room',session,{connectionId:'g'+i,principalId:'p'+i,displayName:'Guest'});
  expect(()=>sessions.wait('room',session,{connectionId:'overflow',principalId:'p',displayName:'Guest'})).toThrow(/full/i);
  expect(()=>sessions.admit('room',session,'other','g0',1)).toThrow(/host/i);expect(()=>sessions.admit('room',session,'host','g0',2)).toThrow(/revision/i);
  const admitted=sessions.admit('room',session,'host','g0',1);expect(admitted.state.revision).toBe(2);expect(admitted.participant.connectionId).toBe('g0');
 });
 it('bounds admitted audience state to the protocol maximum',()=>{
  const sessions=new PresentationSessions(),session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';sessions.start('room',session,'host','user');
  for(let i=0;i<PRESENTATION_AUDIENCE_LIMIT;i++){sessions.wait('room',session,{connectionId:'a'+i,principalId:'p'+i,displayName:'Guest'});sessions.admit('room',session,'host','a'+i,i+1);}
  sessions.wait('room',session,{connectionId:'overflow',principalId:'overflow',displayName:'Guest'});
  expect(()=>sessions.admit('room',session,'host','overflow',PRESENTATION_AUDIENCE_LIMIT+1)).toThrow(/full/i);
 });

 it('rejects stale host revisions and keeps one authoritative frame and lock state',()=>{
  const sessions=new PresentationSessions(),session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
  sessions.start('room',session,'host','user');
  expect(()=>sessions.setFrame('room',session,'host','frame-a',2)).toThrow(/revision/i);
  expect(sessions.setFrame('room',session,'host','frame-a',1)).toMatchObject({frameId:'frame-a',revision:2});
  expect(sessions.setLocked('room',session,'host',true,2)).toMatchObject({locked:true,revision:3});
  expect(()=>sessions.setFrame('room',session,'host','frame-b',2)).toThrow(/revision/i);
 });

 it('transfers only the explicit current host revision and clears spotlight when target leaves',()=>{
  const sessions=new PresentationSessions(),session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
  sessions.start('room',session,'host','user');
  sessions.wait('room',session,{connectionId:'guest',principalId:'presentation:g',displayName:'Guest'});
  sessions.admit('room',session,'host','guest',1);
  expect(sessions.spotlight('room',session,'host','guest',2)).toMatchObject({spotlight:'guest',revision:3});
  expect(sessions.disconnect('guest')?.state).toMatchObject({spotlight:null,revision:4});
  expect(()=>sessions.transfer('room',session,'other','editor','editor-user',4)).toThrow(/host/i);
  expect(sessions.transfer('room',session,'host','editor','editor-user',4)).toMatchObject({hostConnectionId:'editor',hostPrincipalId:'editor-user',revision:5});
 });

 it('allowlists and rate limits reactions and gives them a three second expiry',()=>{
  let now=1_000;const sessions=new PresentationSessions(()=>now),session='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
  sessions.start('room',session,'host','user');
  expect(()=>sessions.react('room',session,'host','💣')).toThrow(/reaction/i);
  expect(sessions.react('room',session,'host','👏')).toMatchObject({connectionId:'host',emoji:'👏',expiresAt:4_000});
  expect(()=>sessions.react('room',session,'host','👍')).toThrow(/rate/i);
  now=2_000;expect(sessions.react('room',session,'host','👍')).toMatchObject({expiresAt:5_000});
  now=4_001;expect(sessions.state('room')?.reactions).toHaveLength(1);
 });
});
