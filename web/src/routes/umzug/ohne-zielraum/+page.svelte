<script lang="ts">
	import { api, errorText, type Named, type ItemListEntry, type ItemPage } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';
	import { loadMove, loadMoveRooms } from '#lib/move.ts';

	let items = $state<ItemListEntry[]>([]);
	let more = $state(false);
	let rooms = $state<Named[]>([]);
	let targetRoom = $state<Record<number, string>>({});
	let error = $state('');

	async function load() {
		try {
			const m = await loadMove();
			if (m) rooms = (await loadMoveRooms(m)).new;
			({ items, more } = await api<ItemPage>('GET', '/api/items?withoutTargetRoom=1&perPage=100'));
		} catch (e) {
			error = errorText(e);
		}
	}
	load();

	async function set(item: ItemListEntry) {
		error = '';
		try {
			await api('PATCH', `/api/items/${item.id}/move`, { targetRoomId: Number(targetRoom[item.id]) });
			items = items.filter((i) => i.id !== item.id);
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/umzug">← Umzug</a>
		<h1>Ohne Zielraum</h1>
	</header>
	<p class="muted">Items der alten Wohnung, die noch nicht wissen, wohin.</p>
	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	<ul class="rows">
		{#each items as i (i.id)}
			<li>
				<a class="with-image" href="/items/{i.id}"><img src={i.thumbnailUrl} alt="" loading="lazy" />{i.name}</a>
				<span class="button-row">
					<select aria-label="Zielraum für {i.name}" bind:value={targetRoom[i.id]}>
						<option value="">Zielraum …</option>
						{#each rooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
					</select>
					<button class="small" disabled={!targetRoom[i.id]} onclick={() => set(i)}>Setzen</button>
				</span>
			</li>
		{:else}
			<li class="muted">Alle Items haben einen Zielraum.</li>
		{/each}
	</ul>
	{#if more}<p class="muted">Es gibt weitere; nach dem Setzen erscheinen sie hier.</p>{/if}
</main>
