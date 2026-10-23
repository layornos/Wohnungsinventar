import { goto } from '$app/navigation';

/** Fachlicher Fehler der API, z. B. code "anmeldung_fehlgeschlagen". */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		readonly code: string,
		meldung: string
	) {
		super(meldung);
	}
}

export const ANMELDUNG = '/anmeldung';

export function zurAnmeldung() {
	return goto(ANMELDUNG, { replaceState: true });
}

export type Nutzer = { personId: number; name: string; email: string };

export type Person = { id: number; name: string; nutzer: boolean };

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

export type Ort = { id: number; name: string };

/** Wo etwas liegt: Wohnung › Raum › Lagerorte von außen nach innen. */
export type Pfad = { wohnung: Ort; raum: Ort; lagerorte: Ort[] };

export function pfadText(p: Pfad): string {
	return [p.wohnung, p.raum, ...p.lagerorte].map((o) => o.name).join(' › ');
}

/** Ein Raum oder Lagerort als Platz für ein Item oder einen Lagerort. */
export type Platz = { art: 'raum' | 'lagerort'; id: number };
export type PlatzEintrag = Platz & { pfad: string };

export type LagerortKnoten = Ort & { moebel: Ort | null; unter: LagerortKnoten[] };
export type Lagerort = Ort & { pfad: Pfad; moebel: Ort | null; unter: Ort[] };
export type Raum = Ort & { wohnung: Ort; lagerorte: LagerortKnoten[] };
export type Wohnung = Ort & { raeume: Ort[] };

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
	moebelLagerorte: Ort[];
};

export type ItemListenEintrag = {
	id: number;
	name: string;
	verbleib: Verbleib | null;
	vorschauUrl: string;
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
		throw new ApiError(res.status, data.code ?? 'unbekannt', data.meldung ?? res.statusText);
	}
	return data as T;
}
