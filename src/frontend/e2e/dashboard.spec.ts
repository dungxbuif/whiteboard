import { test, expect } from '@playwright/test';

const screenshotPath = process.env.SCREENSHOT_PATH;
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

test.describe('Dashboard E2E Flows', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({ json: { token: 'playwright-token' } });
    });
    await page.route('**/api/v1/users/me', async (route) => {
      await route.fulfill({ json: mockUser });
    });
    await page.route('**/api/v1/workspaces', async (route) => {
      await route.fulfill({ json: mockWorkspaces });
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
});
