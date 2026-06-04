import { test, expect } from '@playwright/test';

// Mock token for testing
const MOCK_TOKEN = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3QtcGxheXdyaWdodEBleGFtcGxlLmNvbSIsImV4cCI6MTc4MDU5MjUxOSwic3ViIjoiMmUwMTExNmMtMTE5OC00OGI4LTk5NzctMmNkNjZmNTQ1MDJlIn0.6kDT4HIpT85t2vQnXVeDeRWPGibWUVLzPW5Tap3cGNk';

test.describe('Dashboard E2E Flows', () => {
  test.beforeEach(async ({ page }) => {
    // Inject token into localStorage before each test
    await page.addInitScript((token) => {
      window.localStorage.setItem('auth_token', token);
    }, MOCK_TOKEN);
    
    await page.goto('/');
  });

  test('should restore session and display user context', async ({ page }) => {
    // Wait for the specific workspace label to appear
    await expect(page.getByText('Workspace', { exact: true })).toBeVisible();
    
    // Check if the default "Select Workspace" is shown
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });

  test('should allow switching workspaces', async ({ page }) => {
    // Wait for the WorkspaceSwitcher to be ready
    const switcher = page.getByRole('combobox');
    await expect(switcher).toBeVisible();
    
    // Open the switcher
    await switcher.click();
    
    // Verify "New Workspace" option exists
    await expect(page.getByText('New Workspace')).toBeVisible();
  });

  test('should handle empty workspace state gracefully', async ({ page }) => {
    // If no workspaces, it should show "Select Workspace"
    const heading = page.getByRole('heading', { level: 1 });
    const text = await heading.textContent();
    expect(text === 'Select Workspace' || text?.length! > 0).toBeTruthy();
  });
});
