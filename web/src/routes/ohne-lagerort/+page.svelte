<script lang="ts">
	import { api, errorText, type ItemListEntry, type ItemPage, type Place, type PlaceEntry } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';
	import { loadPlaces, placementBody } from '#lib/places.ts';
	import PlacePicker from '#lib/PlacePicker.svelte';

	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let places = $state<PlaceEntry[]>([]);
	let place = $state<Place | null>(null);
	let active = $state<number | null>(null);
	let error = $state('');
	let busy = $state(false);

	async function load() {
		try {
			const r = await api<ItemPage>('GET', '/api/items?unstored=1&perPage=100');
			items = r.items;
			more = r.more;
			places = await loadPlaces();
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function assign(item: ItemListEntry) {
		const p = place;
		if (!p) return;
		error = '';
		busy = true;
		try {
			await api('PUT', `/api/items/${item.id}/placement`, placementBody(p));
			// Only a storage location sorts an item in; directly in the room it stays „ohne Lagerort“.
			if (p.kind === 'storageLocation') items = items.filter((i) => i.id !== item.id);
			active = null;
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/">← Liste</a>
		<h1>Ohne Lagerort</h1>
	</header>

	<p class="muted">Diese Items haben noch keinen festen Platz in einem Lagerort. Der gewählte Platz bleibt für das nächste Item stehen.</p>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	<ul class="rows">
		{#each items as item (item.id)}
			<li class="assign">
				<a class="with-image" href="/items/{item.id}">
					<img src={item.thumbnailUrl} alt="" loading="lazy" />
					{item.name}
				</a>
				{#if active === item.id}
					<div class="full-width">
						<PlacePicker bind:places bind:value={place} noneAllowed={false} create />
						<div class="button-row">
							<button disabled={busy || !place} onclick={() => assign(item)}>Einsortieren</button>
							<button class="secondary" onclick={() => (active = null)}>Abbrechen</button>
						</div>
					</div>
				{:else}
					<button class="secondary small" onclick={() => (active = item.id)}>Einsortieren</button>
				{/if}
			</li>
		{:else}
			<li class="muted">Alles einsortiert.</li>
		{/each}
	</ul>
	{#if more}
		<p class="muted">Es gibt weitere; nach dem Einsortieren erscheinen sie hier.</p>
	{/if}
</main>
