import { expect, test, USER } from '../fixtures.ts';
import { openMenu } from './helpers.ts';

test('login with e-mail and password', async ({ page, guard }) => {
	// Without a session the app asks the server who is logged in and gets 401.
	guard.allow(401, '/api/me');

	await page.goto('/');
	await expect(page).toHaveURL(/\/anmeldung$/);
	await page.getByLabel('E-Mail').fill(USER.email);
	await page.getByLabel('Passwort').fill(USER.password);
	await page.getByRole('button', { name: 'Anmelden' }).click();

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { name: 'Inventar' })).toBeVisible();
});

test('wrong password shows an error', async ({ page, guard }) => {
	guard.allow(401, '/api/me');
	guard.allow(401, '/api/login');

	await page.goto('/anmeldung');
	await page.getByLabel('E-Mail').fill(USER.email);
	await page.getByLabel('Passwort').fill('falsches-passwort');
	await page.getByRole('button', { name: 'Anmelden' }).click();

	await expect(page.getByRole('alert')).toHaveText('E-Mail oder Passwort falsch');
	await expect(page).toHaveURL(/\/anmeldung$/);
});

test('logout from the side menu', async ({ page, loggedIn, guard }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'Inventar' })).toBeVisible();

	const menu = await openMenu(page);
	await menu.getByRole('button', { name: 'Abmelden' }).click();
	await expect(page).toHaveURL(/\/anmeldung$/);
	await expect(page.getByRole('heading', { name: 'Anmelden' })).toBeVisible();

	// The session is gone on the server, too.
	guard.allow(401, '/api/me');
	await page.goto('/');
	await expect(page).toHaveURL(/\/anmeldung$/);
});
