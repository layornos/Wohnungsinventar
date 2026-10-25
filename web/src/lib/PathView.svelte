<script lang="ts">
	import type { Path } from './api.ts';

	// Path with links to the room and storage locations. The space before › doesn't break, so a wrapped path never
	// starts a line with ›, and no space ends up inside a link.
	let { path }: { path: Path } = $props();

	const links = $derived([
		{ href: '/wohnungen', name: path.apartment.name },
		{ href: `/raeume/${path.room.id}`, name: path.room.name },
		...path.storageLocations.map((s) => ({ href: `/lagerorte/${s.id}`, name: s.name }))
	]);
</script>

<nav class="path" aria-label="Platz">
	{#each links as l, i (l.href)}{#if i > 0}&nbsp;›{' '}{/if}<a href={l.href}>{l.name}</a>{/each}
</nav>
