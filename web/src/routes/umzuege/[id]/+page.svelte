<script lang="ts">
	import { page } from '$app/state';
	import { api, boxName, datumKurz, fehlerText, type Umzug } from '#lib/api.ts';

	let umzug = $state<Umzug | null>(null);
	let fehler = $state('');

	$effect(() => {
		api<Umzug>('GET', `/api/umzuege/${page.params.id}`)
			.then((u) => (umzug = u))
			.catch((e) => (fehler = fehlerText(e)));
	});
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/umzug">← Umzug</a>
	</header>
	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}
	{#if umzug}
		<h1>{umzug.alteWohnung.name} → {umzug.neueWohnung.name}</h1>
		<p class="leise">
			Angelegt am {datumKurz(umzug.erstelltAm)}{#if umzug.abgeschlossenAm}, abgeschlossen am {datumKurz(umzug.abgeschlossenAm)}{/if}
		</p>
		<ul class="zeilen">
			{#each umzug.boxen as b (b.id)}
				<li><a href="/boxen/{b.id}">{boxName(b)}</a> <span class="leise">{b.anzahl} Items</span></li>
			{:else}
				<li class="leise">Keine Boxen.</li>
			{/each}
		</ul>
	{/if}
</main>
