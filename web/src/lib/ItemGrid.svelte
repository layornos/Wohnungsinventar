<script lang="ts">
	import { boxName, dispositionName, type ItemListEntry } from './api.ts';

	let { items }: { items: ItemListEntry[] } = $props();

	// Place and box in a muted line below the name.
	const where = (item: ItemListEntry) =>
		[item.place, item.move?.box ? boxName(item.move.box) : ''].filter(Boolean).join(' · ');
</script>

<!-- Depending on the theme a list (Minimal) or tiles (Neon, Soft); app.css alone decides. -->
<ul class="item-grid">
	{#each items as item (item.id)}
		<li>
			<a class="card" href="/items/{item.id}">
				<img src={item.thumbnailUrl} alt="" loading="lazy" />
				<span class="card-text">
					<span class="card-name">{item.name}</span>
					{#if where(item)}<span class="card-place">{where(item)}</span>{/if}
				</span>
				<span class="disposition" data-disposition={item.disposition ?? 'undecided'}>{dispositionName(item.disposition)}</span>
			</a>
		</li>
	{/each}
</ul>
