import { test, expect, type Page } from "@playwright/test";

test.use({ reducedMotion: "reduce" });

const workspace = { id: "workspace-style", name: "Design studio", ownerId: "user-style", createdAt: "2026-10-16T00:00:00Z" };
const scene = { id: "scene-style", workspaceId: workspace.id, folderId: null, title: "System architecture", content: { elements: [], appState: {}, files: {} }, contentVersion: 1, isStarred: true, createdAt: "2026-10-16T00:00:00Z", updatedAt: "2026-10-16T00:00:00Z" };

async function setup(page: Page, options: { longName?: boolean; empty?: boolean } = {}) {
  let currentScene = scene;
  await page.addInitScript(() => localStorage.setItem("auth_token", "style-test"));
  await page.route("**/api/v1/users/me", route => route.fulfill({ json: { id: "user-style", email: "designer@example.com", name: "Designer" } }));
  await page.route("**/api/v1/workspaces", route => route.fulfill({ json: [{ ...workspace, name: options.longName ? "A very long workspace name for architecture and distributed systems" : workspace.name }] }));
  await page.route("**/api/v1/folders**", route => route.fulfill({ json: [] }));
  await page.route("**/api/v1/scenes**", route => {
    if (route.request().method() === "POST") {
      currentScene = { ...scene, ...route.request().postDataJSON() };
      return route.fulfill({ json: currentScene });
    }
    return route.fulfill({ json: /\/scenes\/[^/]+/.test(new URL(route.request().url()).pathname) ? currentScene : options.empty ? [] : [currentScene] });
  });
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
}

test("legacy desktop shell, collapse and original light/dark portal colors", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 960 });
  await setup(page);
  await expect(page.getByRole("navigation", { name: "Workspace navigation", exact: true })).toBeVisible();
  const nav = await page.getByRole("navigation", { name: "Workspace navigation", exact: true }).boundingBox();
  const search = await page.getByRole("textbox", { name: "Search scenes and folders" }).boundingBox();
  expect(search!.x + search!.width).toBeLessThanOrEqual(nav!.x + nav!.width);
  await expect(page.getByRole("button", { name: "Import scenes" })).toBeVisible();
  await expect(page).toHaveScreenshot("dashboard-legacy-light.png");
  const toggle = page.getByRole("button", { name: "Collapse sidebar" });
  await toggle.click();
  await expect(page.getByRole("navigation", { name: "Workspace navigation", exact: true })).toBeHidden();
  await page.getByRole("button", { name: "Expand sidebar" }).click();
  await page.getByRole("button", { name: "Account menu" }).click();
  await page.getByRole("menuitem", { name: "Dark theme" }).click();
  await expect(page.locator("main.dashboard-theme")).toHaveAttribute("data-theme", "dark");
  await page.getByRole("button", { name: "New folder" }).click();
  await expect(page.getByRole("dialog")).toHaveCSS("background-color", "rgb(10, 10, 10)");
  await expect(page.getByRole("dialog")).toHaveCSS("color", "rgb(250, 250, 250)");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("button", { name: "New folder" })).toBeFocused();
  await expect(page).toHaveScreenshot("dashboard-legacy-dark.png");
  await page.reload();
  await expect(page.locator("main.dashboard-theme")).toHaveAttribute("data-theme", "dark");
  await page.getByRole("button", { name: "Open", exact: true }).click();
  await expect(page.getByRole("region", { name: "Excalidraw canvas" })).toBeVisible();
  await expect(page.locator(".dashboard-theme")).toHaveCount(0);
  await expect(page.locator("html")).not.toHaveClass(/dark/);

});

test("mobile drawer restores keyboard focus, and long names do not overflow", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await setup(page, { longName: true, empty: true });
  const toggle = page.getByRole("button", { name: "Open navigation" });
  await toggle.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Workspace navigation" })).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Workspace navigation" })).toHaveCSS("left", "0px");
  const drawer = page.getByRole("dialog", { name: "Workspace navigation" });
  const switcher = await drawer.getByRole("combobox").boundingBox();
  const drawerBox = await drawer.boundingBox();
  expect(switcher!.x + switcher!.width).toBeLessThanOrEqual(drawerBox!.x + drawerBox!.width);
  await expect(page).toHaveScreenshot("dashboard-legacy-mobile-drawer.png");
  await page.keyboard.press("Escape");
  await expect(toggle).toBeFocused();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await expect(page.getByText("No scenes in this location.")).toBeVisible();
  await expect(page).toHaveScreenshot("dashboard-legacy-mobile.png");
});

test("import rejects invalid/oversized files without writes, then creates editable content atomically", async ({ page }) => {
  await setup(page, { empty: true });
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  const writes: unknown[] = [];
  page.on("request", request => { if (request.method() === "POST") writes.push(request.postDataJSON()); });
  const input = page.getByLabel("Scene file");
  await input.setInputFiles({ name: "bad.excalidraw", mimeType: "application/json", buffer: Buffer.from("bad") });
  await expect(page.getByRole("alert")).toContainText("valid Excalidraw");
  await input.setInputFiles({ name: "large.excalidraw", mimeType: "application/json", buffer: Buffer.alloc(25 * 1024 * 1024 + 1) });
  await expect(page.getByRole("alert")).toContainText("25 MiB");
  expect(writes).toEqual([]);
  const elements = [{ id: "shape", type: "rectangle", x: 10, y: 10, width: 100, height: 100 }];
  await input.setInputFiles({ name: "broken-bindings.excalidraw", mimeType: "application/json", buffer: Buffer.from(JSON.stringify({ type: "excalidraw", version: 2, elements: [{ ...elements[0], boundElementIds: {} }] })) });
  await expect(page.getByRole("alert")).toContainText("invalid or duplicate element");
  expect(writes).toEqual([]);
  await input.setInputFiles({ name: "Imported drawing.excalidraw", mimeType: "application/json", buffer: Buffer.from(JSON.stringify({ type: "excalidraw", version: 2, elements, appState: {}, files: {} })) });
  await expect(page).toHaveURL(/\/w\/workspace-style\/scenes\/scene-style/);
  expect(writes).toEqual([{ title: "Imported drawing", folderId: null, content: { elements, appState: {}, files: {}, presentationOrder: [] } }]);
  await expect(page.getByRole("region", { name: "Excalidraw canvas" })).toBeVisible();
  await expect(page.getByRole("status")).toHaveText("Saved");
  expect(errors).toEqual([]);
});
