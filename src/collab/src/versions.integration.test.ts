import { createHash, randomUUID } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { afterEach, expect, it } from 'vitest';
import * as Y from 'yjs';

import { hash } from './auth.js';
import { Persistence } from './persistence.js';
import { canonicalVersionPayload, type CaptureVersionInput } from './versions.js';
import { fixture, grantSecret, rectangle, signed } from '../tests/fixture.js';

const cleanup: Array<() => Promise<void>> = [];
afterEach(async () => {
  for (const close of cleanup.splice(0).reverse()) await close();
});

async function setup(now?: () => number) {
  const f = await fixture();
  cleanup.push(() => f.close());
  await f.pool.query(await readFile(new URL('../../backend/migrations/000015_workspace_events.up.sql', import.meta.url), 'utf8'));
  await f.pool.query(await readFile(new URL('../../backend/migrations/000018_scene_versions.up.sql', import.meta.url), 'utf8'));
  const store = new Persistence(f.pool, f.authority, { promoteV2: true, now });
  const state = await store.read(f.ref, f.principal);
  return { ...f, store, state };
}

async function grant(f: Awaited<ReturnType<typeof setup>>, input: CaptureVersionInput) {
  const bytes = canonicalVersionPayload(input);
  const now = Math.floor(Date.now() / 1000);
  const aclVersion = (await f.pool.query('SELECT acl_version::text AS value FROM workspaces WHERE id=$1', [f.ref.workspaceId])).rows[0].value;
  return signed({
    iss: 'whiteboard-api', aud: 'whiteboard-mutation', purpose: 'mutation-decision',
    iat: now, exp: now + 5, jti: randomUUID(), sub: f.principal.principalId,
    ...f.ref, principalId: f.principal.principalId, principalType: 'user',
    grantId: `membership:${f.ref.workspaceId}:${f.principal.principalId}`,
    access: 'edit', aclVersion, action: 'version:create', requestId: input.requestId,
    payloadHash: hash(bytes), expectedSequence: input.expectedSequence,
  }, grantSecret);
}

function input(sequence: string, overrides: Partial<CaptureVersionInput> = {}): CaptureVersionInput {
  return { requestId: randomUUID(), expectedSequence: sequence, pinned: false, ...overrides };
}

it('checkpointMatchesExactSequence', async () => {
  const f = await setup();
  const create = input(f.state.sequence, { label: 'Before launch' });
  const version = await f.store.captureVersion(f.ref, await grant(f, create), create);

  expect(version.documentId).toBe(f.state.documentId);
  expect(version.sequence).toBe(f.state.sequence);
  expect(version.hash).toMatch(/^[a-f0-9]{64}$/);
  const row = (await f.pool.query('SELECT content_bytes,content_hash,source_sequence::text FROM scene_versions WHERE id=$1', [version.id])).rows[0];
  expect(createHash('sha256').update(row.content_bytes).digest('hex')).toBe(row.content_hash);
  expect(row.source_sequence).toBe(f.state.sequence);

  const stale = input('1');
  await expect(f.store.captureVersion(f.ref, await grant(f, stale), stale)).rejects.toMatchObject({ code: 'CONFLICT' });
  expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('1');
});

it('checkpointRetryIsImmutable', async () => {
  const f = await setup();
  const create = input(f.state.sequence, { pinned: true });
  const token = await grant(f, create);
  const first = await f.store.captureVersion(f.ref, token, create);
  const retry = await f.store.captureVersion(f.ref, token, create);

  expect(retry).toEqual(first);
  expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('1');
  expect((await f.pool.query("SELECT count(*) FROM workspace_events WHERE source='scene_version'")).rows[0].count).toBe('1');
  expect((await f.pool.query('SELECT count(*) FROM workspace_event_outbox')).rows[0].count).toBe('1');

  const changed = { ...create, label: 'Changed retry' };
  await expect(f.store.captureVersion(f.ref, token, changed)).rejects.toMatchObject({ code: 'FORBIDDEN' });
});

