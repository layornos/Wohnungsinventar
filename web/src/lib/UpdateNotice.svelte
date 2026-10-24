<script lang="ts">
	// Small notice "Neue Version verfügbar" with "Neu laden" (appUpdate.svelte.ts). On narrow screens it floats
	// just above the bottom bar, without covering it.
	import { appUpdate, applyUpdate } from './appUpdate.svelte.ts';

	let { aboveBottomBar }: { aboveBottomBar: boolean } = $props();
</script>

<div role="status">
	{#if appUpdate.available}
		<div class="update-notice" class:above-bottom-bar={aboveBottomBar}>
			<span>Neue Version verfügbar</span>
			<button type="button" class="small" onclick={applyUpdate}>Neu laden</button>
		</div>
	{/if}
</div>
