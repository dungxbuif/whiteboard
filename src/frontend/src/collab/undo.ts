import type { SceneContent } from '@collab/contracts';
import type * as Y from 'yjs';
import { ExcalidrawAdapter } from './excalidraw-adapter';
import { mergeSceneChanges } from './scene-diff';

type HistoryEntry = { base: SceneContent; desired: SceneContent; bytes: number };
const CAPTURE_MS = 500;
const MAX_ENTRIES = 25;
const MAX_HISTORY_BYTES = 32 * 1024 * 1024;
const copy = <T,>(value: T): T => structuredClone(value);
const same = (a: SceneContent, b: SceneContent) => JSON.stringify(a) === JSON.stringify(b);
const size = (base: SceneContent, desired: SceneContent) => new TextEncoder().encode(JSON.stringify([base, desired])).length;

export class LocalUndo {
  private undoStack: HistoryEntry[] = [];
  private redoStack: HistoryEntry[] = [];
  private undoBytes = 0;
  private lastCapture = 0;

  constructor(private readonly adapter: ExcalidrawAdapter) {}

  apply(input: SceneContent): void {
    const base = this.adapter.content(), desired = copy(input);
    if (same(base, desired)) return;
    const now = Date.now(), previous = this.undoStack.at(-1);
    if (previous && now - this.lastCapture <= CAPTURE_MS) {
      this.undoBytes -= previous.bytes;
      previous.desired = desired;
      previous.bytes = size(previous.base, desired);
      this.undoBytes += previous.bytes;
    } else {
      const entry = { base: copy(base), desired, bytes: size(base, desired) };
      this.undoStack.push(entry); this.undoBytes += entry.bytes;
    }
    this.lastCapture = now; this.redoStack = []; this.prune();
    this.adapter.applyLocalScene(input);
  }

  undo(): void {
    const entry = this.undoStack.pop(); if (!entry) return;
    this.undoBytes -= entry.bytes; this.lastCapture = 0;
    this.adapter.applyLocalScene(mergeSceneChanges(entry.desired, entry.base, this.adapter.content()));
    this.redoStack.push(entry);
  }

  redo(): void {
    const entry = this.redoStack.pop(); if (!entry) return;
    const current = this.adapter.content();
    const revived = new Set(entry.desired.elements.filter(element => !entry.base.elements.some(old => old.id === element.id)).map(element => element.id));
    const merged = mergeSceneChanges(entry.base, entry.desired, current);
    merged.elements = merged.elements.map(element => revived.has(element.id) && element.isDeleted
      ? { ...element, isDeleted: entry.desired.elements.find(value => value.id === element.id)?.isDeleted ?? false }
      : element);
    this.adapter.applyLocalScene(merged);
    this.undoStack.push(entry); this.undoBytes += entry.bytes; this.lastCapture = 0; this.prune();
  }

  canUndo(): boolean { return this.undoStack.length > 0; }
  canRedo(): boolean { return this.redoStack.length > 0; }
  stopCapturing(): void { this.lastCapture = 0; }
  destroy(): void { this.undoStack = []; this.redoStack = []; this.undoBytes = 0; this.lastCapture = 0; }

  private prune(): void {
    while (this.undoStack.length > MAX_ENTRIES || this.undoBytes > MAX_HISTORY_BYTES) {
      const removed = this.undoStack.shift(); if (!removed) break; this.undoBytes -= removed.bytes;
    }
  }
}

export function createLocalUndo(input: ExcalidrawAdapter | Y.Doc) {
  return new LocalUndo(input instanceof ExcalidrawAdapter ? input : new ExcalidrawAdapter(input));
}
