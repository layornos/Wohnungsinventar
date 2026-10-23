import { api, type ItemListenEintrag, type Platz, type PlatzEintrag } from './api.ts';

export async function ladePlaetze(): Promise<PlatzEintrag[]> {
	return (await api<{ plaetze: PlatzEintrag[] }>('GET', '/api/plaetze')).plaetze;
}

/** Body für das Setzen einer Platzierung; null entfernt sie. */
export function platzBody(p: Platz | null) {
	if (!p) return {};
	return p.art === 'raum' ? { raumId: p.id } : { lagerortId: p.id };
}

/** Body für das Anlegen oder Verschieben eines Lagerorts in einen Raum oder Lagerort. */
export function elternBody(p: Platz) {
	return p.art === 'raum' ? { raumId: p.id } : { elternId: p.id };
}

/** Fragt vor dem Löschen nach und nennt die Items, die dann ohne Lagerort dastehen. */
export async function wegfallBestaetigen(vorschau: string, frage: string): Promise<boolean> {
	const { items } = await api<{ items: ItemListenEintrag[] }>('GET', vorschau);
	if (items.length === 0) return confirm(frage);
	const namen = items.map((i) => `• ${i.name}`).join('\n');
	return confirm(`${frage}\n\nDiese Items stehen danach ohne Lagerort im Raum:\n${namen}`);
}

const LETZTER_PLATZ = 'inventar.letzterPlatz';

/** Der zuletzt beim Erfassen gewählte Platz, nur in diesem Browser gemerkt. */
export function letzterPlatz(): Platz | null {
	try {
		const p = JSON.parse(localStorage.getItem(LETZTER_PLATZ) ?? 'null');
		return p && (p.art === 'raum' || p.art === 'lagerort') && typeof p.id === 'number' ? p : null;
	} catch {
		return null;
	}
}

export function letztenPlatzMerken(p: Platz | null) {
	try {
		if (p) localStorage.setItem(LETZTER_PLATZ, JSON.stringify(p));
		else localStorage.removeItem(LETZTER_PLATZ);
	} catch {
		// Ohne Speicher wird eben nichts vorausgewählt.
	}
}
