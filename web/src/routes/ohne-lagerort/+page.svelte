<script lang="ts">
	import { api, fehlerText, type ItemListenEintrag, type Platz, type PlatzEintrag } from '#lib/api.ts';
	import { ladePlaetze, platzBody } from '#lib/plaetze.ts';
	import PlatzAuswahl from '#lib/PlatzAuswahl.svelte';

	let items = $state<ItemListenEintrag[]>([]);
	let weitere = $state(false);
	let plaetze = $state<PlatzEintrag[]>([]);
	let platz = $state<Platz | null>(null);
	let aktiv = $state<number | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	async function laden() {
		try {
			const r = await api<{ items: ItemListenEintrag[]; weitere: boolean }>('GET', '/api/items?ohneLagerort=1&proSeite=100');
			items = r.items;
			weitere = r.weitere;
			plaetze = await ladePlaetze();
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	laden();

	async function einsortieren(item: ItemListenEintrag) {
		const p = platz;
		if (!p) return;
		fehler = '';
		laeuft = true;
		try {
			await api('PUT', `/api/items/${item.id}/platzierung`, platzBody(p));
			// Nur ein Lagerort sortiert ein; direkt im Raum bleibt das Item „ohne Lagerort“.
			if (p.art === 'lagerort') items = items.filter((i) => i.id !== item.id);
			aktiv = null;
		} catch (e) {
			fehler = fehlerText(e);
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Ohne Lagerort</h1>
	</header>

	<p class="leise">Diese Items haben noch keinen festen Platz in einem Lagerort. Der gewählte Platz bleibt für das nächste Item stehen.</p>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	<ul class="zeilen">
		{#each items as item (item.id)}
			<li class="einsortieren">
				<a class="mit-bild" href="/items/{item.id}">
					<img src={item.vorschauUrl} alt="" loading="lazy" />
					{item.name}
				</a>
				{#if aktiv === item.id}
					<div class="breit">
						<PlatzAuswahl {plaetze} bind:wert={platz} ohneErlaubt={false} />
						<div class="knopfzeile">
							<button disabled={laeuft || !platz} onclick={() => einsortieren(item)}>Einsortieren</button>
							<button class="sekundaer" onclick={() => (aktiv = null)}>Abbrechen</button>
						</div>
					</div>
				{:else}
					<button class="sekundaer klein" onclick={() => (aktiv = item.id)}>Einsortieren</button>
				{/if}
			</li>
		{:else}
			<li class="leise">Alles einsortiert.</li>
		{/each}
	</ul>
	{#if weitere}
		<p class="leise">Es gibt weitere; nach dem Einsortieren erscheinen sie hier.</p>
	{/if}
</main>
