import { test as base, expect } from '@playwright/test';
import { RealStack } from './harness';
export const test = base.extend<{}, { stack: RealStack }>({
  stack: [async ({}, use) => {
    const stack = new RealStack();
    try { await stack.start(); await use(stack); }
    catch (e) { console.error(stack.safeLogs()); throw e; }
    finally { await stack.close(); }
  }, { scope: 'test', timeout: 180_000 }],
});
export { expect };
