import { api, type Nutzer } from './api.ts';
import { darstellung, designWaehlen, istDesign, type Design } from './design.svelte.ts';

/** Der angemeldete Nutzer; null, solange niemand angemeldet ist. */
export const sitzung = $state<{ nutzer: Nutzer | null; geladen: boolean }>({
	nutzer: null,
	geladen: false
});

// Hat der Nutzer am Konto ein Design gewählt, gilt das; sonst bleibt das dieses Geräts.
function uebernehmen(nutzer: Nutzer) {
	sitzung.nutzer = nutzer;
	if (istDesign(nutzer.design)) designWaehlen(nutzer.design);
}

export async function ladeSitzung() {
	try {
		uebernehmen(await api<Nutzer>('GET', '/api/ich', undefined, { ohneUmleitung: true }));
	} catch {
		sitzung.nutzer = null;
	} finally {
		sitzung.geladen = true;
	}
}

export async function anmelden(email: string, passwort: string) {
	uebernehmen(await api<Nutzer>('POST', '/api/anmeldung', { email, passwort }, { ohneUmleitung: true }));
}

/** Nimmt eine Einladung an; danach ist die neue Person angemeldet. */
export async function einladungAnnehmen(token: string, email: string, passwort: string) {
	uebernehmen(
		await api<Nutzer>(
			'POST',
			'/api/einladungen/annehmen',
			{ token, email, passwort },
			{ ohneUmleitung: true }
		)
	);
}

/** Das zuletzt aktive Design bleibt lokal stehen, damit die Anmeldeseite so aussieht wie zuletzt. */
export async function abmelden() {
	await api('POST', '/api/abmeldung', undefined, { ohneUmleitung: true });
	sitzung.nutzer = null;
}

// Zählt die Wahlen mit, damit bei schnellem Umschalten nur die letzte zählt.
let letzteWahl = 0;

/**
 * Wendet das Design sofort an und speichert es am Konto. Schlägt das Speichern fehl, gilt wieder das vorherige
 * Design und der Fehler geht an den Aufrufer – außer es wurde inzwischen schon wieder umgeschaltet.
 */
export async function designSpeichern(d: Design) {
	const vorher = darstellung.design;
	const nr = ++letzteWahl;
	designWaehlen(d);
	try {
		const nutzer = await api<Nutzer>('PUT', '/api/ich/design', { design: d });
		if (nr === letzteWahl) sitzung.nutzer = nutzer;
	} catch (e) {
		if (nr !== letzteWahl) return;
		designWaehlen(vorher);
		throw e;
	}
}
