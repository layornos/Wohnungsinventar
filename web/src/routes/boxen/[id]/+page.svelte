<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, boxName, fehlerText, type Box, type ItemListenEintrag } from '#lib/api.ts';
	import Packliste from '#lib/Packliste.svelte';
	import { ladeUmzug } from '#lib/umzug.ts';

	const id = $derived(Number(page.params.id));

	let box = $state<Box | null>(null);
	let items = $state<ItemListenEintrag[]>([]);
	let laufend = $state(false);
	let bezeichnung = $state<string | null>(null);
	let fehler = $state('');

	async function laden(boxID: number) {
		fehler = '';
		try {
			const r = await api<{ box: Box; items: ItemListenEintrag[] }>('GET', `/api/boxen/${boxID}`);
			box = r.box;
			items = r.items;
			laufend = !!(await ladeUmzug())?.boxen.some((b) => b.id === boxID);
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	$effect(() => {
		laden(id);
	});

	async function umbenennen(event: SubmitEvent) {
		event.preventDefault();
		try {
			await api('PUT', `/api/boxen/${id}`, { bezeichnung });
			bezeichnung = null;
			await laden(id);
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	async function loeschen() {
		if (!box || !confirm(`${boxName(box)} löschen?`)) return;
		try {
			await api('DELETE', `/api/boxen/${id}`);
			await goto('/umzug', { replaceState: true });
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/umzug">← Umzug</a>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if box}
		{#if bezeichnung !== null}
			<form class="inline" onsubmit={umbenennen}>
				<input aria-label="Bezeichnung" bind:value={bezeichnung} />
				<button type="submit">Speichern</button>
				<button type="button" class="sekundaer" onclick={() => (bezeichnung = null)}>Abbrechen</button>
			</form>
		{:else}
			<h1 class="boxnummer">Box {box.nummer}</h1>
			{#if box.bezeichnung}<p>{box.bezeichnung}</p>{/if}
			{#if laufend}
				<div class="knopfzeile">
					<a class="knopf" href="/umzug/packen?box={box.id}">In diese Box packen</a>
					<button class="sekundaer klein" onclick={() => (bezeichnung = box?.bezeichnung ?? '')}>Umbenennen</button>
					{#if items.length === 0}<button class="sekundaer klein" onclick={loeschen}>Löschen</button>{/if}
				</div>
			{:else}
				<p class="leise">Gehört zu einem abgeschlossenen Umzug.</p>
			{/if}
		{/if}

		<section>
			<h2>Packliste ({items.length})</h2>
			<Packliste {items} />
		</section>
	{/if}
</main>
