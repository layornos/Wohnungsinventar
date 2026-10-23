<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, fehlerText, verbleibName, type Item, type Person, type Platz, type PlatzEintrag, type Verbleib } from '#lib/api.ts';
	import FotoEingabe from '#lib/FotoEingabe.svelte';
	import { ladePlaetze, platzBody, wegfallBestaetigen } from '#lib/orte.ts';
	import PfadAnzeige from '#lib/PfadAnzeige.svelte';
	import PlatzAuswahl from '#lib/PlatzAuswahl.svelte';
	import { ladePersonen } from '#lib/personen.ts';
	import VerbleibAuswahl from '#lib/VerbleibAuswahl.svelte';

	const pfad = $derived(`/api/items/${page.params.id}`);

	let item = $state<Item | null>(null);
	let personen = $state<Person[]>([]);
	let gezeigt = $state<number | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);
	let bearbeiten = $state(false);
	let entwurf = $state({ name: '', beschreibung: '', verbleib: null as Verbleib | null });

	const foto = $derived(item?.fotos.find((f) => f.id === gezeigt) ?? item?.fotos.find((f) => f.hauptfoto));
	const weiterePersonen = $derived(personen.filter((p) => !item?.besitzer.some((b) => b.id === p.id)));

	$effect(() => {
		const p = pfad;
		item = null;
		fehler = '';
		api<Item>('GET', p)
			.then((i) => (item = i))
			.catch((e) => (fehler = fehlerText(e)));
	});

	ladePersonen()
		.then((p) => (personen = p))
		.catch((e) => (fehler = fehlerText(e)));

	/** Führt eine Änderung aus, die das Item zurückliefert, und zeigt Fehler an. */
	async function aendern(method: string, adresse: string, body?: unknown) {
		fehler = '';
		laeuft = true;
		try {
			item = await api<Item>(method, adresse, body);
			return true;
		} catch (e) {
			fehler = fehlerText(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	function bearbeitenBeginnen() {
		if (!item) return;
		entwurf = { name: item.name, beschreibung: item.beschreibung, verbleib: item.verbleib };
		bearbeiten = true;
	}

	async function speichern(event: SubmitEvent) {
		event.preventDefault();
		if (await aendern('PUT', pfad, entwurf)) bearbeiten = false;
	}

	function fotosHochladen(dateien: File[]) {
		const form = new FormData();
		for (const d of dateien) form.append('foto', d);
		aendern('POST', `${pfad}/fotos`, form);
	}

	async function fotoLoeschen(id: number) {
		if (!confirm('Dieses Foto löschen?')) return;
		if (await aendern('DELETE', `/api/fotos/${id}`)) gezeigt = null;
	}

	let neuerBesitzer = $state('');
	async function besitzerHinzufuegen() {
		if (!neuerBesitzer) return;
		if (await aendern('PUT', `${pfad}/besitzer/${neuerBesitzer}`)) neuerBesitzer = '';
	}

	let plaetze = $state<PlatzEintrag[]>([]);
	let platzWaehlen = $state(false);
	let platz = $state<Platz | null>(null);

	async function platzAendernBeginnen() {
		try {
			plaetze = await ladePlaetze();
		} catch (e) {
			fehler = fehlerText(e);
			return;
		}
		const p = item?.platzierung;
		const innerster = p?.lagerorte.at(-1);
		platz = innerster ? { art: 'lagerort', id: innerster.id } : p ? { art: 'raum', id: p.raum.id } : null;
		platzWaehlen = true;
	}

	async function platzSpeichern() {
		if (await aendern('PUT', `${pfad}/platzierung`, platzBody(platz))) platzWaehlen = false;
	}

	async function itemLoeschen() {
		if (!item) return;
		fehler = '';
		try {
			const frage = `„${item.name}“ mit allen Fotos löschen?`;
			const bestaetigt =
				item.moebelLagerorte.length > 0
					? await wegfallBestaetigen(`${pfad}/wegfall`, `${frage} Seine Lagerorte (${item.moebelLagerorte.map((l) => l.name).join(', ')}) fallen weg.`)
					: confirm(frage);
			if (!bestaetigt) return;
			await api('DELETE', pfad);
			await goto('/', { replaceState: true });
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		{#if item && !bearbeiten}
			<button class="sekundaer" onclick={bearbeitenBeginnen}>Bearbeiten</button>
		{/if}
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if item}
		{#if foto}
			<figure class="grossfoto">
				<a href={foto.url} target="_blank" rel="noopener"><img src={foto.vorschauUrl} alt={item.name} /></a>
			</figure>
		{/if}
		<ul class="fotoleiste">
			{#each item.fotos as f (f.id)}
				<li>
					<button type="button" class="foto-wahl" class:aktiv={f.id === foto?.id} onclick={() => (gezeigt = f.id)}>
						<img src={f.vorschauUrl} alt={f.hauptfoto ? 'Hauptfoto' : 'Foto'} loading="lazy" />
					</button>
				</li>
			{/each}
		</ul>
		{#if foto}
			<div class="knopfzeile">
				{#if !foto.hauptfoto}
					<button class="sekundaer" disabled={laeuft} onclick={() => aendern('PUT', `${pfad}/hauptfoto`, { fotoId: foto.id })}>
						Als Hauptfoto
					</button>
				{:else}
					<span class="marke">Hauptfoto</span>
				{/if}
				<button class="sekundaer" disabled={laeuft} onclick={() => fotoLoeschen(foto.id)}>Foto löschen</button>
			</div>
		{/if}
		<FotoEingabe gewaehlt={fotosHochladen} deaktiviert={laeuft} />

		{#if bearbeiten}
			<form onsubmit={speichern}>
				<label>
					Name
					<input bind:value={entwurf.name} required />
				</label>
				<label>
					Beschreibung
					<textarea bind:value={entwurf.beschreibung} rows="3"></textarea>
				</label>
				<VerbleibAuswahl bind:wert={entwurf.verbleib} />
				<div class="knopfzeile">
					<button type="submit" disabled={laeuft}>Speichern</button>
					<button type="button" class="sekundaer" onclick={() => (bearbeiten = false)}>Abbrechen</button>
				</div>
			</form>
		{:else}
			<h1>{item.name}</h1>
			{#if item.beschreibung}
				<p class="beschreibung">{item.beschreibung}</p>
			{/if}
			<p>Verbleib: <span class="marke" class:offen={item.verbleib === null}>{verbleibName(item.verbleib)}</span></p>
		{/if}

		<section>
			<h2>Platz</h2>
			{#if platzWaehlen}
				<PlatzAuswahl {plaetze} bind:wert={platz} />
				<div class="knopfzeile">
					<button disabled={laeuft} onclick={platzSpeichern}>Speichern</button>
					<button class="sekundaer" onclick={() => (platzWaehlen = false)}>Abbrechen</button>
				</div>
			{:else}
				{#if item.platzierung}
					<PfadAnzeige pfad={item.platzierung} />
					{#if item.platzierung.lagerorte.length === 0}<p class="leise">Direkt im Raum, ohne Lagerort.</p>{/if}
				{:else}
					<p class="leise">Noch nicht platziert.</p>
				{/if}
				<button class="sekundaer klein" onclick={platzAendernBeginnen}>Platz ändern</button>
			{/if}
			{#if item.moebelLagerorte.length > 0}
				<p>
					Lagerorte dieses Möbelstücks:
					{#each item.moebelLagerorte as l, i (l.id)}{i > 0 ? ', ' : ''}<a href="/lagerorte/{l.id}">{l.name}</a>{/each}
				</p>
			{/if}
		</section>

		<section>
			<h2>Besitzer</h2>
			<ul class="zeilen">
				{#each item.besitzer as b (b.id)}
					<li>
						<span>{b.name}</span>
						<button class="sekundaer klein" disabled={laeuft} onclick={() => aendern('DELETE', `${pfad}/besitzer/${b.id}`)} aria-label="{b.name} als Besitzer entfernen">
							Entfernen
						</button>
					</li>
				{/each}
			</ul>
			{#if weiterePersonen.length > 0}
				<div class="knopfzeile">
					<select aria-label="Besitzer hinzufügen" bind:value={neuerBesitzer}>
						<option value="">Person wählen …</option>
						{#each weiterePersonen as p (p.id)}
							<option value={String(p.id)}>{p.name}</option>
						{/each}
					</select>
					<button class="sekundaer" disabled={laeuft || !neuerBesitzer} onclick={besitzerHinzufuegen}>Hinzufügen</button>
				</div>
			{/if}
		</section>

		<section>
			<button class="gefahr" onclick={itemLoeschen}>Item löschen</button>
		</section>
	{:else if !fehler}
		<p class="leise">Lädt …</p>
	{/if}
</main>
