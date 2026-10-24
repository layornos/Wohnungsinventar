<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, boxName, errorText, type Box, type ItemListEntry } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';
	import PackingList from '#lib/PackingList.svelte';
	import { loadMove } from '#lib/move.ts';

	const id = $derived(Number(page.params.id));

	let box = $state<Box | null>(null);
	let items = $state<ItemListEntry[]>([]);
	let active = $state(false);
	let label = $state<string | null>(null);
	let error = $state('');

	async function load(boxID: number) {
		error = '';
		try {
			const r = await api<{ box: Box; items: ItemListEntry[] }>('GET', `/api/boxes/${boxID}`);
			box = r.box;
			items = r.items;
			active = !!(await loadMove())?.boxes.some((b) => b.id === boxID);
		} catch (e) {
			error = errorText(e);
		}
	}

	$effect(() => {
		load(id);
	});

	async function rename(event: SubmitEvent) {
		event.preventDefault();
		try {
			await api('PUT', `/api/boxes/${id}`, { label });
			label = null;
			await load(id);
		} catch (e) {
			error = errorText(e);
		}
	}

	async function remove() {
		if (!box || !confirm(`${boxName(box)} löschen?`)) return;
		try {
			await api('DELETE', `/api/boxes/${id}`);
			await goto('/umzug', { replaceState: true });
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/umzug">← Umzug</a>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if box}
		{#if label !== null}
			<form class="inline" onsubmit={rename}>
				<input aria-label="Bezeichnung" bind:value={label} />
				<button type="submit">Speichern</button>
				<button type="button" class="secondary" onclick={() => (label = null)}>Abbrechen</button>
			</form>
		{:else}
			<h1 class="box-number">Box {box.number}</h1>
			{#if box.label}<p>{box.label}</p>{/if}
			{#if active}
				<div class="button-row">
					<a class="button" href="/umzug/packen?box={box.id}">In diese Box packen</a>
					<button class="secondary small" onclick={() => (label = box?.label ?? '')}>Umbenennen</button>
					{#if items.length === 0}<button class="secondary small" onclick={remove}>Löschen</button>{/if}
				</div>
			{:else}
				<p class="muted">Gehört zu einem abgeschlossenen Umzug.</p>
			{/if}
		{/if}

		<section>
			<h2>Packliste ({items.length})</h2>
			<PackingList {items} />
		</section>
	{/if}
</main>
