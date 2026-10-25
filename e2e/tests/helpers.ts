import { expect, type Locator, type Page } from '@playwright/test';
import { isNarrow } from '../fixtures.ts';

/** Opens the side menu: on the phone via ☰ in the bottom bar, on the desktop it is the sidebar. */
export async function openMenu(page: Page): Promise<Locator> {
	if (isNarrow(page)) {
		await page.getByRole('button', { name: 'Menü', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Menü' });
		await expect(dialog).toBeVisible();
		return dialog;
	}
	const sidebar = page.getByRole('navigation', { name: 'Menü' });
	await expect(sidebar).toBeVisible();
	return sidebar;
}
