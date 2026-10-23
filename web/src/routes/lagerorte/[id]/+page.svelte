<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, fehlerText, type ItemListenEintrag, type Lagerort, type Platz, type PlatzEintrag } from '#lib/api.ts';
	import ItemRaster from '#lib/ItemRaster.svelte';
	import ItemSuche from '#lib/ItemSuche.svelte';
	import { elternBody, ladePlaetze, wegfallBestaetigen } from '#lib/plaetze.ts';
	import PfadAnzeige from '#lib/PfadAnzeige.svelte';
	import PlatzAuswahl from '#lib/PlatzAuswahl.svelte';

	const id = $derived(Number(page.params.id));

	let lagerort = $state<Lagerort | null>(null);
	let items = $state<ItemListenEintrag[]>([]);
	let weitere = $state(false);
	let seite = 1;
	let plaetze = $state<PlatzEintrag[]>([]);
	let neuerLagerort = $state('');
	let name = $state<string | null>(null);
	let moebelWaehlen = $state(false);
	let verschieben = $state(false);
	let ziel = $state<Platz | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	// In sich selbst oder einen Unterlagerort lässt er sich nicht verschieben.
	const imEigenenBaum = (p: PlatzEintrag) => p.art === 'lagerort' && !!lagerort?.unterbaum.includes(p.id);

	async function laden(lagerortID: number) {
		fehler = '';
		try {
			lagerort = await api<Lagerort>('GET', `/api/lagerorte/${lagerortID}`);
			await itemsLaden(lagerortID, 1);
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	async function itemsLaden(lagerortID: number, s: number) {
		const r = await api<{ items: ItemListenEintrag[]; weitere: boolean }>('GET', `/api/items?lagerort=${lagerortID}&seite=${s}`);
		items = s === 1 ? r.items : [...items, ...r.items];
		weitere = r.weitere;
		seite = s;
	}

	async function mehrLaden() {
		try {
			await itemsLaden(id, seite + 1);
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	$effect(() => {
		laden(id);
	});

	async function ausfuehren(aktion: () => Promise<unknown>) {
		fehler = '';
		laeuft = true;
		try {
			await aktion();
			await laden(id);
			return true;
		} catch (e) {
			fehler = fehlerText(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	async function unterlagerortAnlegen(event: SubmitEvent) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', '/api/lagerorte', { name: neuerLagerort, elternId: id }))) neuerLagerort = '';
	}

	async function umbenennen(event: SubmitEvent) {
		event.preventDefault();
		const n = name;
		if (n !== null && (await ausfuehren(() => api('PUT', `/api/lagerorte/${id}`, { name: n })))) name = null;
	}

	async function verschiebenBeginnen() {
		try {
			plaetze = await ladePlaetze();
			ziel = null;
			verschieben = true;
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	async function verschiebenSpeichern() {
		const z = ziel;
		if (z && (await ausfuehren(() => api('PUT', `/api/lagerorte/${id}/eltern`, elternBody(z))))) verschieben = false;
	}

	async function moebelSetzen(itemId: number | null) {
		if (await ausfuehren(() => api('PUT', `/api/lagerorte/${id}/moebel`, { itemId }))) moebelWaehlen = false;
	}

	async function loeschen() {
		if (!lagerort) return;
		fehler = '';
		try {
			const frage = `Lagerort „${lagerort.name}“ samt Unterlagerorten löschen?`;
			if (!(await wegfallBestaetigen(`/api/lagerorte/${id}/wegfall`, frage))) return;
			const raum = lagerort.pfad.raum.id;
			await api('DELETE', `/api/lagerorte/${id}`);
			await goto(`/raeume/${raum}`, { replaceState: true });
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		{#if lagerort}
			<a class="knopf sekundaer" href="/raeume/{lagerort.pfad.raum.id}">← {lagerort.pfad.raum.name}</a>
		{:else}
			<a class="knopf sekundaer" href="/wohnungen">← Wohnungen</a>
		{/if}
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if lagerort}
		<PfadAnzeige pfad={lagerort.pfad} />
		{#if name !== null}
			<form class="inline" onsubmit={umbenennen}>
				<input aria-label="Neuer Name" bind:value={name} required />
				<button type="submit" disabled={laeuft}>Speichern</button>
				<button type="button" class="sekundaer" onclick={() => (name = null)}>Abbrechen</button>
			</form>
		{:else}
			<h1>{lagerort.name}</h1>
			<div class="knopfzeile">
				<button class="sekundaer klein" onclick={() => (name = lagerort?.name ?? '')}>Umbenennen</button>
				<button class="sekundaer klein" onclick={verschiebenBeginnen}>Verschieben</button>
				<button class="sekundaer klein" onclick={loeschen}>Löschen</button>
			</div>
		{/if}

		{#if verschieben}
			<section>
				<PlatzAuswahl {plaetze} bind:wert={ziel} titel="Verschieben nach" ohneErlaubt={false} ausblenden={imEigenenBaum} />
				<div class="knopfzeile">
					<button disabled={laeuft || !ziel} onclick={verschiebenSpeichern}>Verschieben</button>
					<button class="sekundaer" onclick={() => (verschieben = false)}>Abbrechen</button>
				</div>
			</section>
		{/if}

		<section>
			<h2>Möbelstück</h2>
			{#if lagerort.moebel}
				<p>Gehört zu <a href="/items/{lagerort.moebel.id}">{lagerort.moebel.name}</a>.</p>
				<button class="sekundaer klein" disabled={laeuft} onclick={() => moebelSetzen(null)}>Zuordnung lösen</button>
			{:else if moebelWaehlen}
				<ItemSuche gewaehlt={(item) => moebelSetzen(item.id)} />
				<button class="sekundaer klein" onclick={() => (moebelWaehlen = false)}>Abbrechen</button>
			{:else}
				<p class="leise">Gehört zu keinem Möbelstück.</p>
				<button class="sekundaer klein" onclick={() => (moebelWaehlen = true)}>Möbelstück zuordnen</button>
			{/if}
		</section>

		<section>
			<h2>Unterlagerorte</h2>
			<ul class="zeilen">
				{#each lagerort.unter as u (u.id)}
					<li><a href="/lagerorte/{u.id}">{u.name}</a></li>
				{:else}
					<li class="leise">Keine.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={unterlagerortAnlegen}>
				<input aria-label="Name des neuen Unterlagerorts" placeholder="z. B. obere Schublade" bind:value={neuerLagerort} required />
				<button type="submit" class="sekundaer" disabled={laeuft}>Anlegen</button>
			</form>
		</section>

		<section>
			<h2>Items darin</h2>
			{#if items.length > 0}
				<ItemRaster {items} />
				{#if weitere}
					<button class="sekundaer breit" onclick={mehrLaden}>Mehr laden</button>
				{/if}
			{:else}
				<p class="leise">Keine Items, auch nicht in Unterlagerorten.</p>
			{/if}
		</section>
	{/if}
</main>
