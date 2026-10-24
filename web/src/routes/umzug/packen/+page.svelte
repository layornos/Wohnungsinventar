<script lang="ts">
	import { page } from '$app/state';
	import { api, boxName, errorText, type Named, type ItemListEntry, type ItemPage, type Move } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';
	import { loadMove, loadMoveRooms, type MoveDetailsChange } from '#lib/move.ts';

	let move = $state<Move | null>(null);
	let oldRooms = $state<Named[]>([]);
	let newRooms = $state<Named[]>([]);
	let box = $state(page.url.searchParams.get('box') ?? '');
	let origin = $state('');
	let search = $state('');
	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let selection = $state<number[]>([]);
	let targetRoom = $state('');
	let error = $state('');
	let busy = $state(false);

	const activeBox = $derived(move?.boxes.find((b) => String(b.id) === box));

	async function reloadMove() {
		try {
			move = await loadMove();
			if (move) ({ old: oldRooms, new: newRooms } = await loadMoveRooms(move));
		} catch (e) {
			error = errorText(e);
		}
	}
	reloadMove();

	async function loadItems() {
		const p = new URLSearchParams({ perPage: '100' });
		if (origin) p.set('originRoom', origin);
		if (search.trim()) p.set('q', search.trim());
		try {
			({ items, more } = await api<ItemPage>('GET', '/api/items?' + p));
		} catch (e) {
			error = errorText(e);
		}
	}

	$effect(() => {
		void [origin, search];
		const t = setTimeout(loadItems, 250);
		return () => clearTimeout(t);
	});

	async function run(action: () => Promise<unknown>) {
		error = '';
		busy = true;
		try {
			await action();
			await Promise.all([loadItems(), reloadMove()]);
			return true;
		} catch (e) {
			error = errorText(e);
			return false;
		} finally {
			busy = false;
		}
	}

	function pack(item: ItemListEntry) {
		if (!activeBox) {
			error = 'Bitte zuerst oben eine Box wählen.';
			return;
		}
		const out = item.move?.box?.id === activeBox.id;
		const change: MoveDetailsChange = { boxId: out ? null : activeBox.id };
		run(() => api('PATCH', `/api/items/${item.id}/move`, change));
	}

	function toggle(id: number) {
		selection = selection.includes(id) ? selection.filter((s) => s !== id) : [...selection, id];
	}

	async function bulk(change: MoveDetailsChange) {
		if (await run(() => api('POST', '/api/move/bulk', { itemIds: selection, ...change }))) selection = [];
	}

	async function createBox() {
		await run(async () => {
			const b = await api<{ id: number }>('POST', '/api/move/boxes', { label: '' });
			box = String(b.id);
		});
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/umzug">← Umzug</a>
		<h1>Packen</h1>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if move}
		<div class="pack-header">
			<label>
				Aktive Box
				<select bind:value={box}>
					<option value="">Keine</option>
					{#each move.boxes as b (b.id)}<option value={String(b.id)}>{boxName(b)} ({b.count})</option>{/each}
				</select>
			</label>
			<button class="secondary" disabled={busy} onclick={createBox}>Neue Box</button>
			{#if activeBox}
				<a class="button" href="/items/neu?box={activeBox.id}">+ Neu in {boxName(activeBox)}</a>
			{/if}
		</div>

		<div class="filter">
			<select aria-label="Herkunftsraum" bind:value={origin}>
				<option value="">Alle Räume</option>
				{#each oldRooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
			</select>
			<input type="search" placeholder="Suchen …" aria-label="Suchen" bind:value={search} />
		</div>

		<p class="muted">Antippen packt ein Item in die aktive Box oder nimmt es wieder heraus.</p>

		<ul class="rows">
			{#each items as item (item.id)}
				<li>
					<input type="checkbox" class="large" aria-label="{item.name} auswählen" checked={selection.includes(item.id)} onchange={() => toggle(item.id)} />
					<button class="pack" class:packed={!!activeBox && item.move?.box?.id === activeBox.id} disabled={busy} onclick={() => pack(item)}>
						<img src={item.thumbnailUrl} alt="" loading="lazy" />
						<span>
							{item.name}
							<span class="muted small-text">
								{item.move?.box ? boxName(item.move.box) : 'in keiner Box'} · {item.move?.targetRoom?.name ?? 'kein Zielraum'}
							</span>
						</span>
					</button>
				</li>
			{:else}
				<li class="muted">Keine Items.</li>
			{/each}
		</ul>
		{#if more}<p class="muted">Es gibt weitere Items; grenze mit Raum oder Suche ein.</p>{/if}

		{#if selection.length > 0}
			<div class="action-bar">
				<span>{selection.length} ausgewählt</span>
				<select aria-label="Zielraum" bind:value={targetRoom}>
					<option value="">Zielraum …</option>
					{#each newRooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
				</select>
				<button disabled={busy || !targetRoom} onclick={() => bulk({ targetRoomId: Number(targetRoom) })}>Zielraum setzen</button>
				{#if activeBox}
					<button class="secondary" disabled={busy} onclick={() => bulk({ boxId: activeBox.id })}>In {boxName(activeBox)}</button>
				{/if}
			</div>
		{/if}
	{:else}
		<p class="muted">Es läuft kein Umzug. <a href="/umzug">Umzug anlegen</a></p>
	{/if}
</main>
