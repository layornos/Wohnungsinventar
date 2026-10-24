<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, errorText, type ItemListEntry, type ItemPage, type Room } from '#lib/api.ts';
	import ItemGrid from '#lib/ItemGrid.svelte';
	import MenuButton from '#lib/MenuButton.svelte';
	import StorageLocationTree from '#lib/StorageLocationTree.svelte';

	const id = $derived(Number(page.params.id));

	let room = $state<Room | null>(null);
	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let newStorageLocation = $state('');
	let name = $state<string | null>(null);
	let error = $state('');
	let busy = $state(false);

	async function load(roomID: number) {
		error = '';
		try {
			room = await api<Room>('GET', `/api/rooms/${roomID}`);
			const r = await api<ItemPage>('GET', `/api/items?room=${roomID}&perPage=100`);
			items = r.items;
			more = r.more;
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

	async function createStorageLocation(event: SubmitEvent) {
		event.preventDefault();
		if (await run(() => api('POST', '/api/storage-locations', { name: newStorageLocation, roomId: id }))) newStorageLocation = '';
	}

	async function rename(event: SubmitEvent) {
		event.preventDefault();
		const n = name;
		if (n !== null && (await run(() => api('PUT', `/api/rooms/${id}`, { name: n })))) name = null;
	}

	async function remove() {
		if (!room || !confirm(`Raum „${room.name}“ löschen?`)) return;
		error = '';
		try {
			await api('DELETE', `/api/rooms/${id}`);
			await goto('/wohnungen', { replaceState: true });
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/wohnungen">← Wohnungen</a>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if room}
		{#if name !== null}
			<form class="inline" onsubmit={rename}>
				<input aria-label="Neuer Name" bind:value={name} required />
				<button type="submit" disabled={busy}>Speichern</button>
				<button type="button" class="secondary" onclick={() => (name = null)}>Abbrechen</button>
			</form>
		{:else}
			<p class="muted">{room.apartment.name}</p>
			<div class="header">
				<h1>{room.name}</h1>
				<span class="button-row">
					<button class="secondary small" onclick={() => (name = room?.name ?? '')}>Umbenennen</button>
					<button class="secondary small" onclick={remove}>Löschen</button>
				</span>
			</div>
		{/if}

		<section>
			<h2>Lagerorte</h2>
			{#if room.storageLocations.length > 0}
				<StorageLocationTree nodes={room.storageLocations} />
			{:else}
				<p class="muted">Noch keine Lagerorte.</p>
			{/if}
			<form class="inline" onsubmit={createStorageLocation}>
				<input aria-label="Name des neuen Lagerorts" placeholder="z. B. Regal links" bind:value={newStorageLocation} required />
				<button type="submit" class="secondary" disabled={busy}>Lagerort anlegen</button>
			</form>
		</section>

		<section>
			<h2>Items in diesem Raum</h2>
			{#if items.length > 0}
				<ItemGrid {items} />
				{#if more}<p class="muted">Es gibt weitere Items; die Liste zeigt die zuletzt geänderten.</p>{/if}
			{:else}
				<p class="muted">Keine Items.</p>
			{/if}
		</section>
	{/if}
</main>
