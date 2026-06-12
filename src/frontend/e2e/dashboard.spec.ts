import { test, expect } from '@playwright/test';

const screenshotPath = process.env.SCREENSHOT_PATH;
const canvasScreenshotPath = process.env.CANVAS_SCREENSHOT_PATH;
const mobileScreenshotPath = process.env.MOBILE_SCREENSHOT_PATH;
const mockUser = {
  id: '2e01116c-1198-48b8-9977-2cd66f54502e',
  email: 'test-playwright@example.com',
  name: 'Playwright User',
};
const mockWorkspaces = [
  {
    id: '2c7f253e-b06b-4ad4-988c-c4590fbc964a',
    name: 'Test Workspace',
    ownerId: mockUser.id,
    createdAt: '2026-06-11T00:00:00Z',
  },
];
const mockFolders = [
  {
    id: 'folder-product',
    workspaceId: mockWorkspaces[0].id,
    parentId: null,
    name: 'Product Boards',
    path: '/folderproduct/',
    depth: 0,
    isStarred: false,
    createdAt: '2026-06-11T00:00:00Z',
    updatedAt: '2026-06-11T00:00:00Z',
  },
  {
    id: 'folder-research',
    workspaceId: mockWorkspaces[0].id,
    parentId: 'folder-product',
    name: 'Research',
    path: '/folderproduct/folderresearch/',
    depth: 1,
    isStarred: false,
    createdAt: '2026-06-11T00:00:00Z',
    updatedAt: '2026-06-11T00:00:00Z',
  },
];
const emptySceneContent = { elements: [], appState: {}, files: {} };
let mockScenes: Array<{
  id: string;
  workspaceId: string;
  folderId: string | null;
  title: string;
  content: typeof emptySceneContent;
  contentVersion: number;
  isStarred: boolean;
  createdAt: string;
  updatedAt: string;
}>;

