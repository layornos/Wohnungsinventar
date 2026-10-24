<script lang="ts">
	import { api, type ItemListEntry } from './api.ts';

	let { chosen }: { chosen: (item: ItemListEntry) => void } = $props();

	let search = $state('');
	let results = $state<ItemListEntry[]>([]);

	$effect(() => {
		const q = search.trim();
		if (!q) {
			results = [];
			return;
		}
		const t = setTimeout(async () => {
			try {
				results = (await api<{ items: ItemListEntry[] }>('GET', `/api/items?perPage=10&q=${encodeURIComponent(q)}`)).items;
			} catch {
				results = [];
			}
		}, 250);
		return () => clearTimeout(t);
	});
</script>

<input type="search" placeholder="Item suchen, z. B. Schreibtisch" aria-label="Item suchen" bind:value={search} />
<ul class="results">
	{#each results as item (item.id)}
		<li>
			<button type="button" class="secondary with-image" onclick={() => chosen(item)}>
				<img src={item.thumbnailUrl} alt="" />
				{item.name}
			</button>
		</li>
	{/each}
</ul>
