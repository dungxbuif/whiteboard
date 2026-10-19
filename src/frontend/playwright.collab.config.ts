import { defineConfig, devices } from '@playwright/test';
export default defineConfig({
  testDir: './e2e/real-stack', testMatch: '**/*.spec.ts',
  fullyParallel: false, workers: 1, retries: 0, timeout: 120_000,
  forbidOnly: !!process.env.CI,
  reporter: [['list'], ['json', { outputFile: 'test-results/real-stack-results.json' }]],
  use: { ...devices['Desktop Chrome'], actionTimeout: 10000, trace: 'retain-on-failure' },
});
