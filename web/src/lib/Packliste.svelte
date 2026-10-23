<script lang="ts">
	import type { ItemListenEintrag } from './api.ts';

	// Items mit Foto, Name, Herkunftsraum und Zielraum.
	let { items }: { items: ItemListenEintrag[] } = $props();
</script>

<ul class="zeilen">
	{#each items as i (i.id)}
		<li>
			<a class="mit-bild" href="/items/{i.id}">
				<img src={i.vorschauUrl} alt="" loading="lazy" />
				<span>
					{i.name}
					<span class="leise klein-text">
						{i.umzug?.herkunftsraum?.name ?? '–'} → {i.umzug?.zielraum?.name ?? 'kein Zielraum'}
					</span>
				</span>
			</a>
		</li>
	{:else}
		<li class="leise">Leer.</li>
	{/each}
</ul>
