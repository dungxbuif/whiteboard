import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: [
    {
      command: 'npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'cd ../backend && GOOGLE_CLIENT_ID="test-google-client-id" GOOGLE_CLIENT_SECRET="test-google-client-secret" DATABASE_URL="postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable" JWT_SECRET="test-jwt-secret-minimum-32-bytes-long" ./server',
      url: 'http://localhost:8080/healthz',
      reuseExistingServer: !process.env.CI,
    }
  ],
});
