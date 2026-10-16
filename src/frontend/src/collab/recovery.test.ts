import { it, expect } from 'vitest';
import { encodeRecoveryBytes, decodeRecoveryBytes, encodeRecoveryBlob } from './recovery';
it('collab recovery JSON roundtrips exact update and Blob bytes including zero and non-UTF8 bytes', async () => {
  const bytes = new Uint8Array([0, 1, 127, 128, 254, 255]);
  const blob = new Blob([bytes], { type: 'image/png' });
  // jsdom Blob exposes FileReader, while browser Blob also exposes arrayBuffer.
  Object.defineProperty(blob, 'arrayBuffer', { value: () => new Promise<ArrayBuffer>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result as ArrayBuffer); reader.onerror = reject; reader.readAsArrayBuffer(blob); }) });
  const value = { updateBase64: encodeRecoveryBytes(bytes), blobBase64: await encodeRecoveryBlob(blob) };
  const restored = JSON.parse(JSON.stringify(value));
  expect(Array.from(decodeRecoveryBytes(restored.updateBase64))).toEqual(Array.from(bytes));
  expect(Array.from(decodeRecoveryBytes(restored.blobBase64))).toEqual(Array.from(bytes));
});
