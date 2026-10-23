<script lang="ts">
	import { api, boxName, fehlerText, type Benannt, type Box, type ItemListenEintrag } from '#lib/api.ts';
	import Packliste from '#lib/Packliste.svelte';

	type Zielraum = { raum: Benannt; boxen: Box[]; items: ItemListenEintrag[] };

	let zielraeume = $state<Zielraum[]>([]);
	let fehler = $state('');

	api<{ zielraeume: Zielraum[] }>('GET', '/api/umzug/zielraeume')
		.then((r) => (zielraeume = r.zielraeume))
		.catch((e) => (fehler = fehlerText(e)));
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/umzug">← Umzug</a>
		<h1>Nach Zielraum</h1>
	</header>
	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}
	{#each zielraeume as z (z.raum.id)}
		<section>
			<h2>{z.raum.name}</h2>
			{#if z.boxen.length > 0}
				<p class="knopfzeile">
					{#each z.boxen as b (b.id)}<a class="marke" href="/boxen/{b.id}">{boxName(b)}</a>{/each}
				</p>
			{/if}
			<details>
				<summary>{z.items.length} Items</summary>
				<Packliste items={z.items} />
			</details>
		</section>
	{:else}
		<p class="leise">Die neue Wohnung hat noch keine Räume.</p>
	{/each}
</main>
