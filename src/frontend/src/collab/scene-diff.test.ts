import { expect, it } from 'vitest';
import { advanceLocalBaseline, mergeSceneChanges } from './scene-diff';
import { rectangle, scene } from './fixtures';

it('merges a local movement field while retaining independent remote fields', () => {
  const base = scene([{ ...rectangle('image'), x: 90 }]);
  const desired = scene([{ ...base.elements[0], x: 270 }]);
  const current = scene([{ ...base.elements[0], strokeColor: '#00aa00', y: 35 }, rectangle('remote-addition')]);
  const merged = mergeSceneChanges(base, desired, current);
  expect(merged.elements[0]).toMatchObject({ x: 270, y: 35, strokeColor: '#00aa00' });
  expect(merged.elements[1].id).toBe('remote-addition');
});

it('does not overwrite an unrelated current element when a new local ID collides', () => {
  const desired = scene([{ ...rectangle('collision'), x: 270 }]);
  const current = scene([{ ...rectangle('collision'), x: 90, strokeColor: '#00aa00' }]);
  expect(mergeSceneChanges(scene([]), desired, current).elements).toEqual(current.elements);
});

it('advances locally applied insertion intent without claiming a colliding ID', () => {
  const desired = scene([{ ...rectangle('own'), x: 90 }, { ...rectangle('collision'), x: 5 }]);
  const beforeApply = scene([{ ...rectangle('collision'), x: 800 }]);
  const baseline = advanceLocalBaseline(scene([]), desired, beforeApply);
  const projected = mergeSceneChanges(scene([]), desired, beforeApply);
  const current = { ...projected, elements: projected.elements.map(e => ({ ...e, strokeColor: '#00aa00' })) };
  const next = { ...desired, elements: desired.elements.map(e => ({ ...e, x: 270 })) };
  const merged = mergeSceneChanges(baseline, next, current);
  expect(merged.elements.find(e => e.id === 'own')).toMatchObject({ x: 270, strokeColor: '#00aa00' });
  expect(merged.elements.find(e => e.id === 'collision')).toMatchObject({ x: 800, strokeColor: '#00aa00' });
});

it('does not turn remote fields merged before projection into a later local overwrite', () => {
  const base = scene([rectangle('existing')]);
  const desired = scene([rectangle('existing'), { ...rectangle('own'), x: 90 }]);
  const remote = scene([{ ...rectangle('existing'), strokeColor: '#00aa00', y: 35 }]);
  const baseline = advanceLocalBaseline(base, desired, remote);
  const projected = mergeSceneChanges(base, desired, remote);
  const next = { ...desired, elements: desired.elements.map(e => e.id === 'own' ? { ...e, x: 270 } : e) };
  const merged = mergeSceneChanges(baseline, next, projected);
  expect(merged.elements.find(e => e.id === 'existing')).toMatchObject({ strokeColor: '#00aa00', y: 35 });
  expect(merged.elements.find(e => e.id === 'own')?.x).toBe(270);
});
