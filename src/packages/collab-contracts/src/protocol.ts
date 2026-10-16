export const MAX_FRAME_BYTES = 1024 * 1024;
export const PROTOCOL_VERSION = 1;
export const FRAME_HEADER_BYTES = 8;
export type Frame = { channel: 1 | 2 | 3; updateId: string; payload: Uint8Array };
const encoder = new TextEncoder();
const decoder = new TextDecoder('utf-8', {fatal: true});
export function encodeFrame(frame: Frame): Uint8Array {
  if (!frame || Object.keys(frame).some(key => !['channel','updateId','payload'].includes(key)) || ![1,2,3].includes(frame.channel) || typeof frame.updateId !== 'string' || !(frame.payload instanceof Uint8Array)) throw new Error('Invalid frame');
  const id = encoder.encode(frame.updateId);
  if (id.length < 1 || id.length > 128 || decoder.decode(id) !== frame.updateId) throw new Error('Invalid update ID');
  const length = FRAME_HEADER_BYTES + id.length + frame.payload.length;
  if (length > MAX_FRAME_BYTES) throw new Error('Frame byte limit exceeded');
  const result = new Uint8Array(length);
  const view = new DataView(result.buffer);
  result[0] = PROTOCOL_VERSION; result[1] = frame.channel;
  view.setUint16(2,id.length);view.setUint32(4,frame.payload.length);
  result.set(id,FRAME_HEADER_BYTES);result.set(frame.payload,FRAME_HEADER_BYTES+id.length);
  return result;
}
export function decodeFrame(bytes: Uint8Array): Frame {
  if (!(bytes instanceof Uint8Array)) throw new Error('Invalid frame bytes');
  if (bytes.length > MAX_FRAME_BYTES) throw new Error('Frame byte limit exceeded');
  if (bytes.length < FRAME_HEADER_BYTES || bytes[0] !== PROTOCOL_VERSION || ![1,2,3].includes(bytes[1])) throw new Error('Invalid frame version or channel');
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  const idLength = view.getUint16(2), payloadLength = view.getUint32(4);
  if (idLength < 1 || idLength > 128 || FRAME_HEADER_BYTES + idLength + payloadLength !== bytes.length) throw new Error('Invalid frame length or trailing bytes');
  const updateId = decoder.decode(bytes.subarray(FRAME_HEADER_BYTES, FRAME_HEADER_BYTES+idLength));
  return {channel:bytes[1] as Frame['channel'],updateId,payload:bytes.slice(FRAME_HEADER_BYTES+idLength)};
}
