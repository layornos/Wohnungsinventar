<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText, type ItemListEntry, type ItemPage, type StorageLocation, type Place, type PlaceEntry } from '#lib/api.ts';
	import ItemGrid from '#lib/ItemGrid.svelte';
	import ItemSearch from '#lib/ItemSearch.svelte';
	import { parentBody, loadPlaces, confirmRemoval } from '#lib/places.ts';
	import PathView from '#lib/PathView.svelte';
	import PlacePicker from '#lib/PlacePicker.svelte';

	const id = $derived(Number(page.params.id));

	let location = $state<StorageLocation | null>(null);
	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let pageNumber = 1;
	let places = $state<PlaceEntry[]>([]);
	let newStorageLocation = $state('');
	let name = $state<string | null>(null);
	let choosingFurniture = $state(false);
	let moving = $state(false);
	let target = $state<Place | null>(null);
	let error = $state('');
	let busy = $state(false);

	// It can't be moved into itself or one of its nested storage locations.
	const inOwnTree = (p: PlaceEntry) => p.kind === 'storageLocation' && !!location?.subtree.includes(p.id);

	async function load(locationID: number) {
		error = '';
		try {
			location = await api<StorageLocation>('GET', `/api/storage-locations/${locationID}`);
			await loadItems(locationID, 1);
		} catch (e) {
			error = errorText(e);
		}
	}

	async function loadItems(locationID: number, n: number) {
		const r = await api<ItemPage>('GET', `/api/items?storageLocation=${locationID}&page=${n}`);
		items = n === 1 ? r.items : [...items, ...r.items];
		more = r.more;
		pageNumber = n;
	}

	async function loadMore() {
		try {
			await loadItems(id, pageNumber + 1);
		} catch (e) {
			error = errorText(e);
		}
	}

	$effect(() => {
		load(id);
	});

	async function run(action: () => Promise<unknown>) {
		error = '';
		busy = true;
		try {
			await action();
			await load(id);
			return true;
		} catch (e) {
			error = errorText(e);
			return false;
		} finally {
			busy = false;
		}
	}

	async function createChild(event: SubmitEvent) {
		event.preventDefault();
		if (await run(() => api('POST', '/api/storage-locations', { name: newStorageLocation, parentId: id }))) newStorageLocation = '';
	}

	async function rename(event: SubmitEvent) {
		event.preventDefault();
		const n = name;
		if (n !== null && (await run(() => api('PUT', `/api/storage-locations/${id}`, { name: n })))) name = null;
	}

	async function startMoving() {
		try {
			places = await loadPlaces();
			target = null;
			moving = true;
		} catch (e) {
			error = errorText(e);
		}
	}

	async function saveMove() {
		const t = target;
		if (t && (await run(() => api('PUT', `/api/storage-locations/${id}/parent`, parentBody(t))))) moving = false;
	}

	async function setFurniture(itemId: number | null) {
		if (await run(() => api('PUT', `/api/storage-locations/${id}/furniture`, { itemId }))) choosingFurniture = false;
	}

	async function remove() {
		if (!location) return;
		error = '';
		try {
			const question = `Lagerort „${location.name}“ samt Unterlagerorten löschen?`;
			if (!(await confirmRemoval(`/api/storage-locations/${id}/removal`, question))) return;
			const room = location.path.room.id;
			await api('DELETE', `/api/storage-locations/${id}`);
			await goto(`/raeume/${room}`, { replaceState: true });
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page">
	<header class="header">
		{#if location}
			<a class="button secondary" href="/raeume/{location.path.room.id}">← {location.path.room.name}</a>
		{:else}
			<a class="button secondary" href="/wohnungen">← Wohnungen</a>
		{/if}
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if location}
		<PathView path={location.path} />
		{#if name !== null}
			<form class="inline" onsubmit={rename}>
				<input aria-label="Neuer Name" bind:value={name} required />
				<button type="submit" disabled={busy}>Speichern</button>
				<button type="button" class="secondary" onclick={() => (name = null)}>Abbrechen</button>
			</form>
		{:else}
			<h1>{location.name}</h1>
			<div class="button-row">
				<button class="secondary small" onclick={() => (name = location?.name ?? '')}>Umbenennen</button>
				<button class="secondary small" onclick={startMoving}>Verschieben</button>
				<button class="secondary small" onclick={remove}>Löschen</button>
			</div>
		{/if}

		{#if moving}
			<section>
				<PlacePicker {places} bind:value={target} title="Verschieben nach" noneAllowed={false} hide={inOwnTree} />
				<div class="button-row">
					<button disabled={busy || !target} onclick={saveMove}>Verschieben</button>
					<button class="secondary" onclick={() => (moving = false)}>Abbrechen</button>
				</div>
			</section>
		{/if}

		<section>
			<h2>Möbelstück</h2>
			{#if location.furniture}
				<p>Gehört zu <a href="/items/{location.furniture.id}">{location.furniture.name}</a>.</p>
				<button class="secondary small" disabled={busy} onclick={() => setFurniture(null)}>Zuordnung lösen</button>
			{:else if choosingFurniture}
				<ItemSearch chosen={(item) => setFurniture(item.id)} />
				<button class="secondary small" onclick={() => (choosingFurniture = false)}>Abbrechen</button>
			{:else}
				<p class="muted">Gehört zu keinem Möbelstück.</p>
				<button class="secondary small" onclick={() => (choosingFurniture = true)}>Möbelstück zuordnen</button>
			{/if}
		</section>

		<section>
			<h2>Unterlagerorte</h2>
			<ul class="rows">
				{#each location.children as c (c.id)}
					<li><a href="/lagerorte/{c.id}">{c.name}</a></li>
				{:else}
					<li class="muted">Keine.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={createChild}>
				<input aria-label="Name des neuen Unterlagerorts" placeholder="z. B. obere Schublade" bind:value={newStorageLocation} required />
				<button type="submit" class="secondary" disabled={busy}>Anlegen</button>
			</form>
		</section>

		<section>
			<h2>Items darin</h2>
			{#if items.length > 0}
				<ItemGrid {items} />
				{#if more}
					<button class="secondary full-width" onclick={loadMore}>Mehr laden</button>
				{/if}
			{:else}
				<p class="muted">Keine Items, auch nicht in Unterlagerorten.</p>
			{/if}
		</section>
	{/if}
</main>
