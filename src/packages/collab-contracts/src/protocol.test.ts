import {describe,it,expect} from 'vitest';
import {encodeFrame,decodeFrame,MAX_FRAME_BYTES} from './protocol.js';
describe('binary frame contract',()=>{
  it('round trips payload and request ID without room scope',()=>{
    const frame={channel:3 as const,updateId:'request-✓',payload:new Uint8Array([1,2,3])};
    expect(decodeFrame(encodeFrame(frame))).toEqual(frame);
  });
  it('accepts exactly 1 MiB and rejects one additional byte',()=>{
    const frame={channel:3 as const,updateId:'x',payload:new Uint8Array(MAX_FRAME_BYTES-9)};
    expect(encodeFrame(frame).byteLength).toBe(MAX_FRAME_BYTES);
    expect(decodeFrame(encodeFrame(frame)).payload.byteLength).toBe(MAX_FRAME_BYTES-9);
    expect(()=>encodeFrame({...frame,payload:new Uint8Array(MAX_FRAME_BYTES-8)})).toThrow(/limit/i);
    expect(()=>decodeFrame(new Uint8Array(MAX_FRAME_BYTES+1))).toThrow(/limit/i);
  });
  it.each(['version','channel','id length','payload length','truncated','trailing','utf8'])('rejects malformed %s',kind=>{
    const good=encodeFrame({channel:3,updateId:'x',payload:new Uint8Array([1])});
    let bad=good.slice();
    if(kind==='version')bad[0]=2;
    if(kind==='channel')bad[1]=99;
    if(kind==='id length')bad[3]=127;
    if(kind==='payload length')bad[7]=9;
    if(kind==='truncated')bad=bad.slice(0,-1);
    if(kind==='trailing')bad=new Uint8Array([...bad,0]);
    if(kind==='utf8')bad[8]=255;
    expect(()=>decodeFrame(bad)).toThrow();
  });
  it('rejects empty, oversized, noncanonical IDs and unknown frame keys',()=>{
    for(const updateId of ['', 'x'.repeat(129), '\ud800'])expect(()=>encodeFrame({channel:3,updateId,payload:new Uint8Array()})).toThrow();
    expect(()=>encodeFrame({channel:3,updateId:'x',payload:new Uint8Array(),room:'other'} as never)).toThrow();
  });
});
