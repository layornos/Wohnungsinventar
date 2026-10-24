<script lang="ts">
	import {
		api,
		boxName,
		changeItems,
		errorText,
		DISPOSITIONS,
		type Box,
		type ItemPage,
		type ItemListEntry,
		type ItemsChange,
		type Person,
		type Place,
		type PlaceEntry
	} from '#lib/api.ts';
	import ItemGrid from '#lib/ItemGrid.svelte';
	import PlacePicker from '#lib/PlacePicker.svelte';
	import { useBottomBar } from '#lib/bottomBar.svelte.ts';
	import { loadPlaces, placementBody } from '#lib/places.ts';
	import { loadMove } from '#lib/move.ts';
	import { loadPersons } from '#lib/persons.ts';
	import { untrack } from 'svelte';
	import type { Snapshot } from './$types';

	let search = $state('');
	let owner = $state('');
	let disposition = $state('');
	// Place as "room:3" or "storageLocation:7", so a single select can do both.
	let place = $state('');
	let places = $state<PlaceEntry[]>([]);
	// During a move the list can be filtered by box.
	let box = $state('');
	let boxes = $state<Box[]>([]);
	let persons = $state<Person[]>([]);
	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let loading = $state(false);
	let error = $state('');
	let pageNumber = 1;

	// Selection mode: entered by a long press on an item or the "Auswählen" button; taps then toggle items.
	let selecting = $state(false);
	let selected = $state<number[]>([]);
	// Open while the place for the selected items is being chosen.
	let choosingPlace = $state(false);
	let newPlace = $state<Place | null>(null);
	let busy = $state(false);
	let notice = $state('');

	// Search and filters are kept when coming back from an item.
	export const snapshot: Snapshot<{ search: string; owner: string; disposition: string; place: string; box: string }> = {
		capture: () => ({ search, owner, disposition, place, box }),
		// Defaults, because a snapshot taken by an older version has other keys.
		restore: (s) => ({ search = '', owner = '', disposition = '', place = '', box = '' } = s)
	};

	loadPersons()
		.then((p) => (persons = p))
		.catch((e) => (error = errorText(e)));
	loadPlaces()
		.then((p) => (places = p))
		.catch((e) => (error = errorText(e)));
	loadMove()
		.then((m) => (boxes = m?.boxes ?? []))
		.catch((e) => (error = errorText(e)));

	function url(n: number) {
		const p = new URLSearchParams({ page: String(n) });
		if (search.trim()) p.set('q', search.trim());
		if (owner) p.set('owner', owner);
		if (disposition) p.set('disposition', disposition);
		const [kind, id] = place.split(':');
		if (id) p.set(kind, id);
		if (box) p.set('box', box);
		return '/api/items?' + p;
	}

	// Only the response to the latest request counts, in case of quick typing.
	let request = 0;
	async function load(fresh: boolean) {
		const nr = ++request;
		const n = fresh ? 1 : pageNumber + 1;
		loading = true;
		error = '';
		try {
			const r = await api<ItemPage>('GET', url(n));
			if (nr !== request) return;
			items = fresh ? r.items : [...items, ...r.items];
			more = r.more;
			pageNumber = n;
		} catch (e) {
			if (nr === request) error = errorText(e);
		} finally {
			if (nr === request) loading = false;
		}
	}

	$effect(() => {
		void [search, owner, disposition, place, box];
		const t = setTimeout(() => load(true), 250);
		return () => clearTimeout(t);
	});

	// Search and "Erfassen" move into the bottom bar on narrow screens; in selection mode the actions take their place.
	$effect(() => useBottomBar(selecting ? selectionActions : barContent));

	function startSelecting(id?: number) {
		selecting = true;
		notice = '';
		if (id !== undefined && !selected.includes(id)) selected = [...selected, id];
	}

	function stopSelecting() {
		selecting = false;
		selected = [];
		choosingPlace = false;
		newPlace = null;
	}

	function toggle(id: number) {
		selected = selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id];
	}

	// The notice disappears after a while by itself.
	$effect(() => {
		if (!notice) return;
		const t = setTimeout(() => (notice = ''), 4000);
		return () => clearTimeout(t);
	});

	async function apply(change: ItemsChange) {
		if (selected.length === 0 || busy) return;
		busy = true;
		error = '';
		try {
			const count = await changeItems(selected, change);
			stopSelecting();
			notice = count === 1 ? '1 Item geändert' : `${count} Items geändert`;
			await load(true);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	// The selects apply right away: choose the disposition, done.
	function applyDisposition(e: Event & { currentTarget: HTMLSelectElement }) {
		const value = e.currentTarget.value;
		e.currentTarget.value = '';
		if (value) apply({ disposition: value === 'undecided' ? null : (value as ItemsChange['disposition']) });
	}

	function applyOwner(e: Event & { currentTarget: HTMLSelectElement }) {
		const value = e.currentTarget.value;
		e.currentTarget.value = '';
		if (value) apply({ addOwnerId: Number(value) });
	}

	// Choosing a place in the picker applies it as well (only newPlace triggers this).
	$effect(() => {
		const p = newPlace;
		if (p) untrack(() => apply(placementBody(p)));
	});
</script>

{#snippet barContent()}
	<input type="search" placeholder="Suchen …" aria-label="Nach Name suchen" bind:value={search} />
	<a class="button add-button" href="/items/neu" aria-label="Item erfassen">+</a>
{/snippet}

<!-- Actions for the selected items: in the bottom bar on narrow screens, in the selection bar on wide ones. -->
{#snippet selectionActions()}
	<div class="selection-actions">
		<select aria-label="Verbleib setzen" disabled={busy || selected.length === 0} onchange={applyDisposition}>
			<option value="">Verbleib …</option>
			{#each DISPOSITIONS as d (d.value)}
				<option value={d.value}>{d.name}</option>
			{/each}
			<option value="undecided">Offen</option>
		</select>
		<select aria-label="Besitzer hinzufügen" disabled={busy || selected.length === 0} onchange={applyOwner}>
			<option value="">Besitzer + …</option>
			{#each persons as p (p.id)}
				<option value={String(p.id)}>{p.name}</option>
			{/each}
		</select>
		<button
			class="secondary"
			aria-expanded={choosingPlace}
			disabled={busy || selected.length === 0}
			onclick={() => (choosingPlace = !choosingPlace)}>Platz …</button
		>
	</div>
{/snippet}

<main class="page wide">
	<header class="header with-menu">
		<h1>Inventar</h1>
		{#if !selecting}
			<button class="secondary select-start" onclick={() => startSelecting()}>Auswählen</button>
		{/if}
	</header>

	<div class="filter">
		<!-- On narrow screens the search sits in the bottom bar instead (barContent). -->
		<input class="on-wide" type="search" placeholder="Suchen …" aria-label="Nach Name suchen" bind:value={search} />
		<select aria-label="Besitzer" bind:value={owner}>
			<option value="">Alle Besitzer</option>
			{#each persons as p (p.id)}
				<option value={String(p.id)}>{p.name}</option>
			{/each}
		</select>
		<select aria-label="Verbleib" bind:value={disposition}>
			<option value="">Jeder Verbleib</option>
			<option value="undecided">Offen</option>
			{#each DISPOSITIONS as d (d.value)}
				<option value={d.value}>{d.name}</option>
			{/each}
		</select>
		<select class="span-all" aria-label="Platz" bind:value={place}>
			<option value="">Überall</option>
			{#each places as p (p.kind + p.id)}
				<option value="{p.kind}:{p.id}">{p.path}</option>
			{/each}
		</select>
		{#if boxes.length > 0}
			<select class="span-all" aria-label="Box" bind:value={box}>
				<option value="">Jede Box</option>
				{#each boxes as b (b.id)}<option value={String(b.id)}>{boxName(b)}</option>{/each}
			</select>
		{/if}
	</div>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	{#if notice}
		<p class="notice" role="status">{notice}</p>
	{/if}

	{#if selecting}
		<div class="selection-bar">
			<div class="selection-row">
				<span class="selection-count">{selected.length} ausgewählt</span>
				<button class="secondary small" onclick={() => (selected = [...new Set([...selected, ...items.map((i) => i.id)])])}>Alle</button>
				<button class="secondary small" onclick={() => (selected = [])}>Keine</button>
				<button class="small" onclick={stopSelecting}>Fertig</button>
			</div>
			<div class="selection-wide">{@render selectionActions()}</div>
			{#if choosingPlace}
				<div class="selection-place">
					<PlacePicker bind:places bind:value={newPlace} title={selected.length === 1 ? 'Platz für 1 Item' : `Platz für ${selected.length} Items`} noneAllowed={false} create />
					<button class="secondary" onclick={() => (choosingPlace = false)}>Abbrechen</button>
				</div>
			{/if}
		</div>
	{/if}

	<ItemGrid {items} selected={selecting ? selected : null} ontoggle={toggle} onlongpress={(id) => (selecting ? toggle(id) : startSelecting(id))} />

	{#if !loading && items.length === 0 && !error}
		<p class="muted">Keine Items gefunden.</p>
	{/if}
	{#if more}
		<button class="secondary full-width" onclick={() => load(false)} disabled={loading}>Mehr laden</button>
	{/if}

	{#if !selecting}
		<a class="button floating on-wide" href="/items/neu">+ Erfassen</a>
	{/if}
</main>
