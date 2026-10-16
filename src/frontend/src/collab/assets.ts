import type { AssetReference } from '@collab/contracts';
import type { PendingBlob } from './offline-store';
export type UploadGrant = { assetId: string; uploadUrl: string; headers: Record<string, string> };
export async function uploadBlob(item: PendingBlob, capability: (assets: { fileId: string; mimeType: string; byteSize: number }[]) => Promise<{ urls: Record<string, UploadGrant> }>, put: typeof fetch = fetch): Promise<AssetReference> {
  const grant = (await capability([{ fileId: item.id, mimeType: item.mimeType, byteSize: item.blob.size }])).urls[item.id];
  if (!grant?.assetId || !grant.uploadUrl) throw new Error('Upload capability missing');
  const headers = Object.fromEntries(Object.entries(grant.headers ?? {}).filter(([name]) => name.toLowerCase() !== 'content-length'));
  const contentType = Object.entries(headers).find(([name]) => name.toLowerCase() === 'content-type')?.[1];
  if (!contentType) throw new Error('Upload Content-Type missing');
  const result = await put(grant.uploadUrl, { method: 'PUT', body: item.blob, headers });
  if (!result.ok) throw new Error(`Asset upload failed (${result.status}); bytes retained for retry`);
  return { id: item.id, assetId: grant.assetId, mimeType: contentType, created: item.created };
}
