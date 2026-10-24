<script lang="ts">
	import type { StorageLocationNode } from './api.ts';
	import StorageLocationTree from './StorageLocationTree.svelte';

	let { nodes }: { nodes: StorageLocationNode[] } = $props();
</script>

<ul class="tree">
	{#each nodes as n (n.id)}
		<li>
			<a href="/lagerorte/{n.id}">{n.name}</a>
			{#if n.furniture}<span class="muted"> · gehört zu {n.furniture.name}</span>{/if}
			{#if n.children.length > 0}
				<StorageLocationTree nodes={n.children} />
			{/if}
		</li>
	{/each}
</ul>
