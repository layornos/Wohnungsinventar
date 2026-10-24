<script lang="ts">
	// Two ways to a photo: straight to the camera or the gallery (several at once).
	// Photos are shrunk before they are handed on (shrinkPhoto.ts), so uploads stay small.
	import { shrinkPhotos } from './shrinkPhoto.ts';

	let { chosen, disabled = false }: { chosen: (files: File[]) => void; disabled?: boolean } = $props();

	let camera: HTMLInputElement;
	let gallery: HTMLInputElement;
	let cameraButton: HTMLButtonElement;
	let shrinking = $state(false);

	/** Opens the camera; only works from a user gesture such as a tap. */
	export function openCamera() {
		camera.click();
	}

	/** Focuses the camera button, so the next photo is one tap (or Enter) away. */
	export function focusCamera() {
		cameraButton.focus({ preventScroll: true });
	}

	async function take(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const files = [...(input.files ?? [])];
		input.value = '';
		if (files.length === 0) return;
		shrinking = true;
		try {
			chosen(await shrinkPhotos(files));
		} finally {
			shrinking = false;
		}
	}
</script>

<div class="button-row">
	<button type="button" bind:this={cameraButton} onclick={openCamera} disabled={disabled || shrinking}>Foto aufnehmen</button>
	<button type="button" class="secondary" onclick={() => gallery.click()} disabled={disabled || shrinking}>Aus Galerie</button>
</div>
{#if shrinking}<p class="muted" role="status">Foto wird verkleinert …</p>{/if}
<input bind:this={camera} type="file" accept="image/*" capture="environment" hidden onchange={take} />
<input bind:this={gallery} type="file" accept="image/*" multiple hidden onchange={take} />
