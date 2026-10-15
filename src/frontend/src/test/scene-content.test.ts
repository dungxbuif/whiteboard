import { describe, expect, it } from "vitest";

import { normalizeSceneContent } from "@/lib/scene-content";

describe("normalizeSceneContent", () => {
  it("converts JSON-persisted collaborators to an Excalidraw Map", () => {
    const content = {
      elements: [],
      appState: { collaborators: {} },
      files: {},
    };

    const normalized = normalizeSceneContent(content);

    expect(normalized.appState.collaborators).toBeInstanceOf(Map);
  });

  it("preserves an existing collaborators Map", () => {
    const collaborators = new Map();
    const content = { elements: [], appState: { collaborators }, files: {} };

    expect(normalizeSceneContent(content).appState.collaborators).toBe(collaborators);
  });
});
