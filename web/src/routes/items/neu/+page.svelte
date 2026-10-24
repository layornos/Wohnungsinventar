<script lang="ts">
	import { page } from '$app/state';
	import { api, fehlerText, type Item, type Person, type Platz, type PlatzEintrag, type Verbleib } from '#lib/api.ts';
	import BesitzerAuswahl from '#lib/BesitzerAuswahl.svelte';
	import FotoEingabe from '#lib/FotoEingabe.svelte';
	import { ladePlaetze, letzterPlatz, letztenPlatzMerken } from '#lib/plaetze.ts';
	import { ladePersonen } from '#lib/personen.ts';
	import PlatzAuswahl from '#lib/PlatzAuswahl.svelte';
	import { sitzung } from '#lib/sitzung.svelte.ts';
	import VerbleibAuswahl from '#lib/VerbleibAuswahl.svelte';

	type Gewaehlt = { datei: File; vorschau: string };

	let personen = $state<Person[]>([]);
	let fotos = $state<Gewaehlt[]>([]);
	let name = $state('');
	let beschreibung = $state('');
	let besitzer = $state<number[]>(sitzung.nutzer ? [sitzung.nutzer.personId] : []);
	let verbleib = $state<Verbleib | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);
	let gespeichert = $state<Item | null>(null);
	// Aus dem Pack-Modus kommen neue Items direkt in die aktive Box.
	const box = page.url.searchParams.get('box');
	let plaetze = $state<PlatzEintrag[]>([]);
	let platz = $state<Platz | null>(null);

	ladePersonen()
		.then((p) => (personen = p))
		.catch((e) => (fehler = fehlerText(e)));

	// Der zuletzt gewählte Platz ist vorausgewählt, damit sich ein Regal schnell durcharbeiten lässt.
	ladePlaetze()
		.then((p) => {
			plaetze = p;
			const letzter = letzterPlatz();
			platz = p.some((e) => e.art === letzter?.art && e.id === letzter?.id) ? letzter : null;
		})
		.catch((e) => (fehler = fehlerText(e)));

	function hinzufuegen(dateien: File[]) {
		fotos = [...fotos, ...dateien.map((datei) => ({ datei, vorschau: URL.createObjectURL(datei) }))];
	}

	function entfernen(i: number) {
		URL.revokeObjectURL(fotos[i].vorschau);
		fotos = fotos.filter((_, j) => j !== i);
	}

	async function speichern(event: SubmitEvent) {
		event.preventDefault();
		fehler = '';
		if (fotos.length === 0) {
			fehler = 'Bitte mindestens ein Foto aufnehmen.';
			return;
		}
		if (besitzer.length === 0) {
			fehler = 'Bitte mindestens einen Besitzer wählen.';
			return;
		}
		const form = new FormData();
		form.set('name', name);
		form.set('beschreibung', beschreibung);
		if (verbleib) form.set('verbleib', verbleib);
		if (platz) form.set(platz.art, String(platz.id));
		if (box) form.set('box', box);
		for (const id of besitzer) form.append('besitzer', String(id));
		for (const f of fotos) form.append('foto', f.datei);

		laeuft = true;
		try {
			gespeichert = await api<Item>('POST', '/api/items', form);
			letztenPlatzMerken(platz);
			// Bereit fürs nächste Item; Besitzer und Platz bleiben, weil oft ähnliche Items folgen.
			for (const f of fotos) URL.revokeObjectURL(f.vorschau);
			fotos = [];
			name = '';
			beschreibung = '';
			verbleib = null;
		} catch (e) {
			fehler = fehlerText(e);
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href={box ? '/umzug/packen?box=' + box : '/'}>{box ? '← Packen' : '← Liste'}</a>
		<h1>Erfassen</h1>
	</header>
	{#if box}<p class="hinweis">Neue Items kommen direkt in die aktive Box.</p>{/if}

	{#if gespeichert}
		<p class="hinweis" role="status">
			Gespeichert: <a href="/items/{gespeichert.id}">{gespeichert.name}</a>
		</p>
	{/if}

	<form onsubmit={speichern}>
		{#if fotos.length > 0}
			<ul class="fotoleiste">
				{#each fotos as f, i (f.vorschau)}
					<li>
						<img src={f.vorschau} alt="Foto {i + 1}" />
						<button type="button" class="sekundaer klein" onclick={() => entfernen(i)} aria-label="Foto {i + 1} entfernen">×</button>
					</li>
				{/each}
			</ul>
		{/if}
		<FotoEingabe gewaehlt={hinzufuegen} />

		<label>
			Name
			<input bind:value={name} required autocomplete="off" />
		</label>
		<label>
			Beschreibung (optional)
			<textarea bind:value={beschreibung} rows="2"></textarea>
		</label>
		<BesitzerAuswahl {personen} bind:ausgewaehlt={besitzer} />
		<PlatzAuswahl bind:plaetze bind:wert={platz} anlegen />
		<VerbleibAuswahl bind:wert={verbleib} />

		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
		<button type="submit" disabled={laeuft}>{laeuft ? 'Speichern …' : 'Speichern'}</button>
	</form>
</main>
