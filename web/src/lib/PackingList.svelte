<script lang="ts">
	import type { ItemListEntry } from './api.ts';

	// Items with photo, name, origin room and target room.
	let { items }: { items: ItemListEntry[] } = $props();
</script>

<ul class="rows">
	{#each items as i (i.id)}
		<li>
			<a class="with-image" href="/items/{i.id}">
				<img src={i.thumbnailUrl} alt="" loading="lazy" />
				<span>
					{i.name}
					<span class="muted small-text">
						{i.move?.originRoom?.name ?? '–'} → {i.move?.targetRoom?.name ?? 'kein Zielraum'}
					</span>
				</span>
			</a>
		</li>
	{:else}
		<li class="muted">Leer.</li>
	{/each}
</ul>
