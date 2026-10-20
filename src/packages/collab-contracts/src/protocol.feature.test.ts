import {describe,expect,it} from 'vitest';
import {parseFeatureChanged} from './protocol.js';

describe('parseFeatureChanged',()=>{
 it('accepts cursor-only invalidations',()=>expect(parseFeatureChanged({eventCursor:'42'})).toEqual({eventCursor:'42'}));
 it.each([{eventCursor:'0'},{eventCursor:'2',subject:'secret'},{eventCursor:2},{eventCursor:'-1'}])('rejects content or invalid cursors: %j',value=>expect(()=>parseFeatureChanged(value)).toThrow());
});
