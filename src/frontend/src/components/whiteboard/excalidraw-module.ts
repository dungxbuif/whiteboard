export type ExcalidrawModule = typeof import('@excalidraw/excalidraw');

export function loadExcalidrawModule(): Promise<ExcalidrawModule> {
  return import('@excalidraw/excalidraw');
}
