import type { SceneContent, NativeElement } from '@collab/contracts';
const equal = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
// Apply only fields actually changed by this renderer, preserving later remote fields.
export function mergeSceneChanges(base: SceneContent, desired: SceneContent, current: SceneContent): SceneContent {
  const result = structuredClone(current), before = new Map(base.elements.map(e => [e.id, e]));
  const elements = new Map(result.elements.map(e => [e.id, e]));
  for (const value of desired.elements) {
    const old = before.get(value.id), live = elements.get(value.id);
    if (!old) { if (!live) elements.set(value.id, structuredClone(value)); continue; }
    if (!live) continue;
    const next = { ...live } as unknown as Record<string, unknown>;
    for (const [key, field] of Object.entries(value)) if (!equal((old as unknown as Record<string, unknown>)[key], field)) next[key] = structuredClone(field);
    for (const key of Object.keys(old)) if (!(key in value)) delete next[key];
    elements.set(value.id, next as unknown as NativeElement);
  }
  for (const old of base.elements) if (!desired.elements.some(e => e.id === old.id) && elements.has(old.id)) elements.set(old.id, { ...elements.get(old.id)!, isDeleted: true });
  const requestedOrder = desired.elements.map(e => e.id);
  const order = equal(base.elements.map(e => e.id), requestedOrder) ? result.elements.map(e => e.id) : requestedOrder;
  result.elements = [...new Set([...order, ...elements.keys()])].map(id => elements.get(id)!).filter(Boolean);
  for (const [key, value] of Object.entries(desired.appState)) if (!equal(base.appState[key as keyof typeof base.appState], value)) (result.appState as Record<string, unknown>)[key] = value;
  result.files = { ...result.files, ...desired.files };
  if (!equal(base.presentationOrder ?? [], desired.presentationOrder ?? [])) result.presentationOrder = desired.presentationOrder;
  return result;
}
