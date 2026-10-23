import { api, type Nutzer } from './api.ts';

/** Der angemeldete Nutzer; null, solange niemand angemeldet ist. */
export const sitzung = $state<{ nutzer: Nutzer | null; geladen: boolean }>({
	nutzer: null,
	geladen: false
});

export async function ladeSitzung() {
	try {
		sitzung.nutzer = await api<Nutzer>('GET', '/api/ich', undefined, { ohneUmleitung: true });
	} catch {
		sitzung.nutzer = null;
	} finally {
		sitzung.geladen = true;
	}
}

export async function anmelden(email: string, passwort: string) {
	sitzung.nutzer = await api<Nutzer>('POST', '/api/anmeldung', { email, passwort }, { ohneUmleitung: true });
}

/** Nimmt eine Einladung an; danach ist die neue Person angemeldet. */
export async function einladungAnnehmen(token: string, email: string, passwort: string) {
	sitzung.nutzer = await api<Nutzer>(
		'POST',
		'/api/einladungen/annehmen',
		{ token, email, passwort },
		{ ohneUmleitung: true }
	);
}

export async function abmelden() {
	await api('POST', '/api/abmeldung', undefined, { ohneUmleitung: true });
	sitzung.nutzer = null;
}
