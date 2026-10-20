import { openScene } from './browser';
import { expect, test } from './fixtures';

for (const viewport of [
  { width: 390, height: 844 },
  { width: 1440, height: 900 },
]) {
  test(`canvas controls float without reserving document space at ${viewport.width}x${viewport.height}`, async ({ page, stack }, info) => {
    await page.setViewportSize(viewport);
    const scene = await stack.scene(`Canvas controls ${viewport.width}`);
    await openScene(page, stack, scene);

    const root = page.getByTestId('scene-canvas-root');
    await expect(root).toBeVisible();
    const bounds = await root.boundingBox();
    expect(bounds).not.toBeNull();
    expect(Math.abs(bounds!.x)).toBeLessThanOrEqual(1);
    expect(Math.abs(bounds!.y)).toBeLessThanOrEqual(1);
    expect(Math.abs(bounds!.width - viewport.width)).toBeLessThanOrEqual(1);
    expect(Math.abs(bounds!.height - viewport.height)).toBeLessThanOrEqual(1);
    await expect(root.locator('header')).toHaveCount(0);

    await expect(page.getByRole('button', { name: /People \(1\)/ })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Join voice' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Present', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'More scene actions' })).toBeVisible();
    await expect(page.getByText('Voice 0/10')).toHaveCount(0);
    await expect(page.getByRole('status')).toHaveText('Saved');

    const overlays = await page.locator('[data-testid="scene-controls"]').evaluate((element) => getComputedStyle(element).pointerEvents);
    expect(overlays).toBe('none');
    await page.screenshot({ path: info.outputPath(`canvas-controls-${viewport.width}x${viewport.height}.png`) });
  });
}
