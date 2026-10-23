<script lang="ts">
	import type { LagerortKnoten } from './api.ts';
	import LagerortBaum from './LagerortBaum.svelte';

	let { knoten }: { knoten: LagerortKnoten[] } = $props();
</script>

<ul class="baum">
	{#each knoten as k (k.id)}
		<li>
			<a href="/lagerorte/{k.id}">{k.name}</a>
			{#if k.moebel}<span class="leise"> · gehört zu {k.moebel.name}</span>{/if}
			{#if k.unter.length > 0}
				<LagerortBaum knoten={k.unter} />
			{/if}
		</li>
	{/each}
</ul>
