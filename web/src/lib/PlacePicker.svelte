<script lang="ts">
	import { api, errorText, pathText, type StorageLocation, type Place, type PlaceEntry } from './api.ts';
	import { parentBody } from './places.ts';

	// Searchable list of all rooms and storage locations as paths.
	// With `create` a storage location that wasn't found can be created under an existing place;
	// it is then added to `places` and chosen right away.
	let {
		places = $bindable(),
		value = $bindable(),
		title = 'Platz',
		noneAllowed = true,
		create = false,
		hide = () => false
	}: {
		places: PlaceEntry[];
		value: Place | null;
		title?: string;
		noneAllowed?: boolean;
		create?: boolean;
		hide?: (p: PlaceEntry) => boolean;
	} = $props();

	let search = $state('');
	let open = $state(false);
	// Name of the new storage location while its parent is being chosen.
	let newName = $state<string | null>(null);
	let error = $state('');
	let busy = $state(false);

	const chosen = $derived(places.find((p) => p.kind === value?.kind && p.id === value?.id));
	const results = $derived.by(() => {
		const s = search.trim().toLowerCase();
		return places.filter((p) => !hide(p) && p.path.toLowerCase().includes(s)).slice(0, 50);
	});

	function choose(p: Place | null) {
		value = p ? { kind: p.kind, id: p.id } : null;
		open = false;
		search = '';
		newName = null;
		error = '';
	}

	function startCreate() {
		newName = search.trim();
		search = '';
		error = '';
	}

	function cancelCreate() {
		search = newName ?? '';
		newName = null;
		error = '';
	}

	async function createIn(parent: PlaceEntry) {
		if (!newName) return;
		error = '';
		busy = true;
		try {
			const s = await api<StorageLocation>('POST', '/api/storage-locations', { name: newName, ...parentBody(parent) });
			const entry: PlaceEntry = { kind: 'storageLocation', id: s.id, apartmentId: s.path.apartment.id, path: pathText(s.path) };
			places = [...places, entry].sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
			choose(entry);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	function toggle() {
		open = !open;
		newName = null;
		error = '';
	}
</script>

<div class="place-picker">
	<span class="field-name">{title}</span>
	<button type="button" class="secondary select-button" onclick={toggle} aria-expanded={open}>
		{chosen?.path ?? (noneAllowed ? 'Kein Platz' : 'Bitte wählen …')}
	</button>
	{#if open && newName !== null}
		<p>„{newName}“ anlegen in …</p>
		<input type="search" placeholder="Raum oder Lagerort suchen …" aria-label="Anlegen in suchen" bind:value={search} />
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
		<ul class="results">
			{#each results as p (p.kind + p.id)}
				<li>
					<button type="button" class="secondary" disabled={busy} onclick={() => createIn(p)}>{p.path}</button>
				</li>
			{:else}
				<li class="muted">Nichts gefunden. Räume legst du unter „Wohnungen“ an.</li>
			{/each}
		</ul>
		<button type="button" class="secondary" onclick={cancelCreate}>Abbrechen</button>
	{:else if open}
		<input type="search" placeholder="Raum oder Lagerort suchen …" aria-label="{title} suchen" bind:value={search} />
		<ul class="results">
			{#if noneAllowed}
				<li><button type="button" class="secondary" onclick={() => choose(null)}>Kein Platz</button></li>
			{/if}
			{#each results as p (p.kind + p.id)}
				<li>
					<button type="button" class="secondary" class:active={p === chosen} onclick={() => choose(p)}>
						{p.path}{p.kind === 'room' ? ' (direkt im Raum)' : ''}
					</button>
				</li>
			{:else}
				{#if create && search.trim()}
					<li><button type="button" class="secondary" onclick={startCreate}>„{search.trim()}“ anlegen in …</button></li>
				{:else}
					<li class="muted">Nichts gefunden. Räume und Lagerorte legst du unter „Wohnungen“ an.</li>
				{/if}
			{/each}
		</ul>
	{/if}
</div>