test.describe('Dashboard E2E Flows', () => {
  test.beforeEach(async ({ page }) => {
    mockScenes = [];
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({ json: { token: 'playwright-token' } });
    });
    await page.route('**/api/v1/users/me', async (route) => {
      await route.fulfill({ json: mockUser });
    });
    await page.route('**/api/v1/workspaces', async (route) => {
      await route.fulfill({ json: mockWorkspaces });
    });
    await page.route('**/api/v1/folders**', async (route) => {
      if (route.request().method() === 'POST') {
        const body = route.request().postDataJSON() as { name: string; parentId: string | null };
        await route.fulfill({
          json: {
            ...mockFolders[0],
            id: 'folder-created',
            name: body.name,
            parentId: body.parentId,
          },
        });
        return;
      }
      await route.fulfill({ json: mockFolders });
    });
    await page.route('**/api/v1/scenes**', async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const sceneId = url.pathname.match(/\/scenes\/([^/]+)/)?.[1];
      const body = request.postDataJSON() as Record<string, unknown> | null;

      if (request.method() === 'GET') {
        await route.fulfill({ json: mockScenes });
        return;
      }
      if (request.method() === 'POST' && url.pathname.endsWith('/scenes/')) {
        const created = {
          id: 'scene-created',
          workspaceId: mockWorkspaces[0].id,
          folderId: (body?.folderId as string | null) ?? null,
          title: body?.title as string,
          content: emptySceneContent,
          contentVersion: 1,
          isStarred: false,
          createdAt: '2026-06-12T00:00:00Z',
          updatedAt: '2026-06-12T00:00:00Z',
        };
        mockScenes = [created, ...mockScenes];
        await route.fulfill({ json: created });
        return;
      }
      if (request.method() === 'PATCH' && sceneId) {
        const current = mockScenes.find((scene) => scene.id === sceneId)!;
        const updated = {
          ...current,
          ...body,
          contentVersion: body?.content ? current.contentVersion + 1 : current.contentVersion,
        };
        mockScenes = mockScenes.map((scene) => scene.id === sceneId ? updated : scene);
        await route.fulfill({ json: updated });
        return;
      }
      if (request.method() === 'POST' && url.pathname.endsWith('/star') && sceneId) {
        const current = mockScenes.find((scene) => scene.id === sceneId)!;
        const updated = { ...current, isStarred: body?.isStarred as boolean };
        mockScenes = mockScenes.map((scene) => scene.id === sceneId ? updated : scene);
        await route.fulfill({ json: updated });
        return;
      }
      if (request.method() === 'DELETE' && sceneId) {
        mockScenes = mockScenes.filter((scene) => scene.id !== sceneId);
        await route.fulfill({ status: 204 });
        return;
      }
      await route.abort();
    });

    await page.goto('/');
  });

  async function login(page: import('@playwright/test').Page) {
    await page.getByLabel('Username').fill('admin');
    await page.getByLabel('Password').fill('admin');
    await page.getByRole('button', { name: 'Sign in' }).click();
  }

  test('should restore session and display user context', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') {
        consoleErrors.push(message.text());
      }
    });

    await expect(page.getByRole('heading', { name: 'Admin login' })).toBeVisible();
    await login(page);
    await expect(page.getByText('Workspace', { exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Test Workspace');
    if (screenshotPath) {
      await page.screenshot({ path: screenshotPath, fullPage: true });
    }
    expect(consoleErrors).toEqual([]);
  });

  test('should allow switching workspaces', async ({ page }) => {
    await login(page);
    const switcher = page.getByRole('combobox');
    await expect(switcher).toBeVisible();
    await switcher.click();
    await expect(page.getByText('New Workspace')).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Test Workspace' })).toBeVisible();
  });

  test('should handle empty workspace state gracefully', async ({ page }) => {
    await page.route('**/api/v1/workspaces', async (route) => {
      await route.fulfill({ json: [] });
    });
    await login(page);

    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Select Workspace');
  });

  test('should show an error for invalid credentials', async ({ page }) => {
    await page.unroute('**/api/v1/auth/login');
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({ status: 401, body: 'Invalid credentials' });
    });
    await page.getByLabel('Username').fill('wrong');
    await page.getByLabel('Password').fill('wrong');
    await page.getByRole('button', { name: 'Sign in' }).click();

    await expect(page.getByRole('alert')).toHaveText('Invalid username or password');
  });

  test('should navigate nested folders and create a folder', async ({ page }) => {
    await login(page);

    await page.getByRole('button', { name: 'Product Boards', exact: true }).first().click();
    await expect(page.getByRole('navigation', { name: 'Folder breadcrumbs' })).toContainText('Product Boards');
    await expect(page.getByRole('button', { name: 'Research', exact: true }).last()).toBeVisible();

    await page.getByRole('button', { name: 'All files' }).last().click();
    await page.getByRole('button', { name: 'New folder' }).click();
    await page.getByLabel('Folder name').fill('Planning');
    await page.getByRole('button', { name: 'Create', exact: true }).click();

    await expect(
      page.getByRole('region', { name: 'Folder contents' }).getByRole('button', { name: 'Planning', exact: true }),
    ).toBeVisible();
  });

  test('should create, edit, rename, star, and trash a scene', async ({ page }) => {
    let contentSaveCount = 0;
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') consoleErrors.push(message.text());
    });
    page.on('request', (request) => {
      if (request.method() === 'PATCH' && request.url().includes('/api/v1/scenes/')) {
        const body = request.postDataJSON() as Record<string, unknown>;
        if (body.content) contentSaveCount += 1;
      }
    });

    await login(page);
    await page.getByRole('button', { name: 'New scene' }).click();
    await expect(page.getByRole('heading', { name: 'Untitled scene' })).toBeVisible();
    await expect(page.getByRole('region', { name: 'Excalidraw canvas' })).toBeVisible();
    await page.waitForTimeout(300);
    await expect(page.getByRole('status')).toHaveText('Saved');
    if (canvasScreenshotPath) await page.screenshot({ path: canvasScreenshotPath, fullPage: false });

    const canvas = page.locator('canvas').first();
    await expect(canvas).toBeVisible();
    const box = await canvas.boundingBox();
    if (!box) throw new Error('Excalidraw canvas has no bounding box');
    await page.mouse.move(box.x + 250, box.y + 180);
    await page.mouse.down();
    await page.mouse.move(box.x + 360, box.y + 250, { steps: 5 });
    await page.mouse.up();
    await expect(page.getByRole('status')).toContainText(/Waiting to save|Saving|Saved/);
    await page.waitForTimeout(900);
    expect(contentSaveCount).toBe(1);

    await page.getByRole('button', { name: 'Back to dashboard' }).click();
    await expect(page.getByRole('region', { name: 'Scenes' })).toContainText('Untitled scene');

    await page.getByRole('button', { name: 'Actions for Untitled scene' }).click();
    page.once('dialog', async (dialog) => dialog.accept('Roadmap sketch'));
    await page.getByRole('menuitem', { name: 'Rename' }).click();
    await expect(page.getByRole('region', { name: 'Scenes' })).toContainText('Roadmap sketch');

    await page.getByRole('button', { name: 'Actions for Roadmap sketch' }).click();
    await page.getByRole('menuitem', { name: 'Star', exact: true }).click();
    await expect(page.getByLabel('Starred')).toBeVisible();

    await page.getByRole('button', { name: 'Actions for Roadmap sketch' }).click();
    await page.getByRole('menuitem', { name: 'Move to trash' }).click();
    await expect(page.getByRole('region', { name: 'Scenes' })).toContainText('No scenes in this location.');
    expect(consoleErrors).toEqual([]);
  });

  test('should render the whiteboard without page overflow on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await login(page);
    await page.getByRole('button', { name: 'New scene' }).click();

    await expect(page.getByRole('region', { name: 'Excalidraw canvas' })).toBeVisible();
    await expect(page.getByRole('status')).toHaveText('Saved');
    const pageWidth = await page.evaluate(() => ({ scrollWidth: document.documentElement.scrollWidth, innerWidth }));
    expect(pageWidth.scrollWidth).toBeLessThanOrEqual(pageWidth.innerWidth);
    if (mobileScreenshotPath) await page.screenshot({ path: mobileScreenshotPath, fullPage: false });
  });
});
