<script lang="ts">
	import type { Pfad } from './api.ts';

	// Pfad mit Links zu Raum und Lagerorten.
	let { pfad }: { pfad: Pfad } = $props();
</script>

<nav class="pfad" aria-label="Platz">
	<a href="/wohnungen">{pfad.wohnung.name}</a> ›
	<a href="/raeume/{pfad.raum.id}">{pfad.raum.name}</a>
	{#each pfad.lagerorte as l (l.id)}
		› <a href="/lagerorte/{l.id}">{l.name}</a>
	{/each}
</nav>
