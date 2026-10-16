import { describe, expect, it } from "vitest";
import { MAX_SCENE_IMPORT_BYTES, parseSceneImport } from "../lib/scene-import";

const rectangle = { id: "rect", type: "rectangle", x: 20, y: 30, width: 100, height: 80 };
const file = (data: unknown) => new File([JSON.stringify(data)], "Diagram.excalidraw");
const drawing = { type: "excalidraw", version: 2, elements: [rectangle], appState: {}, files: {} };

describe("scene import", () => {
  it("retains editable geometry and persisted canvas state, excluding session state", async () => {
    const result = await parseSceneImport(file({ ...drawing, appState: { viewBackgroundColor: "#fafafa", collaborators: { secret: {} }, selectedElementIds: { rect: true } } }));
    expect(result.elements).toEqual([rectangle]);
    expect(result.appState).toEqual({ viewBackgroundColor: "#fafafa" });
  });
  it("rejects invalid JSON and other JSON formats", async () => {
    await expect(parseSceneImport(new File(["broken"], "bad.excalidraw"))).rejects.toThrow(/valid Excalidraw/);
    await expect(parseSceneImport(file({ elements: [] }))).rejects.toThrow(/valid Excalidraw/);
  });
  it("rejects oversized input before reading it", async () => {
    const large = file(drawing);
    Object.defineProperty(large, "size", { value: MAX_SCENE_IMPORT_BYTES + 1 });
    await expect(parseSceneImport(large)).rejects.toThrow(/25 MiB/);
  });
  it.each([
    [{ ...rectangle, x: null }],
    [rectangle, rectangle],
    [{ ...rectangle, type: "script" }],
    [{ ...rectangle, type: "line", points: [[0, "bad"]] }],
    [{ ...rectangle, boundElementIds: {} }],
    [{ ...rectangle, boundElements: [null] }],
    [{ ...rectangle, groupIds: "group" }],
    [{ ...rectangle, strokeColor: {} }],
    [{ ...rectangle, type: "text", text: "hello", font: {} }],
    [{ ...rectangle, type: "text", text: "hello", font: null }],
  ].map(elements => ({ elements })))("rejects malformed or duplicate elements: $elements", async ({ elements }) => {
    await expect(parseSceneImport(file({ ...drawing, elements }))).rejects.toThrow(/element/);
  });
  it("rejects external image URLs", async () => {
    await expect(parseSceneImport(file({ ...drawing, files: { image: { id: "image", mimeType: "image/png", dataURL: "https://untrusted.test/a.png" } } }))).rejects.toThrow(/image/);
  });
});
