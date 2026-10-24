<script lang="ts">
	import { api, boxName, errorText, type Named, type Box, type ItemListEntry } from '#lib/api.ts';
	import PackingList from '#lib/PackingList.svelte';

	type TargetRoom = { room: Named; boxes: Box[]; items: ItemListEntry[] };

	let targetRooms = $state<TargetRoom[]>([]);
	let error = $state('');

	api<{ targetRooms: TargetRoom[] }>('GET', '/api/move/target-rooms')
		.then((r) => (targetRooms = r.targetRooms))
		.catch((e) => (error = errorText(e)));
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/umzug">← Umzug</a>
		<h1>Nach Zielraum</h1>
	</header>
	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	{#each targetRooms as t (t.room.id)}
		<section>
			<h2>{t.room.name}</h2>
			{#if t.boxes.length > 0}
				<p class="button-row">
					{#each t.boxes as b (b.id)}<a class="badge" href="/boxen/{b.id}">{boxName(b)}</a>{/each}
				</p>
			{/if}
			<details>
				<summary>{t.items.length} Items</summary>
				<PackingList items={t.items} />
			</details>
		</section>
	{:else}
		<p class="muted">Die neue Wohnung hat noch keine Räume.</p>
	{/each}
</main>
