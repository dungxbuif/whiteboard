import { it, expect } from 'vitest';
import * as Y from 'yjs';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { rectangle, scene } from './fixtures';
it('collab AI additions and presentation reorder persist through Yjs', async () => {
  const a = new ExcalidrawAdapter();
  a.applyLocalScene(scene([{ ...rectangle('first'), type: 'frame', name: 'First' }, { ...rectangle('second'), type: 'frame', name: 'Second' }]));
  await a.applyAgentElements([rectangle('agent')]); a.reorderPresentation(['second', 'first']);
  const b = new ExcalidrawAdapter(); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
  expect(b.content().elements.map(e => e.id)).toEqual(['first', 'second', 'agent']);
  expect(b.content().presentationOrder).toEqual(['second', 'first']);
});

import { vi, afterEach } from 'vitest';
import useSceneStore from '@/store/scene-store';
import api from '@/lib/api';
import { registerContentWriter, registerLegacyConflict } from './writers';
afterEach(() => { useSceneStore.getState().reset(); vi.restoreAllMocks(); });
it('collab scene-store and presentation writers never send an HTTP content PATCH while admitted', async () => {
  const adapter = new ExcalidrawAdapter(); const content = scene([{ ...rectangle('first'), type: 'frame', name: 'First' }, { ...rectangle('second'), type: 'frame', name: 'Second' }]); adapter.applyLocalScene(content);
  const patch = vi.spyOn(api, 'patch'); const remove = registerContentWriter('w', 's', next => adapter.applyLocalScene(next as never));
  useSceneStore.setState({ activeScene: { id: 's', workspaceId: 'w', writerMode: 'v2', content } as never });
  useSceneStore.getState().updatePresentationOrder('w', 's', ['second', 'first']); await useSceneStore.getState().flushSave();
  expect(adapter.content().presentationOrder).toEqual(['second', 'first']); expect(patch).not.toHaveBeenCalled(); remove();
  useSceneStore.getState().scheduleSave('w', 's', content); await useSceneStore.getState().flushSave();
  expect(patch).not.toHaveBeenCalled(); expect(useSceneStore.getState().saveStatus).toBe('error');
});

it('collab concurrent promotion rejects legacy save confirmation and retains the recovery draft', async () => {
  const content = scene(); const conflict = { response: { status: 409, data: { code: 'COLLAB_REQUIRED' } } };
  const patch = vi.spyOn(api, 'patch').mockRejectedValue(conflict); const recovered = vi.fn(); const remove = registerLegacyConflict('w', 's', recovered);
  useSceneStore.setState({ activeScene: { id: 's', workspaceId: 'w', writerMode: 'v1', content } as never });
  useSceneStore.getState().scheduleSave('w', 's', content);
  await expect(useSceneStore.getState().flushSave()).rejects.toBe(conflict);
  expect(recovered).toHaveBeenCalledWith(content); expect(useSceneStore.getState().activeScene?.writerMode).toBe('v2');
  useSceneStore.getState().scheduleSave('w', 's', content); await useSceneStore.getState().flushSave(); expect(patch).toHaveBeenCalledTimes(1); remove();
});
