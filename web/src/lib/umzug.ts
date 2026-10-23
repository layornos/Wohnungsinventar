import { api, type Benannt, type Umzug, type Wohnung } from './api.ts';

export async function ladeUmzug(): Promise<Umzug | null> {
	return (await api<{ umzug: Umzug | null }>('GET', '/api/umzug')).umzug;
}

export async function ladeWohnungen(): Promise<Wohnung[]> {
	return (await api<{ wohnungen: Wohnung[] }>('GET', '/api/wohnungen')).wohnungen;
}

/** Die Räume der alten und der neuen Wohnung eines Umzugs. */
export async function ladeUmzugsraeume(u: Umzug): Promise<{ alt: Benannt[]; neu: Benannt[] }> {
	const wohnungen = await ladeWohnungen();
	const raeume = (id: number) => wohnungen.find((w) => w.id === id)?.raeume ?? [];
	return { alt: raeume(u.alteWohnung.id), neu: raeume(u.neueWohnung.id) };
}

/** Body für Umzugsangaben: nur genannte Felder ändern sich, null setzt zurück. */
export type AngabenAenderung = { herkunftsraumId?: number | null; zielraumId?: number | null; boxId?: number | null };
