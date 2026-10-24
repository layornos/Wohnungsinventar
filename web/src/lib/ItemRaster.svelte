<script lang="ts">
	import { boxName, verbleibName, type ItemListenEintrag } from './api.ts';

	let { items }: { items: ItemListenEintrag[] } = $props();

	// Platz und Box in einer leisen Zeile unter dem Namen.
	const ort = (item: ItemListenEintrag) =>
		[item.platz, item.umzug?.box ? boxName(item.umzug.box) : ''].filter(Boolean).join(' · ');
</script>

<!-- Je nach Design eine Liste (Minimal) oder Kacheln (Neon); das entscheidet allein app.css. -->
<ul class="raster">
	{#each items as item (item.id)}
		<li>
			<a class="karte" href="/items/{item.id}">
				<img src={item.vorschauUrl} alt="" loading="lazy" />
				<span class="karte-text">
					<span class="karte-name">{item.name}</span>
					{#if ort(item)}<span class="karte-ort">{ort(item)}</span>{/if}
				</span>
				<span class="verbleib" data-verbleib={item.verbleib ?? 'offen'}>{verbleibName(item.verbleib)}</span>
			</a>
		</li>
	{/each}
</ul>
