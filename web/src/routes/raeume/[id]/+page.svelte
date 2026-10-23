<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, fehlerText, type ItemListenEintrag, type Raum } from '#lib/api.ts';
	import ItemRaster from '#lib/ItemRaster.svelte';
	import LagerortBaum from '#lib/LagerortBaum.svelte';

	const id = $derived(Number(page.params.id));

	let raum = $state<Raum | null>(null);
	let items = $state<ItemListenEintrag[]>([]);
	let weitere = $state(false);
	let neuerLagerort = $state('');
	let name = $state<string | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	async function laden(raumID: number) {
		fehler = '';
		try {
			raum = await api<Raum>('GET', `/api/raeume/${raumID}`);
			const r = await api<{ items: ItemListenEintrag[]; weitere: boolean }>('GET', `/api/items?raum=${raumID}&proSeite=100`);
			items = r.items;
			weitere = r.weitere;
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

	async function lagerortAnlegen(event: SubmitEvent) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', '/api/lagerorte', { name: neuerLagerort, raumId: id }))) neuerLagerort = '';
	}

	async function umbenennen(event: SubmitEvent) {
		event.preventDefault();
		const n = name;
		if (n !== null && (await ausfuehren(() => api('PUT', `/api/raeume/${id}`, { name: n })))) name = null;
	}

	async function loeschen() {
		if (!raum || !confirm(`Raum „${raum.name}“ löschen?`)) return;
		fehler = '';
		try {
			await api('DELETE', `/api/raeume/${id}`);
			await goto('/wohnungen', { replaceState: true });
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/wohnungen">← Wohnungen</a>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if raum}
		{#if name !== null}
			<form class="inline" onsubmit={umbenennen}>
				<input aria-label="Neuer Name" bind:value={name} required />
				<button type="submit" disabled={laeuft}>Speichern</button>
				<button type="button" class="sekundaer" onclick={() => (name = null)}>Abbrechen</button>
			</form>
		{:else}
			<p class="leise">{raum.wohnung.name}</p>
			<div class="kopf">
				<h1>{raum.name}</h1>
				<span class="knopfzeile">
					<button class="sekundaer klein" onclick={() => (name = raum?.name ?? '')}>Umbenennen</button>
					<button class="sekundaer klein" onclick={loeschen}>Löschen</button>
				</span>
			</div>
		{/if}

		<section>
			<h2>Lagerorte</h2>
			{#if raum.lagerorte.length > 0}
				<LagerortBaum knoten={raum.lagerorte} />
			{:else}
				<p class="leise">Noch keine Lagerorte.</p>
			{/if}
			<form class="inline" onsubmit={lagerortAnlegen}>
				<input aria-label="Name des neuen Lagerorts" placeholder="z. B. Regal links" bind:value={neuerLagerort} required />
				<button type="submit" class="sekundaer" disabled={laeuft}>Lagerort anlegen</button>
			</form>
		</section>

		<section>
			<h2>Items in diesem Raum</h2>
			{#if items.length > 0}
				<ItemRaster {items} />
				{#if weitere}<p class="leise">Es gibt weitere Items; die Liste zeigt die zuletzt geänderten.</p>{/if}
			{:else}
				<p class="leise">Keine Items.</p>
			{/if}
		</section>
	{/if}
</main>
