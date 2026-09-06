import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  expect: { timeout: 15000 },
  reporter: 'list',
  use: {
    baseURL: process.env.APP_URL || 'http://localhost:3000',
    channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome',
    headless: true,
    viewport: { width: 1440, height: 1100 },
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
});
