<script lang="ts">
	import { api, fehlerText, VERBLEIBE, zurAnmeldung, type ItemListenEintrag, type Person, type PlatzEintrag } from '#lib/api.ts';
	import ItemRaster from '#lib/ItemRaster.svelte';
	import { ladePlaetze } from '#lib/plaetze.ts';
	import { ladePersonen } from '#lib/personen.ts';
	import { abmelden } from '#lib/sitzung.svelte.ts';
	import type { Snapshot } from './$types';

	let suche = $state('');
	let besitzer = $state('');
	let verbleib = $state('');
	// Platz als "raum:3" oder "lagerort:7", damit ein einziges Auswahlfeld beides kann.
	let platz = $state('');
	let plaetze = $state<PlatzEintrag[]>([]);
	let personen = $state<Person[]>([]);
	let items = $state<ItemListenEintrag[]>([]);
	let weitere = $state(false);
	let laedt = $state(false);
	let fehler = $state('');
	let seite = 1;

	// Suche und Filter bleiben erhalten, wenn man von einem Item zurückkommt.
	export const snapshot: Snapshot<{ suche: string; besitzer: string; verbleib: string; platz: string }> = {
		capture: () => ({ suche, besitzer, verbleib, platz }),
		restore: (s) => ({ suche, besitzer, verbleib, platz } = s)
	};

	ladePersonen()
		.then((p) => (personen = p))
		.catch((e) => (fehler = fehlerText(e)));
	ladePlaetze()
		.then((p) => (plaetze = p))
		.catch((e) => (fehler = fehlerText(e)));

	function adresse(s: number) {
		const p = new URLSearchParams({ seite: String(s) });
		if (suche.trim()) p.set('q', suche.trim());
		if (besitzer) p.set('besitzer', besitzer);
		if (verbleib) p.set('verbleib', verbleib);
		const [art, id] = platz.split(':');
		if (id) p.set(art, id);
		return '/api/items?' + p;
	}

	// Nur die Antwort der jüngsten Anfrage zählt, falls schnell hintereinander getippt wird.
	let anfrage = 0;
	async function laden(neu: boolean) {
		const nr = ++anfrage;
		const s = neu ? 1 : seite + 1;
		laedt = true;
		fehler = '';
		try {
			const r = await api<{ items: ItemListenEintrag[]; weitere: boolean }>('GET', adresse(s));
			if (nr !== anfrage) return;
			items = neu ? r.items : [...items, ...r.items];
			weitere = r.weitere;
			seite = s;
		} catch (e) {
			if (nr === anfrage) fehler = fehlerText(e);
		} finally {
			if (nr === anfrage) laedt = false;
		}
	}

	$effect(() => {
		void [suche, besitzer, verbleib, platz];
		const t = setTimeout(() => laden(true), 250);
		return () => clearTimeout(t);
	});

	async function abmeldenUndWeiter() {
		fehler = '';
		try {
			await abmelden();
		} catch {
			// Ohne Antwort des Servers bleibt die Sitzung dort bestehen; nicht so tun, als wäre man abgemeldet.
			fehler = 'Abmelden nicht möglich. Ist der Server erreichbar?';
			return;
		}
		await zurAnmeldung();
	}
</script>

<main class="seite">
	<header class="kopf">
		<h1>Inventar</h1>
		<nav class="knopfzeile">
			<a class="knopf sekundaer" href="/wohnungen">Wohnungen</a>
			<a class="knopf sekundaer" href="/ohne-lagerort">Ohne Lagerort</a>
			<a class="knopf sekundaer" href="/personen">Personen</a>
			<button class="sekundaer" onclick={abmeldenUndWeiter}>Abmelden</button>
		</nav>
	</header>

	<div class="filter">
		<input type="search" placeholder="Suchen …" aria-label="Nach Name suchen" bind:value={suche} />
		<select aria-label="Besitzer" bind:value={besitzer}>
			<option value="">Alle Besitzer</option>
			{#each personen as p (p.id)}
				<option value={String(p.id)}>{p.name}</option>
			{/each}
		</select>
		<select aria-label="Verbleib" bind:value={verbleib}>
			<option value="">Jeder Verbleib</option>
			<option value="offen">Offen</option>
			{#each VERBLEIBE as v (v.wert)}
				<option value={v.wert}>{v.name}</option>
			{/each}
		</select>
		<select class="ganz" aria-label="Platz" bind:value={platz}>
			<option value="">Überall</option>
			{#each plaetze as p (p.art + p.id)}
				<option value="{p.art}:{p.id}">{p.pfad}</option>
			{/each}
		</select>
	</div>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	<ItemRaster {items} />

	{#if !laedt && items.length === 0 && !fehler}
		<p class="leise">Keine Items gefunden.</p>
	{/if}
	{#if weitere}
		<button class="sekundaer breit" onclick={() => laden(false)} disabled={laedt}>Mehr laden</button>
	{/if}

	<a class="knopf schwebend" href="/items/neu">+ Erfassen</a>
</main>
