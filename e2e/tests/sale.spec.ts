import { expect, test, unique } from '../fixtures.ts';

// Spec 06: asking price, listing text, selling into the archive, proceeds and taking an item back into stock.
test('sale: price, listing, sold into the archive, proceeds, back into stock', async ({ page, loggedIn, api }) => {
	const id = unique();
	const name = `Stehlampe ${id}`;
	const a = await api.apartment(`Wohnung ${id}`);
	const room = await api.room(a.id, 'Wohnzimmer');
	const created = await api.item(name, room.id);

	// Set the disposition and a price with a comma.
	await page.goto(`/items/${created.id}`);
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	await page.getByRole('radio', { name: 'Verkaufen', exact: true }).check();
	await page.getByLabel('Angebotspreis in Euro (optional)').fill('45,50');
	await page.getByRole('button', { name: 'Speichern' }).click();
	await expect(page.getByText('Angebotspreis: 45,50 €')).toBeVisible();

	// The listing ends with the warranty disclaimer.
	const text = page.getByLabel('Text', { exact: true });
	await expect(text).toHaveValue(/Preis: 45,50 €/);
	await expect(text).toHaveValue(/Ausschluss jeglicher Gewährleistung, Garantie und Rücknahme\.$/);
	await expect(page.getByLabel('Titel', { exact: true })).toHaveValue(name);

	// Sold for less than asked; the dialog is preset with the asking price.
	await page.getByRole('region', { name: 'Inserat' }).getByRole('button', { name: 'Verkauft' }).click();
	const dialog = page.getByRole('dialog', { name: `„${name}“ verkauft` });
	await expect(dialog.getByLabel('Verkaufserlös in Euro')).toHaveValue('45,50');
	await dialog.getByLabel('Verkaufserlös in Euro').fill('40');
	await dialog.getByRole('button', { name: 'Ins Archiv' }).click();
	await expect(dialog).toBeHidden();
	await expect(page.getByText('Verkauft für 40 €')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Bearbeiten' })).toHaveCount(0);

	// Gone from the start page, found through the archive hint.
	await page.goto('/');
	await page.getByRole('searchbox').first().fill(name);
	await expect(page.getByText('Keine Items gefunden.')).toBeVisible();
	await page.getByRole('link', { name: 'Im Archiv suchen' }).click();
	await expect(page.getByRole('heading', { name: 'Archiv', level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: new RegExp(name) })).toContainText('Verkauft · 40 €');

	// The proceeds of this month contain it.
	await page.goto('/erloese');
	await expect(page.getByRole('link', { name: new RegExp(name) })).toBeVisible();

	// Back into stock: with its disposition and price, without placement.
	await page.goto(`/items/${created.id}`);
	page.once('dialog', (d) => d.accept());
	await page.getByRole('button', { name: 'Zurück in den Bestand' }).click();
	await expect(page.getByText('Angebotspreis: 45,50 €')).toBeVisible();
	await expect(page.getByText('Noch nicht platziert.')).toBeVisible();
});
