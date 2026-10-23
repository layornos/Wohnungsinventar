<script lang="ts">
	import { api, type ItemListenEintrag } from './api.ts';

	let { gewaehlt }: { gewaehlt: (item: ItemListenEintrag) => void } = $props();

	let suche = $state('');
	let treffer = $state<ItemListenEintrag[]>([]);

	$effect(() => {
		const q = suche.trim();
		if (!q) {
			treffer = [];
			return;
		}
		const t = setTimeout(async () => {
			try {
				treffer = (await api<{ items: ItemListenEintrag[] }>('GET', `/api/items?proSeite=10&q=${encodeURIComponent(q)}`)).items;
			} catch {
				treffer = [];
			}
		}, 250);
		return () => clearTimeout(t);
	});
</script>

<input type="search" placeholder="Item suchen, z. B. Schreibtisch" aria-label="Item suchen" bind:value={suche} />
<ul class="trefferliste">
	{#each treffer as item (item.id)}
		<li>
			<button type="button" class="sekundaer mit-bild" onclick={() => gewaehlt(item)}>
				<img src={item.vorschauUrl} alt="" />
				{item.name}
			</button>
		</li>
	{/each}
</ul>
