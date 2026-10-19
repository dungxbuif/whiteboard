import {RenderError,RENDER_LIMITS,type ResolvedAsset} from './render-worker.js';

/** Inspect encoded dimensions before Chromium decodes potentially huge images. */
export function validateRasterAsset(asset:ResolvedAsset):void{
 if(asset.data.length>Math.ceil(RENDER_LIMITS.artifactBytes/3)*4)throw new RenderError('RENDER_LIMIT','Asset bytes exceed limit.');
 const b=Buffer.from(asset.data,'base64');
 if(b.toString('base64')!==asset.data)throw new RenderError('INVALID_ASSET','Invalid asset encoding.');
 let width=0,height=0;
 if(asset.mimeType==='image/png'&&b.length>=24&&b.subarray(0,8).toString('hex')==='89504e470d0a1a0a'){width=b.readUInt32BE(16);height=b.readUInt32BE(20);}
 else if(asset.mimeType==='image/gif'&&b.length>=10&&/^GIF8[79]a$/.test(b.subarray(0,6).toString())){width=b.readUInt16LE(6);height=b.readUInt16LE(8);}
 else if(asset.mimeType==='image/jpeg'&&b.length>=4&&b.readUInt16BE(0)===0xffd8){
  let offset=2;
  while(offset+4<=b.length){if(b[offset++]!==255)break;while(b[offset]===255)offset++;const marker=b[offset++];if(marker===0xd9||marker===0xda)break;const length=b.readUInt16BE(offset);if(length<2||offset+length>b.length)break;if([0xc0,0xc1,0xc2,0xc3,0xc5,0xc6,0xc7,0xc9,0xca,0xcb,0xcd,0xce,0xcf].includes(marker)&&length>=7){height=b.readUInt16BE(offset+3);width=b.readUInt16BE(offset+5);break;}offset+=length;}
 }else if(asset.mimeType==='image/webp'&&b.length>=30&&b.subarray(0,4).toString()==='RIFF'&&b.subarray(8,12).toString()==='WEBP'){
  const chunk=b.subarray(12,16).toString();
  if(chunk==='VP8X'){width=1+b.readUIntLE(24,3);height=1+b.readUIntLE(27,3);}
  else if(chunk==='VP8 '&&b.subarray(23,26).toString('hex')==='9d012a'){width=b.readUInt16LE(26)&0x3fff;height=b.readUInt16LE(28)&0x3fff;}
  else if(chunk==='VP8L'&&b[20]===0x2f){const bits=b.readUInt32LE(21);width=1+(bits&0x3fff);height=1+((bits>>>14)&0x3fff);}
 }
 if(width<1||height<1)throw new RenderError('INVALID_ASSET','Asset MIME or encoded dimensions are invalid.');
 if(width>RENDER_LIMITS.dimension||height>RENDER_LIMITS.dimension)throw new RenderError('RENDER_LIMIT','Asset exceeds 4096 pixels.');
}
