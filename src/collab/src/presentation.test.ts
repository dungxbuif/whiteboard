import {describe,it,expect} from 'vitest';
import {PresentationSessions,PRESENTATION_WAITING_LIMIT} from './presentation.js';

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
});
