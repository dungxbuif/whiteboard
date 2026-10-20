import { syncPresentationOrder, type SceneContent } from "@/store/scene-store";

export const MAX_SCENE_IMPORT_BYTES = 25 * 1024 * 1024;
const ELEMENT_TYPES = new Set(["rectangle", "diamond", "ellipse", "text", "line", "arrow", "freedraw", "image", "frame", "magicframe"]);
const INVALID_FILE = "Choose a valid Excalidraw scene file.";
const isRecord = (value: unknown): value is Record<string, unknown> => !!value && typeof value === "object" && !Array.isArray(value);
const isNumber = (value: unknown) => typeof value === "number" && Number.isFinite(value);
const isString = (value: unknown) => typeof value === "string";
const isPoint = (value: unknown) => Array.isArray(value) && value.length === 2 && value.every(isNumber);
const isStringArray = (value: unknown) => Array.isArray(value) && value.every(isString);
const isBinding = (value: unknown) => isRecord(value) && isString(value.elementId) && isNumber(value.focus) && isNumber(value.gap) && (value.fixedPoint == null || isPoint(value.fixedPoint));

function validOptionalFields(element: Record<string, unknown>) {
  if ("font" in element && (typeof element.font !== "string" || !/^\d+(?:\.\d+)?px\s+\S/.test(element.font) || parseFloat(element.font) <= 0)) return false;
  const numeric = ["angle", "seed", "version", "versionNonce", "updated", "strokeWidth", "roughness", "opacity", "fontSize", "fontFamily", "lineHeight"];
  const strings = ["strokeColor", "backgroundColor", "fillStyle", "strokeStyle", "text", "originalText", "textAlign", "verticalAlign", "status"];
  const nullableStrings = ["frameId", "containerId", "index", "link", "name", "fileId", "startArrowhead", "endArrowhead"];
  const booleans = ["isDeleted", "locked", "autoResize", "simulatePressure", "elbowed", "startIsSpecial", "endIsSpecial"];
  if (!numeric.every(key => element[key] == null || isNumber(element[key])) ||
      !strings.every(key => element[key] == null || isString(element[key])) ||
      !nullableStrings.every(key => element[key] == null || isString(element[key])) ||
      !booleans.every(key => element[key] == null || typeof element[key] === "boolean")) return false;
  const checks: Record<string, (value: unknown) => boolean> = {
    groupIds: isStringArray, boundElementIds: isStringArray,
    boundElements: value => Array.isArray(value) && value.every(item => isRecord(item) && isString(item.id) && ["arrow", "text"].includes(String(item.type))),
    startBinding: isBinding, endBinding: isBinding,
    pressures: value => Array.isArray(value) && value.every(isNumber),
    lastCommittedPoint: isPoint, scale: isPoint,
    roundness: value => isRecord(value) && isNumber(value.type) && (value.value == null || isNumber(value.value)),
    crop: value => isRecord(value) && ["x", "y", "width", "height", "naturalWidth", "naturalHeight"].every(key => isNumber(value[key])),
    fixedSegments: value => Array.isArray(value) && value.every(segment => isRecord(segment) && isPoint(segment.start) && isPoint(segment.end) && isNumber(segment.index)),
    customData: isRecord,
  };
  return Object.entries(checks).every(([key, validate]) => element[key] == null || validate(element[key]));
}

export async function parseSceneImport(file: File): Promise<SceneContent> {
  if (file.size > MAX_SCENE_IMPORT_BYTES) throw new Error("Scene files must be 25 MiB or smaller.");
  let data: unknown;
  try { data = JSON.parse(await file.text()); } catch { throw new Error(INVALID_FILE); }
  if (!isRecord(data) || data.type !== "excalidraw" || data.version !== 2 || !Array.isArray(data.elements)) throw new Error(INVALID_FILE);
  const ids = new Set<string>();
  for (const element of data.elements) {
    if (isRecord(element) && element.type === "embeddable") {
      throw new Error("Web embeds are not supported in imported scenes. Remove them from the drawing and try again.");
    }
    if (!isRecord(element) || typeof element.id !== "string" || !element.id || ids.has(element.id) || !ELEMENT_TYPES.has(String(element.type)) ||
      !["x", "y", "width", "height"].every(key => typeof element[key] === "number" && Number.isFinite(element[key])) ||
      (element.width as number) < 0 || (element.height as number) < 0 || !validOptionalFields(element)) throw new Error("Scene contains an invalid or duplicate element.");
    ids.add(element.id);
    if (["line", "arrow", "freedraw"].includes(String(element.type)) && (!Array.isArray(element.points) || !element.points.every(point => Array.isArray(point) && point.length === 2 && point.every(value => typeof value === "number" && Number.isFinite(value))))) throw new Error("Scene contains invalid element points.");
    if (element.type === "text" && typeof element.text !== "string") throw new Error("Scene contains invalid text element.");
  }
  const files = data.files ?? {};
  if (!isRecord(files)) throw new Error("Scene contains invalid images.");
  for (const [id, image] of Object.entries(files)) {
    if (!isRecord(image) || image.id !== id || typeof image.dataURL !== "string" || !/^data:image\/(png|jpeg|gif|webp);base64,[A-Za-z0-9+/=\s]+$/.test(image.dataURL)) throw new Error("Scene images must be embedded PNG, JPEG, GIF or WebP images.");
  }
  const appState: Record<string, unknown> = {};
  if (isRecord(data.appState)) {
    if (typeof data.appState.viewBackgroundColor === "string") appState.viewBackgroundColor = data.appState.viewBackgroundColor;
    if (typeof data.appState.gridSize === "number" && Number.isFinite(data.appState.gridSize) && data.appState.gridSize > 0) appState.gridSize = data.appState.gridSize;
  }
  const order = Array.isArray(data.presentationOrder) ? data.presentationOrder.filter((id): id is string => typeof id === "string") : [];
  return { elements: data.elements, appState, files, presentationOrder: syncPresentationOrder(data.elements, order) };
}
