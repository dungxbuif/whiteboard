import { it, expect } from 'vitest';
import * as Y from 'yjs';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { createLocalUndo } from './undo';
import { scene } from './fixtures';
it('collab local undo preserves another actor edits and does not undo agent additions', async () => {
  const a = new ExcalidrawAdapter(); a.applyLocalScene(scene());
  const b = new ExcalidrawAdapter(); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
  const undo = createLocalUndo(a.doc);
  const left = a.content(); left.elements[0].x = 20; a.applyLocalScene(left);
  const right = b.content(); right.elements[0].y = 55; b.applyLocalScene(right);
  a.applyRemote(Y.encodeStateAsUpdate(b.doc)); undo.undo();
  expect(a.content().elements[0]).toMatchObject({ x: 0, y: 55 });
});
