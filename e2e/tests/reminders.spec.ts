import { expect, test, unique } from '../fixtures.ts';
import { openMenu } from './helpers.ts';

// Spec 07 without time travel (the test server runs on the real clock): keeping an item plans its keep review in
// three years, and the reminder mail can be switched off in the settings.
test('reminders: next keep review on the item page, mail setting', async ({ page, loggedIn, api }) => {
	const created = await api.item(`Regal ${unique()}`, (await api.room((await api.apartment(`Wohnung ${unique()}`)).id, 'Flur')).id);

	await page.goto(`/items/${created.id}`);
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	await page.getByRole('radio', { name: 'Behalten', exact: true }).check();
	await page.getByRole('button', { name: 'Speichern' }).click();
	const inThreeYears = new Intl.DateTimeFormat('de-DE', { timeZone: 'Europe/Berlin', day: '2-digit', month: '2-digit', year: 'numeric' })
		.format(new Date(new Date().setFullYear(new Date().getFullYear() + 3)));
	await expect(page.getByText(`Nächste Behalte-Prüfung am ${inThreeYears}`)).toBeVisible();

	const menu = await openMenu(page);
	await menu.getByRole('link', { name: 'Erinnerungen' }).click();
	await expect(page.getByText('Keine offenen Erinnerungen.')).toBeVisible();

	await page.goto('/einstellungen');
	const mail = page.getByRole('checkbox', { name: 'Erinnerungen per Mail' });
	await expect(mail).toBeChecked();
	const saved = page.waitForResponse((r) => r.url().endsWith('/api/me/reminder-mail'));
	await mail.uncheck();
	expect((await saved).ok()).toBe(true);
	await page.reload();
	await expect(page.getByRole('checkbox', { name: 'Erinnerungen per Mail' })).not.toBeChecked();
	await page.getByRole('checkbox', { name: 'Erinnerungen per Mail' }).check();
});
