import { it, expect } from 'vitest';
import { uploadBlob } from './assets';
const scope = { principalId: 'alice', workspaceId: 'w', sceneId: 's', documentId: 'd' };
it('collab asset upload uses actual Blob size and returned Content-Type; commits only after successful PUT', async () => {
  const blob = new Blob([new Uint8Array([1, 2, 3, 4])], { type: 'image/png' });
  const item = { scope, id: 'file', blob, mimeType: 'image/png', created: 1 };
  const capability = async (assets: { byteSize: number }[]) => { expect(assets[0].byteSize).toBe(blob.size); return { urls: { file: { assetId: 'stored', uploadUrl: 'https://storage/upload', headers: { 'Content-Type': 'image/png', 'Content-Length': '4' } } } }; };
  const put = (async (_url, init) => { expect(init?.body).toBe(blob); expect(init?.headers).toEqual({ 'Content-Type': 'image/png' }); return { ok: true }; }) as typeof fetch;
  expect(await uploadBlob(item, capability, put)).toEqual({ id: 'file', assetId: 'stored', mimeType: 'image/png', created: 1 });
  await expect(uploadBlob(item, capability, (async () => new Response(null, { status: 403 })) as typeof fetch)).rejects.toThrow(/upload/i);
});