it('checkpointPinsImageBeforeRetention', async () => {
  const f = await setup();
  const assetId = randomUUID();
  await f.pool.query("INSERT INTO scene_assets(id,workspace_id,scene_id,asset_kind,file_id,storage_bucket,storage_key,mime_type,byte_size) VALUES($1,$2,$3,'excalidraw_file','file','bucket','versions/file','image/png',10)", [assetId, f.ref.workspaceId, f.ref.sceneId]);
  const image = { ...rectangle('image'), type: 'image', fileId: 'file', status: 'saved', scale: [1, 1], crop: null };
  const legacy = { elements: [image], appState: {}, files: { file: { id: 'file', mimeType: 'image/png', created: 1, assetId } }, presentationOrder: [] };
  await f.pool.query('DELETE FROM collab_documents WHERE scene_id=$1', [f.ref.sceneId]);
  await f.pool.query("UPDATE scenes SET writer_mode='v1',content=$2 WHERE id=$1", [f.ref.sceneId, legacy]);
  f.state.doc.destroy();
  f.state = await f.store.read(f.ref, f.principal);

  const create = input(f.state.sequence);
  const version = await f.store.captureVersion(f.ref, await grant(f, create), create);
  expect((await f.pool.query('SELECT asset_id FROM version_assets WHERE version_id=$1', [version.id])).rows[0].asset_id).toBe(assetId);

  await f.pool.query("CREATE FUNCTION fail_version_pin() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'pin fault'; END $$; CREATE TRIGGER fail_version_pin BEFORE INSERT ON version_assets FOR EACH ROW EXECUTE FUNCTION fail_version_pin()");
  const failed = input(f.state.sequence);
  await expect(f.store.captureVersion(f.ref, await grant(f, failed), failed)).rejects.toThrow();
  expect((await f.pool.query('SELECT count(*) FROM scene_versions')).rows[0].count).toBe('1');
});

it('versionSurvivesCompaction', async () => {
  const f = await setup();
  const client = new Y.Doc({ gc: false });
  Y.applyUpdate(client, Y.encodeStateAsUpdate(f.state.doc));
  const vector = Y.encodeStateVector(client);
  client.getMap('appState').set('viewBackgroundColor', '#123456');
  const update = Y.encodeStateAsUpdate(client, vector);
  client.destroy();
  const receipt = await f.store.appendUpdate(f.ref, f.principal, randomUUID(), update);
  const create = input(receipt.sequence);
  const token = await grant(f, create);
  const [version] = await Promise.all([f.store.captureVersion(f.ref, token, create), f.store.compactScene(f.ref)]);

  const stored = (await f.pool.query('SELECT content FROM scene_versions WHERE id=$1', [version.id])).rows[0].content;
  expect(stored.appState.viewBackgroundColor).toBe('#123456');
  expect((await f.pool.query('SELECT count(*) FROM collab_updates')).rows[0].count).toBe('0');
});

it('automaticCheckpointUsesLatestSequenceAndOneMinuteFloor', async () => {
  let now = Date.now();
  const f = await setup(() => now);
  const first = await f.store.captureAutomaticVersion(f.ref);
  expect(first?.sequence).toBe(f.state.sequence);

  const client = new Y.Doc({ gc: false });
  Y.applyUpdate(client, Y.encodeStateAsUpdate(f.state.doc));
  const vector = Y.encodeStateVector(client);
  client.getMap('appState').set('gridSize', 20);
  const receipt = await f.store.appendUpdate(f.ref, f.principal, randomUUID(), Y.encodeStateAsUpdate(client, vector));
  client.destroy();

  expect(await f.store.captureAutomaticVersion(f.ref)).toBeUndefined();
  now += 120_000;
  const second = await f.store.captureAutomaticVersion(f.ref);
  expect(second?.sequence).toBe(receipt.sequence);
  expect((await f.pool.query("SELECT count(*) FROM scene_versions WHERE kind='automatic'")).rows[0].count).toBe('2');
});
