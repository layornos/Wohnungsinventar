<script lang="ts">
	import { boxName, verbleibName, type ItemListenEintrag } from './api.ts';

	let { items }: { items: ItemListenEintrag[] } = $props();
</script>

<ul class="raster">
	{#each items as item (item.id)}
		<li>
			<a class="karte" href="/items/{item.id}">
				<img src={item.vorschauUrl} alt="" loading="lazy" />
				<span class="karte-name">{item.name}</span>
				<span class="marke" class:offen={item.verbleib === null}>{verbleibName(item.verbleib)}</span>
				{#if item.umzug?.box}<span class="marke box">{boxName(item.umzug.box)}</span>{/if}
			</a>
		</li>
	{/each}
</ul>
