import { it, expect } from 'vitest';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { createLocalUndo } from './undo';
import { rectangle, scene } from './fixtures';
it('collab local undo preserves another actor edits and does not undo agent additions', async () => {
  const a = new ExcalidrawAdapter(); a.applyLocalScene(scene());
  const undo = createLocalUndo(a);
  const left = a.content(); left.elements[0].x = 20;
  undo.apply(left);
  const right = a.content(); right.elements[0].y = 55; right.elements.push(rectangle('agent')); a.applyLocalScene(right);
  undo.undo();
  expect(a.content().elements[0]).toMatchObject({ x: 0, y: 55 });
  expect(a.content().elements.find(element => element.id === 'agent')?.isDeleted).toBe(false);
  undo.redo();
  expect(a.content().elements[0]).toMatchObject({ x: 20, y: 55 });
  expect(a.content().elements.find(element => element.id === 'agent')?.isDeleted).toBe(false);
});

it('undo tombstones a local addition and redo revives the same stable element map',()=>{
  const adapter=new ExcalidrawAdapter(),undo=createLocalUndo(adapter);
  undo.apply(scene([rectangle('local')]));
  const map=adapter.doc.getMap('elements').get('local');
  undo.undo();expect(adapter.content().elements).toMatchObject([{id:'local',isDeleted:true}]);
  undo.redo();expect(adapter.content().elements).toMatchObject([{id:'local',isDeleted:false}]);
  expect(adapter.doc.getMap('elements').get('local')).toBe(map);
});

it('clear drops acknowledged restore history without mutating the restored document',()=>{
 const adapter=new ExcalidrawAdapter();adapter.applyLocalScene(scene());const undo=createLocalUndo(adapter);
 const changed=adapter.content();changed.elements[0].x=44;undo.apply(changed);undo.undo();
 expect(undo.canRedo()).toBe(true);const before=adapter.content();undo.clear();
 expect(undo.canUndo()).toBe(false);expect(undo.canRedo()).toBe(false);expect(adapter.content()).toEqual(before);
});
