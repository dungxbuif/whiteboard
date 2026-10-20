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
  const idBytes=bytes.subarray(FRAME_HEADER_BYTES,FRAME_HEADER_BYTES+idLength);
  const updateId=decoder.decode(idBytes);
  const canonical=encoder.encode(updateId);
  if(canonical.length!==idLength||canonical.some((byte,index)=>byte!==idBytes[index]))throw new Error('Invalid noncanonical update ID');
  return {channel:bytes[1] as Frame['channel'],updateId,payload:bytes.slice(FRAME_HEADER_BYTES+idLength)};
}

export type FeatureChanged={eventCursor:string};
export function parseFeatureChanged(value:unknown):FeatureChanged {
  if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>key!=='eventCursor'))throw new Error('Invalid feature invalidation');
  const cursor=(value as {eventCursor?:unknown}).eventCursor;
  if(typeof cursor!=='string'||!/^[1-9]\d*$/.test(cursor))throw new Error('Invalid feature cursor');
  return {eventCursor:cursor};
}

export type ScreenAdmission={admitted:boolean;connectionId:string;kind:'screen';peers:string[];publisher?:string;receiving?:boolean;sessionId?:string;reason?:'MEDIA_FULL'|'SCREEN_BUSY'};
const screenId=(value:unknown):value is string=>typeof value==='string'&&value.length>0&&value.length<=128;
const screenSessionId=(value:unknown):value is string=>typeof value==='string'&&/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
export function parseScreenAdmission(value:unknown):ScreenAdmission {
 if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!['admitted','connectionId','kind','peers','publisher','receiving','sessionId','reason'].includes(key)))throw new Error('Invalid screen admission');
 const input=value as Record<string,unknown>;
 if(typeof input.admitted!=='boolean'||!screenId(input.connectionId)||input.kind!=='screen'||!Array.isArray(input.peers)||input.peers.length>9||input.peers.some(peer=>!screenId(peer))||new Set(input.peers).size!==input.peers.length)throw new Error('Invalid screen admission');
 if(input.admitted){if(!screenId(input.publisher)||typeof input.receiving!=='boolean'||!screenSessionId(input.sessionId)||input.reason!==undefined)throw new Error('Invalid screen admission');}
 else if(!['MEDIA_FULL','SCREEN_BUSY'].includes(String(input.reason))||input.publisher!==undefined||input.receiving!==undefined||input.sessionId!==undefined)throw new Error('Invalid screen admission');
 return input as ScreenAdmission;
}

export type PresentationWaiting={sessionId:string;status:'pending';connectionId:string};
export type PresentationParticipant={connectionId:string;principalId:string;displayName:string};
export type PresentationState={sessionId:string;hostConnectionId:string;hostPrincipalId:string;revision:number;pending:PresentationParticipant[];admitted:PresentationParticipant[]};
const presentationConnection=(value:unknown):value is string=>typeof value==='string'&&value.length>0&&value.length<=128;
const presentationParticipant=(value:unknown):value is PresentationParticipant=>{
 if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!['connectionId','principalId','displayName'].includes(key)))return false;
 const input=value as Record<string,unknown>;
 return presentationConnection(input.connectionId)&&typeof input.principalId==='string'&&input.principalId.length>0&&input.principalId.length<=256&&typeof input.displayName==='string'&&input.displayName.length>0&&input.displayName.length<=200;
};
export function parsePresentationWaiting(value:unknown):PresentationWaiting {
 if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!['sessionId','status','connectionId'].includes(key)))throw new Error('Invalid presentation waiting status');
 const input=value as Record<string,unknown>;
 if(!screenSessionId(input.sessionId)||input.status!=='pending'||!presentationConnection(input.connectionId))throw new Error('Invalid presentation waiting status');
 return input as PresentationWaiting;
}
export function parsePresentationState(value:unknown):PresentationState {
 if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!['sessionId','hostConnectionId','hostPrincipalId','revision','pending','admitted'].includes(key)))throw new Error('Invalid presentation state');
 const input=value as Record<string,unknown>,pending=input.pending,admitted=input.admitted;
 if(!screenSessionId(input.sessionId)||!presentationConnection(input.hostConnectionId)||typeof input.hostPrincipalId!=='string'||!input.hostPrincipalId||!Number.isSafeInteger(input.revision)||Number(input.revision)<1||!Array.isArray(pending)||!Array.isArray(admitted)||pending.length>100||admitted.length>100||pending.some(item=>!presentationParticipant(item))||admitted.some(item=>!presentationParticipant(item)))throw new Error('Invalid presentation state');
 const ids=[...pending,...admitted].map(item=>(item as PresentationParticipant).connectionId);
 if(new Set(ids).size!==ids.length)throw new Error('Invalid presentation state');
 return input as PresentationState;
}
