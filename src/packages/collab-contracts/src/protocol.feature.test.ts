import {describe,expect,it} from 'vitest';
import {parseFeatureChanged,parseScreenAdmission} from './protocol.js';

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
