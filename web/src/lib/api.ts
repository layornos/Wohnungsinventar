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

/**
 * Ruft die API auf. Bei 401 wird zur Anmeldung umgeleitet, außer `ohneUmleitung` ist gesetzt.
 */
export async function api<T>(
	method: string,
	path: string,
	body?: unknown,
	opts: { ohneUmleitung?: boolean } = {}
): Promise<T> {
	const res = await fetch(path, {
		method,
		headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
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
