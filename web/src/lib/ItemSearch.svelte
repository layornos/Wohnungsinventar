<script lang="ts">
	import { api, type ItemListEntry } from './api.ts';

	// clearOnChoose: after a tap the search starts over and keeps the focus, for choosing one item after another.
	let { chosen, clearOnChoose = false }: { chosen: (item: ItemListEntry) => void; clearOnChoose?: boolean } = $props();

	let search = $state('');
	let results = $state<ItemListEntry[]>([]);
	let input = $state<HTMLInputElement>();

	function choose(item: ItemListEntry) {
		chosen(item);
		if (clearOnChoose) {
			search = '';
			input?.focus();
		}
	}

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

<input type="search" placeholder="Item suchen, z. B. Schreibtisch" aria-label="Item suchen" bind:value={search} bind:this={input} />
<ul class="results">
	{#each results as item (item.id)}
		<li>
			<button type="button" class="secondary with-image" onclick={() => choose(item)}>
				<img src={item.thumbnailUrl} alt="" />
				{item.name}
			</button>
		</li>
	{/each}
</ul>
