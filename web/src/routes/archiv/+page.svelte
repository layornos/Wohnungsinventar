<script lang="ts">
	// Archive (spec 06): sold and given away items, most recently handed over first, with search and filters.
	// The start page links here with ?q= when its search finds nothing.
	import { page } from '$app/state';
	import { api, errorText, type ItemListEntry, type ItemPage, type Person } from '#lib/api.ts';
	import ItemGrid from '#lib/ItemGrid.svelte';
	import { loadPersons } from '#lib/persons.ts';

	let search = $state(page.url.searchParams.get('q') ?? '');
	let owner = $state('');
	let outcome = $state('');
	let from = $state('');
	let to = $state('');
	let persons = $state<Person[]>([]);
	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let loading = $state(false);
	let error = $state('');
	let pageNumber = 1;

	loadPersons()
		.then((p) => (persons = p))
		.catch((e) => (error = errorText(e)));

	function url(n: number) {
		const p = new URLSearchParams({ page: String(n) });
		if (search.trim()) p.set('q', search.trim());
		if (owner) p.set('owner', owner);
		if (outcome) p.set('outcome', outcome);
		if (from) p.set('from', from);
		if (to) p.set('to', to);
		return '/api/archive?' + p;
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
		void [search, owner, outcome, from, to];
		const t = setTimeout(() => load(true), 250);
		return () => clearTimeout(t);
	});
</script>

<main class="page wide">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Archiv</h1>
	</header>

	<div class="filter">
		<input class="span-all" type="search" placeholder="Im Archiv suchen …" aria-label="Im Archiv nach Name suchen" bind:value={search} />
		<select aria-label="Besitzer" bind:value={owner}>
			<option value="">Alle Besitzer</option>
			{#each persons as p (p.id)}<option value={String(p.id)}>{p.name}</option>{/each}
		</select>
		<select aria-label="Verkauft oder verschenkt" bind:value={outcome}>
			<option value="">Verkauft und verschenkt</option>
			<option value="sold">Verkauft</option>
			<option value="givenAway">Verschenkt</option>
		</select>
		<label>
			Von
			<input type="date" bind:value={from} />
		</label>
		<label>
			Bis
			<input type="date" bind:value={to} />
		</label>
	</div>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	<ItemGrid {items} />

	{#if !loading && items.length === 0 && !error}
		<p class="muted">Nichts im Archiv gefunden.</p>
	{/if}
	{#if more}
		<button class="secondary full-width" onclick={() => load(false)} disabled={loading}>Mehr laden</button>
	{/if}
</main>
