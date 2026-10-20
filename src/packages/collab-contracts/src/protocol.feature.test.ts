import {describe,expect,it} from 'vitest';
import {parseFeatureChanged,parsePresentationState,parsePresentationWaiting,parseScreenAdmission} from './protocol.js';

describe('parseFeatureChanged',()=>{
 it('accepts cursor-only invalidations',()=>expect(parseFeatureChanged({eventCursor:'42'})).toEqual({eventCursor:'42'}));
 it.each([{eventCursor:'0'},{eventCursor:'2',subject:'secret'},{eventCursor:2},{eventCursor:'-1'}])('rejects content or invalid cursors: %j',value=>expect(()=>parseFeatureChanged(value)).toThrow());
});

it('parses bounded server-owned screen admission metadata',()=>{
 const sessionId='00000000-0000-4000-8000-000000000001';
 expect(parseScreenAdmission({admitted:true,connectionId:'connection',kind:'screen',peers:['publisher'],publisher:'publisher',receiving:true,sessionId})).toEqual({admitted:true,connectionId:'connection',kind:'screen',peers:['publisher'],publisher:'publisher',receiving:true,sessionId});
 expect(()=>parseScreenAdmission({admitted:true,connectionId:'connection',kind:'screen',peers:Array(10).fill('peer'),publisher:'publisher',receiving:true,sessionId})).toThrow(/screen admission/i);
 expect(()=>parseScreenAdmission({admitted:true,connectionId:'connection',kind:'screen',peers:[],publisher:'publisher',receiving:true,sessionId,content:'secret'})).toThrow(/screen admission/i);
});

describe('presentation protocol',()=>{
 const sessionId='00000000-0000-4000-8000-000000000001';
 it('parses only the waiting participant own status',()=>{
  expect(parsePresentationWaiting({sessionId,status:'pending',connectionId:'guest'})).toEqual({sessionId,status:'pending',connectionId:'guest'});
  expect(()=>parsePresentationWaiting({sessionId,status:'pending',connectionId:'guest',scene:{elements:[]}})).toThrow(/presentation waiting/i);
 });
 it('parses bounded host state and rejects duplicate participants',()=>{
  const participant={connectionId:'guest',principalId:'presentation:guest',displayName:'Guest'};
  expect(parsePresentationState({sessionId,hostConnectionId:'host',hostPrincipalId:'00000000-0000-4000-8000-000000000002',revision:1,pending:[participant],admitted:[]})).toMatchObject({sessionId,revision:1});
  expect(()=>parsePresentationState({sessionId,hostConnectionId:'host',hostPrincipalId:'user',revision:1,pending:[participant,participant],admitted:[]})).toThrow(/presentation state/i);
 });
});
