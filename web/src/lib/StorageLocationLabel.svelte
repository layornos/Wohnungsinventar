<script lang="ts">
	// Printable label of a storage location: the path down to it, its name large, and the QR code of its page.
	// It uses the same sheet as the box label (label-sheet in app.css): black on white in every theme and,
	// when printing, a page of its own per label.
	import type { StorageLocationLabelData } from './storageLocationLabels.ts';

	let { label }: { label: StorageLocationLabelData } = $props();

	const parents = $derived(label.path.slice(0, -1));
	const name = $derived(label.path[label.path.length - 1]);
</script>

<article class="label-sheet storage-location-label">
	{#if parents.length > 0}<p class="label-path">{parents.join(' › ')} ›</p>{/if}
	<p class="label-location">{name}</p>
	<img class="label-qr" src="/api/storage-locations/{label.id}/qr" alt="QR-Code zum Lagerort {label.path.join(' › ')}" />
</article>
