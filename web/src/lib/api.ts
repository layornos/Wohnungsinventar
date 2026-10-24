import { goto } from '$app/navigation';
import type { Design } from './design.svelte.ts';

/** Fachlicher Fehler der API, z. B. code "anmeldung_fehlgeschlagen". */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		readonly code: string,
		meldung: string,
		/** Die vollständige Fehlerantwort, z. B. mit der Liste der Items ohne Zielraum. */
		readonly daten: Record<string, unknown> = {}
	) {
		super(meldung);
	}
}

export const ANMELDUNG = '/anmeldung';

export function zurAnmeldung() {
	return goto(ANMELDUNG, { replaceState: true });
}

/** design ist null, solange der Nutzer noch nie eines gewählt hat. */
export type Nutzer = { personId: number; name: string; email: string; design: Design | null };

export type Person = { id: number; name: string; nutzer: boolean };

/** Offene Einladung; personId ist null, wenn die Person erst beim Annehmen angelegt wird. */
export type Einladung = {
	id: number;
	personId: number | null;
	name: string;
	ersteller: string;
	erstelltAm: string;
	laeuftAbAm: string;
};

/** Öffentliche Seite für Einladungslinks; der Token steht im Fragment. */
export const EINLADUNG = '/einladung';

export function datumText(iso: string): string {
	return new Date(iso).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' });
}

export function datumKurz(iso: string): string {
	return new Date(iso).toLocaleDateString('de-DE');
}

/** null heißt: Verbleib noch offen. */
export type Verbleib = 'behalten' | 'verkaufen' | 'spaeter_verkaufen' | 'verschenken';

export const VERBLEIBE: { wert: Verbleib; name: string }[] = [
	{ wert: 'behalten', name: 'Behalten' },
	{ wert: 'verkaufen', name: 'Verkaufen' },
	{ wert: 'spaeter_verkaufen', name: 'Später verkaufen' },
	{ wert: 'verschenken', name: 'Verschenken' }
];

export function verbleibName(v: Verbleib | null): string {
	return VERBLEIBE.find((e) => e.wert === v)?.name ?? 'Offen';
}

export type Foto = { id: number; hauptfoto: boolean; url: string; vorschauUrl: string };

export type Benannt = { id: number; name: string };

/** Wo etwas liegt: Wohnung › Raum › Lagerorte von außen nach innen. */
export type Pfad = { wohnung: Benannt; raum: Benannt; lagerorte: Benannt[] };

export function pfadText(p: Pfad): string {
	return [p.wohnung, p.raum, ...p.lagerorte].map((o) => o.name).join(' › ');
}

/** Ein Raum oder Lagerort als Platz für ein Item oder einen Lagerort. */
export type Platz = { art: 'raum' | 'lagerort'; id: number };
export type PlatzEintrag = Platz & { pfad: string };

export type LagerortKnoten = Benannt & { moebel: Benannt | null; unter: LagerortKnoten[] };
/** unterbaum: IDs des Lagerorts und aller Unterlagerorte. */
export type Lagerort = Benannt & { pfad: Pfad; moebel: Benannt | null; unter: Benannt[]; unterbaum: number[] };
export type Raum = Benannt & { wohnung: Benannt; lagerorte: LagerortKnoten[] };
export type Wohnung = Benannt & { raeume: Benannt[] };

export type Item = {
	id: number;
	name: string;
	beschreibung: string;
	verbleib: Verbleib | null;
	erstelltAm: string;
	geaendertAm: string;
	besitzer: Person[];
	fotos: Foto[];
	platzierung: Pfad | null;
	moebelLagerorte: Benannt[];
	umzug: Umzugsangaben | null;
};

export type BoxKurz = { id: number; nummer: number; bezeichnung: string };
export type Box = BoxKurz & { anzahl: number };

/** Herkunftsraum, Zielraum und Box eines Items im laufenden Umzug. */
export type Umzugsangaben = { herkunftsraum: Benannt | null; zielraum: Benannt | null; box: BoxKurz | null };

export type Umzug = {
	id: number;
	alteWohnung: Benannt;
	neueWohnung: Benannt;
	laufend: boolean;
	erstelltAm: string;
	abgeschlossenAm: string | null;
	boxen: Box[];
};

export function boxName(b: BoxKurz): string {
	return b.bezeichnung ? `Box ${b.nummer} · ${b.bezeichnung}` : `Box ${b.nummer}`;
}

export type ItemListenEintrag = {
	id: number;
	name: string;
	verbleib: Verbleib | null;
	vorschauUrl: string;
	/** Name des Lagerorts oder, ohne Lagerort, des Raums; leer ohne Platzierung. */
	platz: string;
	umzug: Umzugsangaben | null;
};

/** Meldung für einen Fehler, ob von der API oder weil der Server nicht erreichbar ist. */
export function fehlerText(e: unknown): string {
	return e instanceof ApiError ? e.message : 'Der Server ist nicht erreichbar.';
}

/**
 * Ruft die API auf; FormData geht als multipart, alles andere als JSON.
 * Bei 401 wird zur Anmeldung umgeleitet, außer `ohneUmleitung` ist gesetzt.
 */
export async function api<T>(
	method: string,
	path: string,
	body?: unknown,
	opts: { ohneUmleitung?: boolean } = {}
): Promise<T> {
	const alsJSON = body !== undefined && !(body instanceof FormData);
	const res = await fetch(path, {
		method,
		headers: alsJSON ? { 'Content-Type': 'application/json' } : {},
		body: alsJSON ? JSON.stringify(body) : (body as FormData | undefined)
	});
	if (res.status === 204) return undefined as T;
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		if (res.status === 401 && !opts.ohneUmleitung) {
			await zurAnmeldung();
		}
		throw new ApiError(res.status, data.code ?? 'unbekannt', data.meldung ?? res.statusText, data);
	}
	return data as T;
}
