<script lang="ts">
	import { page } from '$app/state';
	import { api, boxName, errorText, unpackPath, type Box, type BoxChange, type Move, type PackedItem, type Place, type PlaceEntry } from '#lib/api.ts';
	import { useBottomBar } from '#lib/bottomBar.svelte.ts';
	import { loadMove } from '#lib/move.ts';
	import { loadPlaces } from '#lib/places.ts';
	import PlacePicker from '#lib/PlacePicker.svelte';

	// Unpack mode: choose a place of the new apartment once, then tap items or put the whole box there.
	// Items leave the list right away; if the server refuses, they come back with the error next to them.

	const id = $derived(Number(page.params.id));

	let box = $state<Box | null>(null);
	let items = $state<PackedItem[]>([]);
	let move = $state<Move | null>(null);
	let places = $state<PlaceEntry[]>([]);
	let place = $state<Place | null>(null);
	let loaded = $state(false);
	let error = $state('');
	let itemErrors = $state<Record<number, string>>({});
	let unpacked = $state(0);
	let pending = $state(0);
	// Position in the packing list, so that items coming back after an error keep their place.
	let order = new Map<number, number>();

	const inMove = $derived(!!box && !!move?.boxes.some((b) => b.id === box?.id));
	const chosenPath = $derived(places.find((p) => p.kind === place?.kind && p.id === place?.id)?.path ?? '');
	const nextBox = $derived.by(() => {
		if (!move || !box) return null;
		const others = move.boxes.filter((b) => b.id !== box?.id && b.count > 0).sort((a, b) => a.number - b.number);
		return others.find((b) => b.number > (box?.number ?? 0)) ?? others[0] ?? null;
	});

	async function load(boxID: number) {
		loaded = false;
		error = '';
		itemErrors = {};
		unpacked = 0;
		try {
			const [r, m, p] = await Promise.all([api<{ box: Box; items: PackedItem[] }>('GET', `/api/boxes/${boxID}`), loadMove(), loadPlaces()]);
			box = r.box;
			items = r.items;
			order = new Map(r.items.map((i, n) => [i.id, n]));
			move = m;
			places = p;
			place = r.box.targetRoom ? { kind: 'room', id: r.box.targetRoom.id } : null;
		} catch (e) {
			error = errorText(e);
		} finally {
			loaded = true;
		}
	}

	$effect(() => {
		load(id);
	});

	function restore(back: PackedItem[]) {
		const present = new Set(items.map((i) => i.id));
		items = [...items, ...back.filter((i) => !present.has(i.id))].sort((a, b) => (order.get(a.id) ?? 0) - (order.get(b.id) ?? 0));
	}

	/** Unpacks the given items, or all with null, to the chosen place. */
	async function unpack(ids: number[] | null) {
		error = '';
		if (!box || !place) {
			error = 'Bitte zuerst oben einen Platz wählen.';
			return;
		}
		const boxID = box.id;
		const taken = ids ? items.filter((i) => ids.includes(i.id)) : items;
		if (taken.length === 0) return;
		items = ids ? items.filter((i) => !ids.includes(i.id)) : [];
		const rest = { ...itemErrors };
		for (const i of taken) delete rest[i.id];
		itemErrors = rest;
		pending++;
		try {
			const r = await api<BoxChange>('POST', `/api/boxes/${boxID}/unpack`, {
				itemIds: ids,
				storageLocationId: place.kind === 'storageLocation' ? place.id : null,
				roomId: place.kind === 'room' ? place.id : null
			});
			if (box?.id !== boxID) return;
			box = r.box;
			unpacked += r.count;
		} catch (e) {
			if (box?.id !== boxID) return;
			restore(taken);
			if (ids) {
				const message = errorText(e);
				itemErrors = { ...itemErrors, ...Object.fromEntries(ids.map((i) => [i, message])) };
			} else {
				error = errorText(e);
			}
		} finally {
			pending--;
		}
	}

	$effect(() => useBottomBar(barContent));
</script>

{#snippet unpackAllButton(extraClass: string)}
	<button class="unpack-all {extraClass}" disabled={!place || items.length === 0} onclick={() => unpack(null)}>Alle hier einräumen</button>
{/snippet}

{#snippet barContent()}
	{#if inMove && items.length > 0}
		{@render unpackAllButton('')}
	{/if}
{/snippet}

<main class="page">
	<header class="header">
		<a class="button secondary" href="/boxen/{id}">← Box</a>
		{#if box}<h1>{boxName(box)}</h1>{/if}
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if box && !inMove}
		<p class="muted">Diese Box gehört zu keinem laufenden Umzug.</p>
	{:else if box}
		<div class="unpack-place">
			<PlacePicker
				bind:places
				bind:value={place}
				title="Einräumen in"
				noneAllowed={false}
				create
				hide={(p) => p.apartmentId !== move?.newApartment.id}
			/>
		</div>

		{#if unpacked > 0}
			<p class="muted" role="status">{unpacked} {unpacked === 1 ? 'Item' : 'Items'} eingeräumt.</p>
		{/if}

		{#if items.length > 0}
			<p class="muted">Antippen räumt ein Item {chosenPath ? `in „${chosenPath}“` : 'an den gewählten Platz'} ein.</p>
			<ul class="rows unpack-list">
				{#each items as item (item.id)}
					<li>
						<button class="pack" disabled={!place} onclick={() => unpack([item.id])}>
							<img src={item.thumbnailUrl} alt="" loading="lazy" />
							<span>
								{item.name}
								{#if item.previousPlace}<span class="muted small-text">kam aus {item.previousPlace}</span>{/if}
								{#if itemErrors[item.id]}<span class="error small-text" role="alert">{itemErrors[item.id]}</span>{/if}
							</span>
						</button>
					</li>
				{/each}
			</ul>
			<!-- On narrow screens the button sits in the bottom bar instead (barContent). -->
			{@render unpackAllButton('on-wide')}
		{:else if pending === 0}
			<div class="notice empty-box">
				<p><strong>Box ist leer.</strong></p>
				{#if nextBox}
					<a class="button" href={unpackPath(nextBox.id)}>Weiter mit {boxName(nextBox)} ({nextBox.count})</a>
				{:else}
					<p class="muted">Alle Boxen sind ausgepackt. <a href="/umzug">Zum Umzug</a></p>
				{/if}
			</div>
		{/if}
	{:else if loaded && !error}
		<p class="muted">Diese Box gibt es nicht.</p>
	{:else if !loaded}
		<p class="muted">Lädt …</p>
	{/if}
</main>
