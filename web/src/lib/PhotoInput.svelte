<script lang="ts">
	// Two ways to a photo: straight to the camera or the gallery (several at once).
	let { chosen, disabled = false }: { chosen: (files: File[]) => void; disabled?: boolean } = $props();

	let camera: HTMLInputElement;
	let gallery: HTMLInputElement;

	function take(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const files = [...(input.files ?? [])];
		input.value = '';
		if (files.length > 0) chosen(files);
	}
</script>

<div class="button-row">
	<button type="button" onclick={() => camera.click()} {disabled}>Foto aufnehmen</button>
	<button type="button" class="secondary" onclick={() => gallery.click()} {disabled}>Aus Galerie</button>
</div>
<input bind:this={camera} type="file" accept="image/*" capture="environment" hidden onchange={take} />
<input bind:this={gallery} type="file" accept="image/*" multiple hidden onchange={take} />
