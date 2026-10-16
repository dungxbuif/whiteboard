import { describe, it, expect } from 'vitest';
import * as Y from 'yjs';
import { ExcalidrawAdapter, LOCAL_ORIGIN } from './excalidraw-adapter';
import { rectangle, scene } from './fixtures';
describe('collab adapter', () => {
  it('remote updates render without echo and preserve native binding fields', () => {
    const a = new ExcalidrawAdapter(); const b = new ExcalidrawAdapter();
    const node = { ...rectangle(), boundElements: [{ id: 'label', type: 'text' as const }] };
    const label = { ...rectangle('label'), type: 'text' as const, fontSize: 20, fontFamily: 1, text: 'Hello', originalText: 'Hello', textAlign: 'left' as const, verticalAlign: 'top' as const, containerId: 'shape', autoResize: true, lineHeight: 1.25 };
    a.applyLocalScene(scene([node, label]));
    let echoes = 0; b.doc.on('update', (_bytes, origin) => { if (origin === LOCAL_ORIGIN) echoes++; });
    b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    b.applyLocalScene(b.content());
    expect(b.content().elements).toEqual([node, label]); expect(echoes).toBe(0);
  });
  it('concurrent field edits, tombstones and reorder converge without losing independent fields', () => {
    const a = new ExcalidrawAdapter(); a.applyLocalScene(scene([rectangle('one'), rectangle('two')]));
    const b = new ExcalidrawAdapter(); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    const left = a.content(); left.elements[0].x = 75; left.elements.reverse(); a.applyLocalScene(left);
    const right = b.content(); right.elements[0].y = 44; right.elements[1].isDeleted = true; b.applyLocalScene(right);
    a.applyRemote(Y.encodeStateAsUpdate(b.doc)); b.applyRemote(Y.encodeStateAsUpdate(a.doc));
    expect(a.content()).toEqual(b.content());
    expect(a.content().elements.find(e => e.id === 'one')).toMatchObject({ x: 75, y: 44 });
    expect(a.content().elements.find(e => e.id === 'two')?.isDeleted).toBe(true);
  });
});
