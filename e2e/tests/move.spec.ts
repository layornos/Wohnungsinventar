import { expect, test, unique } from '../fixtures.ts';

// Spec 09: create a move, a box with a target room, pack an item, print the label and unpack the whole box into a
// storage location of the new apartment.
test('move: box with target room, pack, label, unpack all', async ({ page, loggedIn, api }) => {
	// Only one move can be active; one left over from an earlier run on this worker's server is completed first.
	const active = await api.call<{ move: unknown }>('GET', '/api/move');
	if (active.move) await api.call('POST', '/api/move/completion', { confirmed: true });

	const id = unique();
	const oldName = `Altbau ${id}`;
	const newName = `Neubau ${id}`;
	const item = `Wasserkocher ${id}`;

	const oldApartment = await api.apartment(oldName);
	const study = await api.room(oldApartment.id, 'Arbeitszimmer');
	const newApartment = await api.apartment(newName);
	const kitchen = await api.room(newApartment.id, 'Küche');
	await api.storageLocation('Oberschrank', { roomId: kitchen.id });
	const created = await api.item(item, study.id);

	// Create the move.
	await page.goto('/umzug');
	await page.getByLabel('Von').selectOption({ label: oldName });
	await page.getByLabel('Nach').selectOption({ label: newName });
	await page.getByRole('button', { name: 'Umzug anlegen' }).click();
	await expect(page.getByText(`${oldName} → ${newName}`)).toBeVisible();

	// New box with a target room from the packing mode; it becomes the active box.
	await page.getByRole('link', { name: 'Packen' }).click();
	await expect(page.getByRole('heading', { name: 'Packen', level: 1 })).toBeVisible();
	await page.getByRole('button', { name: 'Neue Box' }).click();
	const dialog = page.getByRole('dialog', { name: 'Neue Box' });
	await dialog.getByLabel('Bezeichnung (optional)').fill('Küchenkram');
	await dialog.getByLabel('Zielraum').selectOption({ label: 'Küche' });
	await dialog.getByRole('button', { name: 'Anlegen' }).click();
	await expect(dialog).toBeHidden();
	await expect(page.getByLabel('Aktive Box').locator('option:checked')).toHaveText('Box 1 · Küchenkram → Küche (0)');

	// Pack the item by tapping it; it then has the box's target room.
	const row = page.getByRole('button', { name: new RegExp(item) });
	await row.click();
	await expect(row).toContainText('Box 1 · Küchenkram · Küche');
	await expect(page.getByLabel('Aktive Box').locator('option:checked')).toHaveText('Box 1 · Küchenkram → Küche (1)');

	// Box page and label with QR code.
	await page.getByRole('link', { name: '← Umzug' }).click();
	await page.getByRole('link', { name: 'Box 1 · Küchenkram → Küche' }).click();
	await expect(page).toHaveURL(/\/boxen\/\d+$/);
	await page.getByRole('link', { name: 'Etikett' }).click();
	await expect(page.getByRole('heading', { name: 'Etikett' })).toBeVisible();
	const qr = page.getByRole('img', { name: 'QR-Code zum Auspacken von Box 1' });
	await expect(qr).toBeVisible();
	await expect.poll(() => qr.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
	await expect(page.getByText('→ Küche')).toBeVisible();

	// Unpack mode: the box's target room is preselected; choose the cupboard and put everything there.
	await page.getByRole('link', { name: '← Box' }).click();
	await page.getByRole('link', { name: 'Auspacken' }).click();
	await expect(page).toHaveURL(/\/boxen\/\d+\/auspacken$/);
	await expect(page.getByRole('button', { name: new RegExp(item) })).toBeVisible();
	await page.getByRole('button', { name: `${newName} › Küche`, exact: true }).click();
	await page.getByLabel('Einräumen in suchen').fill('Oberschrank');
	await page.getByRole('button', { name: `${newName} › Küche › Oberschrank`, exact: true }).click();
	await expect(page.getByRole('button', { name: `${newName} › Küche › Oberschrank`, expanded: false })).toBeVisible();

	const unpacked = page.waitForResponse((r) => /\/api\/boxes\/\d+\/unpack$/.test(r.url()));
	await page.getByRole('button', { name: 'Alle hier einräumen' }).click();
	expect((await unpacked).ok()).toBe(true);
	await expect(page.getByText('Box ist leer.')).toBeVisible();
	await expect(page.getByRole('status').filter({ hasText: 'eingeräumt' })).toHaveText('1 Item eingeräumt.');

	// The item has its final place right away and is no longer in the box.
	await page.goto(`/items/${created.id}`);
	await expect(page.getByRole('heading', { name: item, level: 1 })).toBeVisible();
	await expect(page.getByRole('navigation', { name: 'Platz' })).toHaveText(`${newName} › Küche › Oberschrank`);
	const detail = await api.call('GET', `/api/items/${created.id}`);
	expect(detail.placement.room.name).toBe('Küche');
	expect(detail.placement.storageLocations.map((s: { name: string }) => s.name)).toEqual(['Oberschrank']);
	expect(detail.move?.box ?? null).toBeNull();
});
