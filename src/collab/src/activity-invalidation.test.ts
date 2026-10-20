import {afterEach,expect,it} from 'vitest';
import {roomFixture} from './test-fixtures/presence.js';

let fixture:Awaited<ReturnType<typeof roomFixture>>;
afterEach(async()=>{await fixture?.close();});

it('reauthorizes recipients and broadcasts only a cursor',async()=>{
 fixture=await roomFixture();const allowed=await fixture.connect('allowed'),revoked=await fixture.connect('revoked');fixture.denied.add('revoked');
 fixture.rooms.featureChanged('workspace','scene','42');
 expect((await allowed.wait('feature_changed','server')).payload).toEqual({eventCursor:'42'});
 await new Promise(resolve=>setTimeout(resolve,30));
 expect(revoked.messages.some(message=>message.type==='feature_changed')).toBe(false);
});
