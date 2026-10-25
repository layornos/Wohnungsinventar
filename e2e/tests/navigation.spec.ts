import { expect, isNarrow, test } from '../fixtures.ts';
import { openMenu } from './helpers.ts';

test.beforeEach(async ({ loggedIn }) => {});

test('side menu on a sub page', async ({ page }) => {
	await page.goto('/wohnungen');
	await expect(page.getByRole('heading', { name: 'Wohnungen', level: 1 })).toBeVisible();

	if (isNarrow(page)) {
		await expect(page.getByRole('navigation', { name: 'Menü' })).toBeHidden();
	} else {
		// No ☰ button and no bottom bar next to the sidebar.
		await expect(page.getByRole('button', { name: 'Menü', exact: true })).toBeHidden();
	}

	const menu = await openMenu(page);
	await expect(menu.getByRole('link', { name: 'Wohnungen' })).toHaveAttribute('aria-current', 'page');
	await menu.getByRole('link', { name: 'Umzug' }).click();

	await expect(page).toHaveURL(/\/umzug$/);
	await expect(page.getByRole('heading', { name: 'Umzug', level: 1 })).toBeVisible();
	if (isNarrow(page)) await expect(page.getByRole('dialog', { name: 'Menü' })).toBeHidden();
});

test('theme switch applies at once and persists', async ({ page }) => {
	await page.goto('/einstellungen');
	const html = page.locator('html');

	const saved = page.waitForResponse((r) => r.url().endsWith('/api/me/theme') && r.request().method() === 'PUT');
	await page.getByRole('radio', { name: /Neon/ }).check();
	await expect(html).toHaveAttribute('data-theme', 'neon');
	expect((await saved).ok()).toBe(true);

	await page.reload();
	await expect(html).toHaveAttribute('data-theme', 'neon');
	await expect(page.getByRole('radio', { name: /Neon/ })).toBeChecked();

	// Back to the default so other tests of this worker see the usual theme.
	const reset = page.waitForResponse((r) => r.url().endsWith('/api/me/theme') && r.request().method() === 'PUT');
	await page.getByRole('radio', { name: /Minimal/ }).check();
	await expect(html).toHaveAttribute('data-theme', 'minimal');
	expect((await reset).ok()).toBe(true);
});
