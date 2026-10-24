<script lang="ts">
	import type { Path } from './api.ts';

	// Path with links to the room and storage locations.
	let { path }: { path: Path } = $props();
</script>

<nav class="path" aria-label="Platz">
	<a href="/wohnungen">{path.apartment.name}</a> ›
	<a href="/raeume/{path.room.id}">{path.room.name}</a>
	{#each path.storageLocations as s (s.id)}
		› <a href="/lagerorte/{s.id}">{s.name}</a>
	{/each}
</nav>
