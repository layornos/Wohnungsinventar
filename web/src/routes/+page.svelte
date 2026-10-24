<script lang="ts">
	import { api, boxName, errorText, DISPOSITIONS, type Box, type ItemPage, type ItemListEntry, type Person, type PlaceEntry } from '#lib/api.ts';
	import ItemGrid from '#lib/ItemGrid.svelte';
	import { loadPlaces } from '#lib/places.ts';
	import { loadMove } from '#lib/move.ts';
	import { loadPersons } from '#lib/persons.ts';
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
</script>

<main class="page wide">
	<header class="header with-menu">
		<h1>Inventar</h1>
	</header>

	<div class="filter">
		<input type="search" placeholder="Suchen …" aria-label="Nach Name suchen" bind:value={search} />
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

	<ItemGrid {items} />

	{#if !loading && items.length === 0 && !error}
		<p class="muted">Keine Items gefunden.</p>
	{/if}
	{#if more}
		<button class="secondary full-width" onclick={() => load(false)} disabled={loading}>Mehr laden</button>
	{/if}

	<a class="button floating" href="/items/neu">+ Erfassen</a>
</main>
