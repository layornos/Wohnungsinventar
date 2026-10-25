import { expect, PHOTO, test, unique } from '../fixtures.ts';

test('create an item with a photo and a new storage location from the place picker', async ({ page, loggedIn, api }) => {
	const id = unique();
	const apartment = `Wohnung ${id}`;
	const room = `Wohnzimmer ${id}`;
	const shelf = `Regal ${id}`;
	const name = `Lampe ${id}`;
	const a = await api.apartment(apartment);
	await api.room(a.id, room);

	await page.goto('/items/neu');
	await expect(page.getByRole('heading', { name: 'Erfassen' })).toBeVisible();

	const chooser = page.waitForEvent('filechooser');
	await page.getByRole('button', { name: 'Aus Galerie' }).click();
	await (await chooser).setFiles(PHOTO);
	await expect(page.getByRole('img', { name: 'Foto 1' })).toBeVisible();

	await page.getByLabel('Name', { exact: true }).fill(name);

	// The storage location doesn't exist yet: search for it, then create it in the room.
	await page.getByRole('button', { name: 'Kein Platz', expanded: false }).click();
	await page.getByLabel('Platz suchen').fill(shelf);
	await page.getByRole('button', { name: `„${shelf}“ anlegen in …` }).click();
	await page.getByLabel('Anlegen in suchen').fill(room);
	const created = page.waitForResponse((r) => r.url().endsWith('/api/storage-locations') && r.request().method() === 'POST');
	await page.getByRole('button', { name: `${apartment} › ${room}`, exact: true }).click();
	expect((await created).status()).toBe(201);
	await expect(page.getByRole('button', { name: `${apartment} › ${room} › ${shelf}`, expanded: false })).toBeVisible();

	await page.getByRole('button', { name: 'Speichern' }).click();
	const status = page.getByRole('status').filter({ hasText: 'Gespeichert' });
	await expect(status).toBeVisible();
	await status.getByRole('link', { name }).click();

	await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
	const path = page.getByRole('navigation', { name: 'Platz' });
	await expect(path).toHaveText(`${apartment} › ${room} › ${shelf}`);

	// The photo arrived on the server and loads.
	const photo = page.locator('main img').first();
	await expect(photo).toBeVisible();
	await expect.poll(() => photo.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
});
