<script lang="ts">
	// Printable label of a box: very large number, label, target room and the QR code of the unpack link.
	// Black on white in every theme; when printing, every label gets a page of its own (app.css, "Print").
	import type { BoxSummary } from './api.ts';

	let { box }: { box: BoxSummary } = $props();
</script>

<article class="box-label-sheet">
	<p class="label-number">{box.number}</p>
	{#if box.label}<p class="label-name">{box.label}</p>{/if}
	<p class="label-target">→ {box.targetRoom?.name ?? '________'}</p>
	<img class="label-qr" src="/api/boxes/{box.id}/qr" alt="QR-Code zum Auspacken von Box {box.number}" />
</article>
