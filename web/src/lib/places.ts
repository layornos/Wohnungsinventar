import { api, type ItemListEntry, type Place, type PlaceEntry } from './api.ts';

export async function loadPlaces(): Promise<PlaceEntry[]> {
	return (await api<{ places: PlaceEntry[] }>('GET', '/api/places')).places;
}

/** Body for setting a placement; null removes it. */
export function placementBody(p: Place | null) {
	if (!p) return {};
	return p.kind === 'room' ? { roomId: p.id } : { storageLocationId: p.id };
}

/** Body for creating or moving a storage location into a room or storage location. */
export function parentBody(p: Place) {
	return p.kind === 'room' ? { roomId: p.id } : { parentId: p.id };
}

/** Asks before deleting and names the items that will then be left without a storage location. */
export async function confirmRemoval(preview: string, question: string): Promise<boolean> {
	const { items } = await api<{ items: ItemListEntry[] }>('GET', preview);
	if (items.length === 0) return confirm(question);
	const names = items.map((i) => `• ${i.name}`).join('\n');
	return confirm(`${question}\n\nDiese Items stehen danach ohne Lagerort im Raum:\n${names}`);
}

const LAST_PLACE = 'inventar.lastPlace';

/** The place last chosen when capturing, remembered only in this browser. */
export function lastPlace(): Place | null {
	try {
		const p = JSON.parse(localStorage.getItem(LAST_PLACE) ?? 'null');
		return p && (p.kind === 'room' || p.kind === 'storageLocation') && typeof p.id === 'number' ? p : null;
	} catch {
		return null;
	}
}

export function rememberLastPlace(p: Place | null) {
	try {
		if (p) localStorage.setItem(LAST_PLACE, JSON.stringify(p));
		else localStorage.removeItem(LAST_PLACE);
	} catch {
		// Without storage nothing is preselected.
	}
}
