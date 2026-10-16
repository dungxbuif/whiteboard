export function encodeRecoveryBytes(bytes: Uint8Array): string {
  const chunks: string[] = [];
  const chunkSize = 8192;
  for (let offset = 0; offset < bytes.length; offset += chunkSize) chunks.push(String.fromCharCode(...bytes.subarray(offset, offset + chunkSize)));
  return btoa(chunks.join(''));
}
export function decodeRecoveryBytes(value: string): Uint8Array { return Uint8Array.from(atob(value), c => c.charCodeAt(0)); }
export async function encodeRecoveryBlob(blob: Blob): Promise<string> { return encodeRecoveryBytes(new Uint8Array(await blob.arrayBuffer())); }
