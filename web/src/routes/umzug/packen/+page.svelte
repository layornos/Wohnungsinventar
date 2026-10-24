<script lang="ts">
	import { page } from '$app/state';
	import {
		api,
		boxName,
		boxTitle,
		errorText,
		type Box,
		type BoxChange,
		type Named,
		type ItemListEntry,
		type ItemPage,
		type Move,
		type Room,
		type StorageLocationNode
	} from '#lib/api.ts';
	import { lastPackOriginRoom, loadMove, loadMoveRooms, rememberPackOriginRoom, type MoveDetailsChange } from '#lib/move.ts';
	import NewBoxDialog from '#lib/NewBoxDialog.svelte';

	/** A storage location of the origin room with the IDs of the listed items in it or nested below it. */
	type Group = { id: number; path: string; itemIds: Set<number> };

	let move = $state<Move | null>(null);
	let oldRooms = $state<Named[]>([]);
	let newRooms = $state<Named[]>([]);
	let box = $state(page.url.searchParams.get('box') ?? '');
	const remembered = lastPackOriginRoom();
	let origin = $state(remembered ? String(remembered) : '');
	let search = $state('');
	let items = $state<ItemListEntry[]>([]);
	let groups = $state<Group[]>([]);
	let more = $state(false);
	let selection = $state<number[]>([]);
	let targetRoom = $state('');
	let error = $state('');
	let notice = $state('');
	let busy = $state(false);
	let creatingBox = $state(false);

	const activeBox = $derived(move?.boxes.find((b) => String(b.id) === box));

	// Every item belongs to the innermost storage location it is listed in; groups are in tree order, so that is
	// the last match. Items outside all storage locations of the room end up in rest.
	const grouped = $derived.by(() => {
		const owner = new Map<number, number>();
		for (const g of groups) for (const id of g.itemIds) owner.set(id, g.id);
		return {
			groups: groups
				.filter((g) => g.itemIds.size > 0)
				.map((g) => ({ ...g, items: items.filter((i) => owner.get(i.id) === g.id) })),
			rest: items.filter((i) => !owner.has(i.id))
		};
	});

	async function reloadMove() {
		try {
			move = await loadMove();
			if (move) {
				({ old: oldRooms, new: newRooms } = await loadMoveRooms(move));
				// The remembered room may belong to an earlier move or be gone.
				if (origin && !oldRooms.some((r) => String(r.id) === origin)) chooseOrigin('');
			}
		} catch (e) {
			error = errorText(e);
		}
	}
	reloadMove();

	function chooseOrigin(value: string) {
		origin = value;
		rememberPackOriginRoom(Number(value));
	}

	function flatten(nodes: StorageLocationNode[], prefix: string): { id: number; path: string }[] {
		return nodes.flatMap((n) => {
			const path = prefix ? `${prefix} › ${n.name}` : n.name;
			return [{ id: n.id, path }, ...flatten(n.children, path)];
		});
	}

	function itemQuery(extra: Record<string, string>) {
		const p = new URLSearchParams({ perPage: '100', ...extra });
		if (search.trim()) p.set('q', search.trim());
		return '/api/items?' + p;
	}

	/** The storage locations of the origin room with the items in each of them (including nested ones). */
	async function loadGroups(roomId: number): Promise<Group[]> {
		const room = await api<Room>('GET', `/api/rooms/${roomId}`);
		const locations = flatten(room.storageLocations, '');
		return Promise.all(
			locations.map(async (l) => {
				const r = await api<ItemPage>('GET', itemQuery({ storageLocation: String(l.id) }));
				return { ...l, itemIds: new Set(r.items.map((i) => i.id)) };
			})
		);
	}

	// Answers of an older search arriving late must not overwrite newer ones.
	let loadCounter = 0;

	async function loadItems() {
		const n = ++loadCounter;
		const roomId = Number(origin);
		try {
			const [list, g] = await Promise.all([
				api<ItemPage>('GET', itemQuery(roomId ? { originRoom: String(roomId) } : {})),
				roomId ? loadGroups(roomId) : Promise.resolve([])
			]);
			if (n !== loadCounter) return;
			({ items, more } = list);
			groups = g;
		} catch (e) {
			if (n === loadCounter) error = errorText(e);
		}
	}

	$effect(() => {
		void [origin, search];
		const t = setTimeout(loadItems, 250);
		return () => clearTimeout(t);
	});

	async function run(action: () => Promise<unknown>) {
		error = '';
		notice = '';
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

	function needBox(): Box | null {
		if (!activeBox) error = 'Bitte zuerst oben eine Box wählen.';
		return activeBox ?? null;
	}

	function pack(item: ItemListEntry) {
		const b = needBox();
		if (!b) return;
		const out = item.move?.box?.id === b.id;
		const change: MoveDetailsChange = { boxId: out ? null : b.id };
		run(() => api('PATCH', `/api/items/${item.id}/move`, change));
	}

	function packAll(group: Group) {
		const b = needBox();
		if (!b) return;
		run(async () => {
			const r = await api<BoxChange>('POST', `/api/boxes/${b.id}/pack`, { storageLocationId: group.id });
			notice = `${r.count} ${r.count === 1 ? 'Item' : 'Items'} aus „${group.path}“ in ${boxName(r.box)} gepackt.`;
		});
	}

	function toggle(id: number) {
		selection = selection.includes(id) ? selection.filter((s) => s !== id) : [...selection, id];
	}

	async function bulk(change: MoveDetailsChange) {
		if (await run(() => api('POST', '/api/move/bulk', { itemIds: selection, ...change }))) selection = [];
	}

	async function boxCreated(b: Box | null) {
		creatingBox = false;
		if (!b) return;
		box = String(b.id);
		await reloadMove();
	}
</script>

{#snippet itemRow(item: ItemListEntry)}
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
{/snippet}

<main class="page">
	<header class="header">
		<a class="button secondary" href="/umzug">← Umzug</a>
		<h1>Packen</h1>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	{#if notice}
		<p class="notice" role="status">{notice}</p>
	{/if}

	{#if move}
		<div class="pack-header">
			<label>
				Aktive Box
				<select bind:value={box}>
					<option value="">Keine</option>
					{#each move.boxes as b (b.id)}<option value={String(b.id)}>{boxTitle(b)} ({b.count})</option>{/each}
				</select>
			</label>
			<button class="secondary" disabled={busy} onclick={() => (creatingBox = true)}>Neue Box</button>
			{#if activeBox}
				<a class="button" href="/items/neu?box={activeBox.id}">+ Neu in {boxName(activeBox)}</a>
			{/if}
		</div>

		<div class="filter">
			<select aria-label="Herkunftsraum" value={origin} onchange={(e) => chooseOrigin(e.currentTarget.value)}>
				<option value="">Alle Räume</option>
				{#each oldRooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
			</select>
			<input type="search" placeholder="Suchen …" aria-label="Suchen" bind:value={search} />
		</div>

		<p class="muted">
			Antippen packt ein Item in die aktive Box oder nimmt es wieder heraus.
			{#if !origin}Wähle einen Raum, um ganze Lagerorte auf einmal einzupacken.{/if}
		</p>

		{#if origin}
			{#each grouped.groups as g (g.id)}
				<section class="pack-group">
					<div class="pack-group-header">
						<h2>{g.path}</h2>
						<button class="secondary small" disabled={busy || !activeBox} onclick={() => packAll(g)}>Alles einpacken</button>
					</div>
					<ul class="rows">
						{#each g.items as item (item.id)}
							{@render itemRow(item)}
						{:else}
							<li class="muted">Nur in Unterlagerorten.</li>
						{/each}
					</ul>
				</section>
			{/each}
			{#if grouped.rest.length > 0 || grouped.groups.length === 0}
				<section class="pack-group">
					{#if grouped.groups.length > 0}<h2>Ohne Lagerort oder woanders</h2>{/if}
					<ul class="rows">
						{#each grouped.rest as item (item.id)}
							{@render itemRow(item)}
						{:else}
							<li class="muted">Keine Items.</li>
						{/each}
					</ul>
				</section>
			{/if}
		{:else}
			<ul class="rows">
				{#each items as item (item.id)}
					{@render itemRow(item)}
				{:else}
					<li class="muted">Keine Items.</li>
				{/each}
			</ul>
		{/if}
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

		{#if creatingBox}
			<NewBoxDialog rooms={newRooms} done={boxCreated} />
		{/if}
	{:else}
		<p class="muted">Es läuft kein Umzug. <a href="/umzug">Umzug anlegen</a></p>
	{/if}
</main>
