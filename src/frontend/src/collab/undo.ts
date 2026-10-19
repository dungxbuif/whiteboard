import * as Y from 'yjs';
import { LOCAL_ORIGIN } from './excalidraw-adapter';
export function createLocalUndo(doc: Y.Doc) {
  return new Y.UndoManager([doc.getMap('elements'), doc.getArray('order'), doc.getMap('appState'), doc.getArray('presentationOrder')], { trackedOrigins: new Set([LOCAL_ORIGIN]), captureTimeout: 500 });
}
