import * as Y from 'yjs';
import { validateSceneContent, type SceneContent, type NativeElement } from '@collab/contracts';
export const LOCAL_ORIGIN = Symbol('canvas');
export const AGENT_ORIGIN = Symbol('agent');
export const REMOTE_ORIGIN = Symbol('remote');
export const REPLAY_ORIGIN = Symbol('replay');
const equal = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
function fields(map: Y.Map<unknown>, value: Record<string, unknown>) {
  for (const [key, item] of Object.entries(value)) if (!equal(map.get(key), item)) map.set(key, structuredClone(item));
  for (const key of map.keys()) if (!(key in value)) map.delete(key);
}
function order(array: Y.Array<string>, ids: string[]) {
  if (equal([...new Set(array.toArray())], ids)) return;
  if (array.length) array.delete(0, array.length);
  if (ids.length) array.push(ids);
}
export class ExcalidrawAdapter {
  constructor(readonly doc = new Y.Doc({ gc: false })) {}
  content(): SceneContent {
    const maps = this.doc.getMap<Y.Map<unknown>>('elements');
    const ids = [...new Set(this.doc.getArray<string>('order').toArray())].filter(id => maps.has(id));
    ids.push(...[...maps.keys()].filter(id => !ids.includes(id)).sort());
    const presentationOrder = [...new Set(this.doc.getArray<string>('presentationOrder').toArray())];
    return structuredClone({ elements: ids.map(id => maps.get(id)!.toJSON()), appState: this.doc.getMap('appState').toJSON(), files: this.doc.getMap('files').toJSON(), ...(presentationOrder.length ? { presentationOrder } : {}) }) as SceneContent;
  }
  applyLocalScene(input: SceneContent): void { this.apply(input, LOCAL_ORIGIN); }
  private apply(input: SceneContent, origin: unknown): void {
    const content = validateSceneContent(input);
    this.doc.transact(() => {
      const maps = this.doc.getMap<Y.Map<unknown>>('elements');
      const ids = new Set(content.elements.map(e => e.id));
      for (const element of content.elements) {
        let map = maps.get(element.id);
        if (!map) { map = new Y.Map(); maps.set(element.id, map); }
        fields(map, element as unknown as Record<string, unknown>);
      }
      // Retain removed IDs as tombstones so concurrent bindings stay resolvable.
      for (const [id, map] of maps) if (!ids.has(id) && !map.get('isDeleted')) map.set('isDeleted', true);
      order(this.doc.getArray('order'), [...content.elements.map(e => e.id), ...[...maps.keys()].filter(id => !ids.has(id)).sort()]);
      fields(this.doc.getMap('appState'), content.appState);
      const files = this.doc.getMap<Y.Map<unknown>>('files');
      for (const [id, file] of Object.entries(content.files)) {
        let map = files.get(id); if (!map) { map = new Y.Map(); files.set(id, map); }
        fields(map, file);
      }
      order(this.doc.getArray('presentationOrder'), content.presentationOrder ?? []);
    }, origin);
  }
  async applyAgentElements(elements: NativeElement[]): Promise<void> {
    const current = this.content();
    const ids = new Set(current.elements.map(e => e.id));
    if (elements.some(e => ids.has(e.id))) throw new Error('Generated element ID already exists');
    this.apply({ ...current, elements: [...current.elements, ...elements] }, AGENT_ORIGIN);
  }
  reorderPresentation(ids: string[]): void { this.applyLocalScene({ ...this.content(), presentationOrder: ids }); }
  applyRemote(update: Uint8Array): void { Y.applyUpdate(this.doc, update, REMOTE_ORIGIN); }
  reapplyIntent(baseInput:SceneContent,desiredInput:SceneContent):Uint8Array {
    const base=validateSceneContent(baseInput),desired=validateSceneContent(desiredInput),vector=Y.encodeStateVector(this.doc);
    this.doc.transact(()=>{
      const elements=this.doc.getMap<Y.Map<unknown>>('elements'),before=new Map(base.elements.map(value=>[value.id,value]));
      for(const value of desired.elements){const old=before.get(value.id);if(!old){let map=elements.get(value.id);if(!map){map=new Y.Map();elements.set(value.id,map);}for(const [key,field] of Object.entries(value))map.set(key,structuredClone(field));continue;}const map=elements.get(value.id);if(!map)continue;const oldFields=old as unknown as Record<string,unknown>,next=value as unknown as Record<string,unknown>,keys=new Set([...Object.keys(oldFields),...Object.keys(next)]);if([...keys].some(key=>!equal(oldFields[key],next[key]))&&next.isDeleted===false)keys.add('isDeleted');for(const key of keys)if(!equal(oldFields[key],next[key])||key==='isDeleted'){if(key in next)map.set(key,structuredClone(next[key]));else map.delete(key);}}
      for(const old of base.elements)if(!desired.elements.some(value=>value.id===old.id))elements.get(old.id)?.set('isDeleted',true);
      if(!equal(base.elements.map(value=>value.id),desired.elements.map(value=>value.id))){const array=this.doc.getArray<string>('order');if(array.length)array.delete(0,array.length);array.push(desired.elements.map(value=>value.id));}
      const app=this.doc.getMap('appState');for(const key of new Set([...Object.keys(base.appState),...Object.keys(desired.appState)]))if(!equal(base.appState[key as keyof typeof base.appState],desired.appState[key as keyof typeof desired.appState])){if(key in desired.appState)app.set(key,structuredClone(desired.appState[key as keyof typeof desired.appState]));else app.delete(key);}
      const files=this.doc.getMap<Y.Map<unknown>>('files');for(const [id,value] of Object.entries(desired.files)){const old=base.files[id],oldFields=(old??{}) as unknown as Record<string,unknown>,next=value as unknown as Record<string,unknown>;if(equal(old,value))continue;let map=files.get(id);if(!map){map=new Y.Map();files.set(id,map);}for(const key of new Set([...Object.keys(oldFields),...Object.keys(next)]))if(!equal(oldFields[key],next[key])){if(key in next)map.set(key,structuredClone(next[key]));else map.delete(key);}}
      for(const id of Object.keys(base.files))if(!(id in desired.files))files.delete(id);
      if(!equal(base.presentationOrder??[],desired.presentationOrder??[])){const array=this.doc.getArray<string>('presentationOrder');if(array.length)array.delete(0,array.length);if(desired.presentationOrder?.length)array.push(desired.presentationOrder);}
    },REPLAY_ORIGIN);
    return Y.encodeStateAsUpdate(this.doc,vector);
  }
}
